package crushdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// realDataDir resolves the real database directory for the env-gated tests:
// CRUSH_DATA_REAL_DATA_DIR when set, else ../.crush when it holds a database.
// The bool reports whether real data is available at all.
func realDataDir(t *testing.T) (string, bool) {
	t.Helper()

	if dataDir := os.Getenv("CRUSH_DATA_REAL_DATA_DIR"); dataDir != "" {
		return dataDir, true
	}

	candidate, err := filepath.Abs(filepath.Join("..", ".crush"))
	if err != nil {
		return "", false
	}

	if _, statErr := os.Stat(filepath.Join(candidate, DBName)); statErr != nil {
		return "", false
	}

	return candidate, true
}

// partCensus is the result of scanning raw parts entries on disk: how often
// each discriminator appears, how many entries were seen, how many rows could
// not be parsed as arrays at all, and whether the byte budget stopped the
// scan before the window ended.
type partCensus struct {
	kinds       map[string]int
	entries     int
	unparseable int
	bytes       int64
	truncated   bool
}

// knownPartKinds is the complete discriminator set Crush v0.92.0 writes
// (the decode table in parts.go plus the two attachment pass-throughs).
// Anything else in a real database means upstream added a part type.
var knownPartKinds = map[string]bool{
	partText:         true,
	partReasoning:    true,
	partToolCall:     true,
	partToolResult:   true,
	partFinish:       true,
	partShellCommand: true,
	partImageURL:     true,
	partBinary:       true,
}

// unknownKinds returns the discriminators this library does not know,
// sorted. Typeless entries (empty type, null data — Crush has always written
// them and the decoder skips them by design) are not drift and stay excluded.
func (c partCensus) unknownKinds() []string {
	unknown := make([]string, 0, len(c.kinds))
	for kind := range c.kinds {
		if kind != "" && !knownPartKinds[kind] {
			unknown = append(unknown, kind)
		}
	}

	sort.Strings(unknown)

	return unknown
}

// The census walks the most recent partCensusWindow messages per database
// (every message when CRUSH_DATA_CENSUS_FULL=1) and stops early once
// partCensusByteBudget of raw parts JSON has been read, newest rows first.
// Two ceilings because rows vary by four orders of magnitude: the row window
// bounds the rowid-index walk, while the byte budget bounds payload I/O —
// registry databases carry recent messages of up to multiple megabytes each
// (measured 2026-09-08: a 2000-row tail of one project database held
// 100–200 MB of parts), so rows alone cannot cap the cost.
const (
	partCensusWindow     = 50_000
	partCensusByteBudget = 256 << 20
)

// censusPartDiscriminators aggregates the raw `{type,data}` discriminators of
// a database's most recent parts entries, parsing in Go rather than through
// SQLite's json_each (whose per-element rendering in the transpiled driver is
// the slow path) and decoding payloads as json.RawMessage, so cost tracks
// bytes read, not payload depth. Rows that are not valid JSON are counted,
// not failed — corruption is a different problem than drift, and
// DB.Messages already degrades such rows to nil Parts. messages.rowid is
// insertion order, so the rowid bound and the DESC scan select the newest
// rows without walking the rest of the table.
func censusPartDiscriminators(ctx context.Context, handle *sql.DB) (partCensus, error) {
	census := partCensus{kinds: make(map[string]int)}

	var maxRowID int64
	if err := handle.QueryRowContext(ctx, "SELECT COALESCE(MAX(rowid), 0) FROM messages").
		Scan(&maxRowID); err != nil {
		return partCensus{}, fmt.Errorf("find newest message row: %w", err)
	}

	// rowids start at 1, so a lower bound of 0 selects every row in full mode.
	lowerBound := int64(0)
	full := os.Getenv("CRUSH_DATA_CENSUS_FULL") == "1"

	if !full {
		lowerBound = maxRowID - partCensusWindow
	}

	rows, err := handle.QueryContext(ctx, `
		SELECT parts FROM messages
		WHERE rowid > ? AND parts IS NOT NULL AND parts != '[]'
		ORDER BY rowid DESC`, lowerBound)
	if err != nil {
		return partCensus{}, fmt.Errorf("census part kinds: %w", err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return partCensus{}, fmt.Errorf("scan parts row: %w", err)
		}

		census.bytes += int64(len(raw))

		var entries []rawPart
		if err := json.Unmarshal(raw, &entries); err != nil {
			census.unparseable++
		} else {
			for _, entry := range entries {
				census.kinds[entry.Type]++
				census.entries++
			}
		}

		if !full && census.bytes >= partCensusByteBudget {
			census.truncated = true

			break
		}
	}

	if err := rows.Err(); err != nil {
		return partCensus{}, fmt.Errorf("census part kinds: %w", err)
	}

	return census, nil
}

