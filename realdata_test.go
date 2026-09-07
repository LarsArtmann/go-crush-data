package crushdata

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// sweepTotals accumulates what one real-data sweep saw, for the final log.
type sweepTotals struct {
	messages      int
	parts         int
	graphNodes    int
	readFilePaths int
	todoLists     int
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
// rebuilding by hand. Skipped when no local database exists; set
// CRUSH_DATA_REAL_DATA_DIR to point at one.
func TestAllAPIOnRealDatabase(t *testing.T) {
	t.Parallel()

	dataDir := os.Getenv("CRUSH_DATA_REAL_DATA_DIR")
	if dataDir == "" {
		candidate, err := filepath.Abs(filepath.Join("..", ".crush"))
		if err != nil {
			t.Skip("no real data dir")
		}

		if _, statErr := os.Stat(filepath.Join(candidate, DBName)); statErr != nil {
			t.Skip("no real data dir (set CRUSH_DATA_REAL_DATA_DIR to run against one)")
		}

		dataDir = candidate
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

	for _, session := range sessions {
		sweepRealSession(t, db, session, &totals)
	}

	if _, err := db.Stats(ctx, StatsFilter{}); err != nil {
		t.Fatalf("Stats(all time): %v", err)
	}

	if !sessions[0].CreatedAt.IsZero() {
		if _, err := db.Stats(ctx, StatsFilter{Day: sessions[0].CreatedAt}); err != nil {
			t.Fatalf("Stats(day): %v", err)
		}
	}

	if missing := db.Schema().MissingColumns(); len(missing) > 0 {
		t.Logf("real database missing columns (reads degrade): %v", missing)
	}

	t.Logf(
		"real database at %s: %d sessions, %d messages, %d parts, %d graph nodes, %d read-file paths, %d todo lists",
		dataDir,
		len(sessions),
		totals.messages,
		totals.parts,
		totals.graphNodes,
		totals.readFilePaths,
		totals.todoLists,
	)
}