// sortedHistogram renders a histogram deterministically for logs.
func sortedHistogram(counts map[string]int) string {
	kinds := make([]string, 0, len(counts))
	for kind := range counts {
		kinds = append(kinds, kind)
	}

	sort.Strings(kinds)

	rendered := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		rendered = append(rendered, fmt.Sprintf("%s=%d", kind, counts[kind]))
	}

	return "{" + strings.Join(rendered, " ") + "}"
}

// sweepTotals accumulates what one real-data sweep saw, for the final log.
type sweepTotals struct {
	messages      int
	parts         int
	graphNodes    int
	readFilePaths int
	todoLists     int
	partKinds     map[string]int
}

// sweepRealSession exercises every per-session read API against one real
// session and cross-checks the two message shapes against each other.
func sweepRealSession(t *testing.T, db *DB, session Session, totals *sweepTotals) {
	t.Helper()

	ctx := context.Background()

	if _, err := db.Session(ctx, session.ID); err != nil {
		t.Fatalf("Session(%s): %v", session.ID, err)
	}

	rows, err := db.Messages(ctx, session.ID)
	if err != nil {
		t.Fatalf("Messages(%s): %v", session.ID, err)
	}

	sliceParts := 0

	for _, message := range rows {
		sliceParts += len(message.Parts)

		for _, part := range message.Parts {
			totals.partKinds[decodedPartKind(part)]++
		}
	}

	streamed, iterParts := 0, 0

	for message, err := range db.IterMessages(ctx, session.ID) {
		if err != nil {
			t.Fatalf("IterMessages(%s): %v", session.ID, err)
		}

		streamed++
		iterParts += len(message.Parts)
	}

	if streamed != len(rows) || iterParts != sliceParts {
		t.Fatalf(
			"IterMessages(%s) streamed %d messages / %d parts, Messages returned %d / %d — the two shapes disagree",
			session.ID, streamed, iterParts, len(rows), sliceParts,
		)
	}

	graph, err := db.AgentGraph(ctx, session.ID)
	if err != nil {
		t.Fatalf("AgentGraph(%s): %v", session.ID, err)
	}

	paths, err := db.ReadFiles(ctx, session.ID)
	if err != nil {
		t.Fatalf("ReadFiles(%s): %v", session.ID, err)
	}

	if session.Todos != nil {
		if _, err := DecodeTodos(session.Todos); err != nil {
			t.Fatalf("DecodeTodos(%s): %v", session.ID, err)
		}
	}

	totals.messages += len(rows)
	totals.parts += sliceParts
	totals.graphNodes += len(graph.Nodes)
	totals.readFilePaths += len(paths)

	if session.Todos != nil {
		totals.todoLists++
	}
}

// TestAllAPIOnRealDatabase sweeps every public read API across a real
// database when one is present, proving Messages, IterMessages, Stats,
// AgentGraph, ReadFiles, and todos decoding against live data instead of
// fixtures — the ad-hoc verification sweep that verification sessions kept
// rebuilding by hand. Skipped when no local database exists (set
// CRUSH_DATA_REAL_DATA_DIR to point at one) and under -short.
//
// Stats runs day-filtered on purpose: an all-time aggregation DISTINCTs
// every message row ever written, which on a production-sized database
// (700k+ messages) under a live Crush writer starves for minutes. The day
// filter exercises the same SQL paths at bounded cost.
func TestAllAPIOnRealDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("real-data sweep (slow) in -short mode")
	}

	t.Parallel()

	dataDir, ok := realDataDir(t)
	if !ok {
		t.Skip("no real data dir (set CRUSH_DATA_REAL_DATA_DIR to run against one)")
	}

	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open real database: %v", err)
	}

	defer func() { _ = db.Close() }()

	ctx := context.Background()

	sessions, err := db.Sessions(ctx, SessionFilter{Limit: 20})
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}

	var totals sweepTotals

	totals.partKinds = make(map[string]int)

	for _, session := range sessions {
		sweepRealSession(t, db, session, &totals)
	}

	if _, err := db.Stats(ctx, StatsFilter{Day: sessions[0].CreatedAt}); err != nil {
		t.Fatalf("Stats(day): %v", err)
	}

	if missing := db.Schema().MissingColumns(); len(missing) > 0 {
		t.Logf("real database missing columns (reads degrade): %v", missing)
	}

	t.Logf(
		"real database at %s: %d sessions, %d messages, %d parts, part kinds %s, %d graph nodes, %d read-file paths, %d todo lists",
		dataDir,
		len(sessions),
		totals.messages,
		totals.parts,
		sortedHistogram(totals.partKinds),
		totals.graphNodes,
		totals.readFilePaths,
		totals.todoLists,
	)
}

// decodedPartKind names a decoded part for the sweep histogram. UnknownPart
// keeps its discriminator in the name so pass-through attachments and
// corrupt payloads stay distinguishable from genuinely new part types.
func decodedPartKind(part Part) string {
	switch typed := part.(type) {
	case TextPart:
		return partText
	case ReasoningPart:
		return partReasoning
	case ToolCallPart:
		return partToolCall
	case ToolResultPart:
		return partToolResult
	case FinishPart:
		return partFinish
	case ShellCommandPart:
		return partShellCommand
	case UnknownPart:
		return "unknown:" + typed.Type
	default:
		return "unknown"
	}
}

// TestPartDiscriminatorsCensusShape is the parts-envelope drift tripwire, the
// TestDecodeTodosCensusShape equivalent for the `{type,data}` envelope: it
// scans the raw parts entries of a real database's most recent messages —
// before decode, because a new upstream discriminator would decode as
// UnknownPart without a peep, which is exactly the silent-drift hole this
// closes — and fails when a discriminator outside the known set appears.
// Skipped without real data and under -short.
func TestPartDiscriminatorsCensusShape(t *testing.T) {
	if testing.Short() {
		t.Skip("real-data sweep (slow) in -short mode")
	}

	t.Parallel()

	dataDir, ok := realDataDir(t)
	if !ok {
		t.Skip("no real data dir (set CRUSH_DATA_REAL_DATA_DIR to run against one)")
	}

	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open real database: %v", err)
	}

	defer func() { _ = db.Close() }()

	census, err := censusPartDiscriminators(context.Background(), db.handle)
	if err != nil {
		t.Fatalf("census parts: %v", err)
	}

	if unknown := census.unknownKinds(); len(unknown) > 0 {
		t.Fatalf(
			"unknown part discriminators %v in %s (histogram %s) — Crush added part types this library does not decode; extend decodePart and knownPartKinds",
			unknown,
			dataDir,
			sortedHistogram(census.kinds),
		)
	}

	t.Logf(
		"parts census at %s: %d entries, kinds %s, %d unparseable rows, %d bytes, complete=%t",
		dataDir,
		census.entries,
		sortedHistogram(census.kinds),
		census.unparseable,
		census.bytes,
		!census.truncated,
	)
}

// TestPartDiscriminatorsCensusRegistry runs the census across every database
// in a real registry: set CRUSH_DATA_REAL_REGISTRY to the global data
// directory holding projects.json. Each database contributes its most
// recent messages (see partCensusWindow; CRUSH_DATA_CENSUS_FULL=1 scans
// everything), so even the local registry's multi-gigabyte database finishes
// in seconds. Intended for upstream-verification sessions (the AGENTS.md
// cadence), not the standard real-data pass. Fails on any unknown
// discriminator.
func TestPartDiscriminatorsCensusRegistry(t *testing.T) {
	if testing.Short() {
		t.Skip("registry census (slow) in -short mode")
	}

	t.Parallel()

	globalDir := os.Getenv("CRUSH_DATA_REAL_REGISTRY")
	if globalDir == "" {
		t.Skip("registry census (heavy); set CRUSH_DATA_REAL_REGISTRY to the global data dir to run it")
	}

	ctx := context.Background()

	projects, err := DiscoverProjects(ctx, DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	aggregate := partCensus{kinds: make(map[string]int)}
	databases := 0

	for _, project := range projects {
		db, err := Open(project.DataDir)
		if err != nil {
			t.Logf("skipping %s: %v", project.DataDir, err)

			continue
		}

		census, err := censusPartDiscriminators(ctx, db.handle)
		_ = db.Close()

		if err != nil {
			t.Fatalf("census %s: %v", project.DataDir, err)
		}

		databases++
		aggregate.entries += census.entries
		aggregate.unparseable += census.unparseable
		aggregate.bytes += census.bytes
		aggregate.truncated = aggregate.truncated || census.truncated

		for kind, count := range census.kinds {
			aggregate.kinds[kind] += count
		}
	}

	if unknown := aggregate.unknownKinds(); len(unknown) > 0 {
		t.Fatalf(
			"unknown part discriminators %v across %d databases — Crush added part types this library does not decode; extend decodePart and knownPartKinds",
			unknown,
			databases,
		)
	}

	t.Logf(
		"registry census at %s: %d databases, %d entries, kinds %s, %d unparseable rows, %d bytes, complete=%t",
		globalDir,
		databases,
		aggregate.entries,
		sortedHistogram(aggregate.kinds),
		aggregate.unparseable,
		aggregate.bytes,
		!aggregate.truncated,
	)
}

// TestPartDiscriminatorsCensusDetectsUnknown pins the tripwire's failure mode
// on a fixture, so CI proves fail-loud behavior on every platform without
// real data: a synthetic discriminator must surface in unknownKinds while
// known and typeless entries must not.
func TestPartDiscriminatorsCensusDetectsUnknown(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(handle *sql.DB) {
		insertSession(t, handle, "session", "", "Session", 1, fixtureBase, fixtureBase)
		insertMessage(t, handle, "message", "session", "assistant",
			`[{"type":"text","data":{"text":"hi"}},{"type":"hologram","data":{"fov":90}},{"data":{"no":"type"}}]`,
			fixtureModel, "", fixtureBase)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	census, err := censusPartDiscriminators(context.Background(), db.handle)
	if err != nil {
		t.Fatalf("census parts: %v", err)
	}

	if got := census.kinds["text"]; got != 1 {
		t.Fatalf("text count = %d, want 1 (histogram %s)", got, sortedHistogram(census.kinds))
	}

	if got := census.kinds[""]; got != 1 {
		t.Fatalf("typeless count = %d, want 1 (histogram %s)", got, sortedHistogram(census.kinds))
	}

	unknown := census.unknownKinds()
	if len(unknown) != 1 || unknown[0] != "hologram" {
		t.Fatalf("unknownKinds() = %v, want [hologram]", unknown)
	}
}
