package crushdata

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

// TestMessagesOrdersByInsertionOrder pins the rowid ordering contract:
// messages come back in the order Crush wrote them. The fixture half
// covers cross-second ordering; the tie half covers same-second groups,
// where the contract matters most — Crush message IDs are UUIDv4
// (random), so (created_at, id) would be deterministic but arbitrary
// within a second and could sort a tool_result before its tool_call.
// The tie IDs below sort in the exact reverse of insertion order, so
// the old (created_at, id) ordering would fail this test.
func TestMessagesOrdersByInsertionOrder(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	messages, err := db.Messages(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(messages) != 11 {
		t.Fatalf("messages = %d, want 11", len(messages))
	}

	// m_future and m_user_2 share a created_at second; m_future was
	// inserted later, so insertion order puts it last even though its ID
	// sorts first.
	if messages[0].ID != "m_user" || messages[len(messages)-1].ID != "m_future" {
		t.Fatalf("order broken: first=%s last=%s", messages[0].ID, messages[len(messages)-1].ID)
	}

	tieIDs := []string{
		"ffffffff-ffff-4fff-bfff-ffffffffffff",
		"88888888-8888-4888-b888-888888888888",
		"11111111-1111-4111-b111-111111111111",
	}

	dataDir := t.TempDir()
	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(handle *sql.DB) {
		insertSession(t, handle, "s1", "", "Session", len(tieIDs), fixtureBase, fixtureBase)

		for _, id := range tieIDs {
			insertMessage(t, handle, id, "s1", "assistant", `[]`, "", "", fixtureBase)
		}
	})

	tieDB, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = tieDB.Close() }()

	tied, err := tieDB.Messages(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(tied) != len(tieIDs) {
		t.Fatalf("same-second messages = %d, want %d", len(tied), len(tieIDs))
	}

	for i, id := range tieIDs {
		if tied[i].ID != id {
			t.Fatalf("same-second messages[%d].ID = %s, want %s (insertion order, not ID order)", i, tied[i].ID, id)
		}
	}
}

func TestMessagesFields(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	messages, err := db.Messages(context.Background(), "fixture-root")
	if err != nil {
		t.Fatal(err)
	}

	byID := map[string]Message{}
	for _, message := range messages {
		byID[message.ID] = message
	}

	assistant := byID["m_assistant_1"]
	if assistant.Role != RoleAssistant {
		t.Fatalf("role = %q, want assistant", assistant.Role)
	}

	if assistant.Model != "minimax/minimax-m3" {
		t.Fatalf("model = %q", assistant.Model)
	}

	if assistant.SessionID != "fixture-root" {
		t.Fatalf("SessionID = %q", assistant.SessionID)
	}

	if assistant.CreatedAt.Unix() != fixtureBase+1 {
		t.Fatalf("CreatedAt = %v", assistant.CreatedAt)
	}

	user := byID["m_user"]
	if user.Role != RoleUser || user.Model != "" {
		t.Fatalf("user message = %+v, want empty model", user)
	}

	if _, hasCall := partOf[ToolCallPart](assistant.Parts); !hasCall {
		t.Fatalf("assistant parts = %#v, want a ToolCallPart", assistant.Parts)
	}

	if _, hasFinish := partOf[FinishPart](user.Parts); !hasFinish {
		t.Fatalf("user parts = %#v, want a FinishPart", user.Parts)
	}
}

// TestMessagesFinishedAtPopulated pins the finished_at scan path: a non-NULL
// column yields the completion time, NULL yields the zero time.
func TestMessagesFinishedAtPopulated(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "s", "", "Session", 2, fixtureBase, fixtureBase+10)
		insertMessage(t, db, "m-done", "s", "assistant", "[]", fixtureModel, "", fixtureBase+1)
		insertMessage(t, db, "m-open", "s", "assistant", "[]", fixtureModel, "", fixtureBase+2)

		if _, err := db.ExecContext(
			context.Background(),
			"UPDATE messages SET finished_at = ? WHERE id = 'm-done'",
			fixtureBase+7,
		); err != nil {
			t.Fatal(err)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	messages, err := db.Messages(context.Background(), "s")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	byID := map[string]Message{}
	for _, message := range messages {
		byID[message.ID] = message
	}

	done := byID["m-done"]
	if done.FinishedAt.Unix() != fixtureBase+7 {
		t.Fatalf("FinishedAt = %v, want %d", done.FinishedAt, fixtureBase+7)
	}

	// insertMessage writes updated_at = created_at; the scan must surface it.
	if done.UpdatedAt.Unix() != fixtureBase+1 {
		t.Fatalf("UpdatedAt = %v, want %d", done.UpdatedAt, fixtureBase+1)
	}

	if open := byID["m-open"]; !open.FinishedAt.IsZero() {
		t.Fatalf("FinishedAt = %v, want zero time for NULL column", open.FinishedAt)
	}
}

// partOf returns the first part of type T.
func partOf[T Part](parts []Part) (T, bool) {
	for _, part := range parts {
		if typed, ok := part.(T); ok {
			return typed, true
		}
	}

	var zero T

	return zero, false
}

func TestMessagesToleratesMalformedParts(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "s1", "", "Session", 2, fixtureBase, fixtureBase+2)
		insertMessage(t, db, "m1", "s1", "assistant", `[{"type":"text","data":{"text":"ok"}}]`, "m", "p", fixtureBase)
		insertMessage(t, db, "m2", "s1", "assistant", `{"broken"`, "m", "p", fixtureBase+1)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	messages, err := db.Messages(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Messages: %v (one malformed row must not fail the read)", err)
	}

	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}

	if messages[0].Parts == nil {
		t.Fatal("first message parts = nil, want decoded")
	}

	if messages[1].Parts != nil {
		t.Fatalf("malformed message parts = %#v, want nil", messages[1].Parts)
	}
}

// TestMessagesKeepsSiblingsAroundCorruptedPart pins the tolerant decode
// end to end: one corrupted tool_call among valid parts degrades to
// UnknownPart carrying the raw payload while its well-formed siblings
// survive, instead of the whole message's parts being dropped to nil.
func TestMessagesKeepsSiblingsAroundCorruptedPart(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "s1", "", "Session", 1, fixtureBase, fixtureBase+1)
		insertMessage(t, db, "m1", "s1", "assistant", `[
			{"type":"text","data":{"text":"before"}},
			{"type":"tool_call","data":"flat"},
			{"type":"tool_result","data":{"tool_call_id":"c1","name":"read","content":"ok"}}
		]`, "m", "p", fixtureBase)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	messages, err := db.Messages(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}

	parts := messages[0].Parts
	if len(parts) != 3 {
		t.Fatalf("parts = %#v, want 3 (text, degraded tool_call, tool_result)", parts)
	}

	if text, ok := parts[0].(TextPart); !ok || text.Text != "before" {
		t.Fatalf("parts[0] = %#v, want TextPart{before}", parts[0])
	}

	corrupted, ok := parts[1].(UnknownPart)
	if !ok || corrupted.Type != "tool_call" || string(corrupted.Data) != `"flat"` {
		t.Fatalf("parts[1] = %#v, want UnknownPart{tool_call} with raw payload", parts[1])
	}

	if result, ok := parts[2].(ToolResultPart); !ok || result.ToolCallID != "c1" || result.Content != "ok" {
		t.Fatalf("parts[2] = %#v, want ToolResultPart{c1}", parts[2])
	}
}

func TestMessagesOnLegacySchema(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	messages, err := db.Messages(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("Messages on legacy schema: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2 in fixture-root", len(messages))
	}

	childMessages, err := db.Messages(context.Background(), "m_child")
	if err != nil {
		t.Fatal(err)
	}

	if len(childMessages) != 1 {
		t.Fatalf("child messages = %d, want 1", len(childMessages))
	}

	for _, message := range append(messages, childMessages...) {
		if message.Model != "" || message.Provider != "" {
			t.Fatalf("message %s = %+v, want empty model/provider (columns absent)", message.ID, message)
		}

		if !message.FinishedAt.IsZero() {
			t.Fatalf("message %s FinishedAt = %v, want zero (column absent)", message.ID, message.FinishedAt)
		}
	}
}

func TestMessagesEmptySession(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	messages, err := db.Messages(context.Background(), "no-such-session")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(messages) != 0 {
		t.Fatalf("messages = %d, want 0", len(messages))
	}
}

func TestReadFiles(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	paths, err := db.ReadFiles(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("ReadFiles: %v", err)
	}

	if len(paths) != 1 || paths[0] != "/repo/main.go" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestReadFilesAbsentOnLegacySchema(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	paths, err := db.ReadFiles(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("ReadFiles on legacy schema: %v", err)
	}

	if len(paths) != 0 {
		t.Fatalf("paths = %v, want none (table absent)", paths)
	}
}

// TestReadFilesFiltersEmptyPaths pins the documented behavior: ReadFiles
// removes empty-string paths Go-side via slices.DeleteFunc, so rows with
// an empty path column do not appear in the result. Distinct read_at values
// double as the ordering pin: rows come back most recently read first.
func TestReadFilesFiltersEmptyPaths(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "session-with-empty-paths", "", "Has empty paths", 1, fixtureBase, fixtureBase)

		for _, row := range []struct {
			path   string
			readAt int64
		}{
			{path: "", readAt: fixtureBase + 1},
			{path: "/repo/a.go", readAt: fixtureBase + 2},
			{path: "", readAt: fixtureBase + 3},
			{path: "/repo/b.go", readAt: fixtureBase + 4},
			{path: "", readAt: fixtureBase + 5},
		} {
			if _, err := db.ExecContext(
				context.Background(),
				`INSERT INTO read_files (session_id, path, read_at) VALUES (?, ?, ?)`,
				"session-with-empty-paths", row.path, row.readAt,
			); err != nil {
				t.Fatal(err)
			}
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	paths, err := db.ReadFiles(context.Background(), "session-with-empty-paths")
	if err != nil {
		t.Fatalf("ReadFiles: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("paths = %v, want 2 (empty paths filtered)", paths)
	}

	// b.go was read last (fixtureBase+4), so it leads the result.
	if paths[0] != "/repo/b.go" || paths[1] != "/repo/a.go" {
		t.Fatalf("paths = %v, want [/repo/b.go /repo/a.go] (most recently read first)", paths)
	}
}

// TestIterMessagesMatchesMessages pins the iterator against the slice API:
// same rows, same order, same decoded fields — one query path, two shapes.
func TestIterMessagesMatchesMessages(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	materialized, err := db.Messages(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	var streamed []Message

	for message, err := range db.IterMessages(context.Background(), "fixture-root") {
		if err != nil {
			t.Fatalf("IterMessages: %v", err)
		}

		streamed = append(streamed, message)
	}

	if len(streamed) != len(materialized) {
		t.Fatalf("IterMessages yielded %d messages, Messages returned %d", len(streamed), len(materialized))
	}

	for i := range materialized {
		if !reflect.DeepEqual(streamed[i], materialized[i]) {
			t.Fatalf("message %d differs: iter = %+v, slice = %+v", i, streamed[i], materialized[i])
		}
	}
}

// TestIterMessagesEarlyBreak verifies that abandoning iteration mid-stream
// stops cleanly: the messages seen so far arrive in order and no error is
// surfaced for the rows left behind.
func TestIterMessagesEarlyBreak(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	var seen []string

	for message, err := range db.IterMessages(context.Background(), "fixture-root") {
		if err != nil {
			t.Fatalf("IterMessages: %v", err)
		}

		seen = append(seen, message.ID)

		if len(seen) == 3 {
			break
		}
	}

	if len(seen) != 3 {
		t.Fatalf("seen = %v, want the first 3 message IDs", seen)
	}

	materialized, err := db.Messages(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	for i, id := range seen {
		if id != materialized[i].ID {
			t.Fatalf("seen[%d] = %s, want %s (iteration order must match Messages)", i, id, materialized[i].ID)
		}
	}
}

// TestIterMessagesEmptySession yields nothing and no error for a session
// with no messages — the iterator equivalent of an empty slice.
func TestIterMessagesEmptySession(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	count := 0

	for _, err := range db.IterMessages(context.Background(), "no-such-session") {
		if err != nil {
			t.Fatalf("IterMessages: %v", err)
		}

		count++
	}

	if count != 0 {
		t.Fatalf("IterMessages yielded %d messages for a session with none", count)
	}
}

// TestIterMessagesCanceledContext verifies the error contract: a canceled
// context surfaces as a yielded error, not a panic or a silent stop.
func TestIterMessagesCanceledContext(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var (
		message Message
		err     error
		yields  int
	)

	for message, err = range db.IterMessages(ctx, "fixture-root") {
		yields++

		break
	}

	if yields != 1 {
		t.Fatalf("yields = %d, want exactly 1 (the error yield)", yields)
	}

	if err == nil {
		t.Fatal("IterMessages with a canceled context yielded no error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}

	if message.ID != "" {
		t.Fatalf("error yield carried message %q, want the zero value", message.ID)
	}
}

// TestMessagesIsSummaryMessage pins the is_summary_message scan path on both
// read shapes (Messages and IterMessages share scanMessage) and the
// legacy-schema zero substitution.
func TestMessagesIsSummaryMessage(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "s", "", "Session", 2, fixtureBase, fixtureBase+10)
		insertMessage(t, db, "m-plain", "s", "user", "[]", "", "", fixtureBase+1)
		insertMessage(t, db, "m-summary", "s", "assistant", "[]", fixtureModel, "", fixtureBase+2)

		if _, err := db.ExecContext(
			context.Background(),
			"UPDATE messages SET is_summary_message = 1 WHERE id = 'm-summary'",
		); err != nil {
			t.Fatal(err)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	if !db.Schema().MessagesIsSummaryMessage {
		t.Fatal("Schema.MessagesIsSummaryMessage = false, want true on the current schema")
	}

	byID := map[string]Message{}

	messages, err := db.Messages(context.Background(), "s")
	if err != nil {
		t.Fatal(err)
	}

	for _, message := range messages {
		byID[message.ID] = message
	}

	if byID["m-plain"].IsSummaryMessage {
		t.Fatal("m-plain IsSummaryMessage = true, want false")
	}

	if !byID["m-summary"].IsSummaryMessage {
		t.Fatal("m-summary IsSummaryMessage = false, want true")
	}

	summaryYields := 0

	for message, err := range db.IterMessages(context.Background(), "s") {
		if err != nil {
			t.Fatalf("IterMessages: %v", err)
		}

		want := message.ID == "m-summary"
		if message.IsSummaryMessage != want {
			t.Fatalf("%s IsSummaryMessage = %v, want %v", message.ID, message.IsSummaryMessage, want)
		}

		if want {
			summaryYields++
		}
	}

	if summaryYields != 1 {
		t.Fatalf("summary yields = %d, want 1", summaryYields)
	}

	legacy := openFixture(t, schemaLegacy)

	legacyMessages, err := legacy.Messages(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("Messages (legacy): %v", err)
	}

	for _, message := range legacyMessages {
		if message.IsSummaryMessage {
			t.Fatalf("%s IsSummaryMessage = true on a schema without the column", message.ID)
		}
	}
}

// TestMessagesSinceMatchesFullRead pins the zero cursor as a full read: the
// same rows in the same order with the same decoded fields as [DB.Messages],
// so a consumer starting from scratch and one paging through agree row for
// row.
func TestMessagesSinceMatchesFullRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := openFixture(t, schemaCurrent)

	full, err := db.Messages(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(full) == 0 {
		t.Fatal("fixture-root holds no messages — the comparison would be vacuous")
	}

	paged, err := db.MessagesSince(ctx, "fixture-root", 0)
	if err != nil {
		t.Fatalf("MessagesSince(0): %v", err)
	}

	if !reflect.DeepEqual(full, paged) {
		t.Fatalf("MessagesSince(0) returned %+v, want the full read %+v", paged, full)
	}
}

// TestMessagesSincePagesWithoutGapsOrRepeats walks the fixture one row per
// read by resuming from the rowid of the single row it just handled: every
// page is strictly past its cursor, no row is returned twice, and the walk
// reproduces the full read exactly. The exclusive cursor is what lets a
// consumer persist the rowid of the last message it finished.
func TestMessagesSincePagesWithoutGapsOrRepeats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := openFixture(t, schemaCurrent)

	full, err := db.Messages(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	var (
		cursor MessageRowID
		walked []string
	)

	for range len(full) + 1 { // one bounded extra read proves the walk ends
		page, err := db.MessagesSince(ctx, "fixture-root", cursor)
		if err != nil {
			t.Fatalf("MessagesSince(%d): %v", cursor, err)
		}

		if len(page) == 0 {
			break
		}

		if page[0].RowID <= cursor {
			t.Fatalf("page starts at rowid %d, not after the cursor %d", page[0].RowID, cursor)
		}

		walked = append(walked, page[0].ID)
		cursor = page[0].RowID
	}

	if len(walked) != len(full) {
		t.Fatalf("walked %d messages one row at a time, want %d", len(walked), len(full))
	}

	for i, message := range full {
		if walked[i] != message.ID {
			t.Fatalf("walk[%d] = %s, want %s", i, walked[i], message.ID)
		}
	}
}

// TestMessagesSinceResumesInsideASecond is the regression pin for the cursor
// unit. Crush stores whole seconds and a live session writes several messages
// per second, so a `created_at > cursor` filter excludes every message
// stamped in the cursor's own second — permanently, because the cursor only
// moves forward. The three rows below share one timestamp; resuming from the
// first row's cursor must still deliver the other two.
func TestMessagesSinceResumesInsideASecond(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ids := []string{"m_one", "m_two", "m_three"}
	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "s1", "", "Same second", len(ids), fixtureBase, fixtureBase)

		for _, id := range ids {
			insertMessage(t, db, id, "s1", "assistant", `[]`, "", "", fixtureBase)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	all, err := db.Messages(ctx, "s1")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(all) != len(ids) {
		t.Fatalf("messages = %d, want %d", len(all), len(ids))
	}

	rest, err := db.MessagesSince(ctx, "s1", all[0].RowID)
	if err != nil {
		t.Fatalf("MessagesSince: %v", err)
	}

	if len(rest) != len(ids)-1 {
		t.Fatalf(
			"resuming one row into a same-second group returned %d messages, want %d — a created_at cursor loses the rest of the second for good",
			len(rest),
			len(ids)-1,
		)
	}

	for i, message := range rest {
		if message.ID != ids[i+1] {
			t.Fatalf("resumed[%d] = %s, want %s", i, message.ID, ids[i+1])
		}
	}

	// A timestamp-shaped cursor compares against rowids and matches nothing.
	// The rows are still on disk, but only a caller who forces the conversion
	// can pass one — the silent-loss shape MessageRowID exists to prevent.
	stale, err := db.MessagesSince(ctx, "s1", MessageRowID(fixtureBase))
	if err != nil {
		t.Fatalf("MessagesSince(timestamp-shaped cursor): %v", err)
	}

	if len(stale) != 0 {
		t.Fatalf("MessagesSince(%d) returned %d messages, want none (the cursor is a rowid)", fixtureBase, len(stale))
	}
}

// TestMessagesSinceOnLegacySchema pins the since path on the oldest schema:
// the probe-gated columns degrade to zero values exactly as they do in
// [DB.Messages], and the rowid cursor works unchanged — messages is an
// ordinary rowid table in every Crush release.
func TestMessagesSinceOnLegacySchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := openFixture(t, schemaLegacy)

	full, err := db.Messages(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("Messages on legacy schema: %v", err)
	}

	if len(full) != 2 {
		t.Fatalf("messages = %d, want 2 in fixture-root", len(full))
	}

	rest, err := db.MessagesSince(ctx, "fixture-root", full[0].RowID)
	if err != nil {
		t.Fatalf("MessagesSince on legacy schema: %v", err)
	}

	if len(rest) != 1 || rest[0].ID != full[1].ID {
		t.Fatalf("MessagesSince returned %d messages (%+v), want just %s", len(rest), rest, full[1].ID)
	}

	if rest[0].Model != "" || rest[0].Provider != "" || !rest[0].FinishedAt.IsZero() {
		t.Fatalf("message %s = %+v, want zero values for the absent columns", rest[0].ID, rest[0])
	}

	watermark, err := db.LastMessageRowID(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("LastMessageRowID on legacy schema: %v", err)
	}

	if watermark != full[len(full)-1].RowID {
		t.Fatalf("LastMessageRowID = %d, want the newest row's %d", watermark, full[len(full)-1].RowID)
	}
}

// TestLastMessageRowID pins the watermark: it is the newest row's rowid, so
// resuming from it reads nothing, and it is scoped to its own session — the
// root's cursor never counts a child session's rows.
func TestLastMessageRowID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := openFixture(t, schemaCurrent)

	full, err := db.Messages(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	want := full[len(full)-1].RowID

	cursor, err := db.LastMessageRowID(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("LastMessageRowID: %v", err)
	}

	if cursor != want {
		t.Fatalf("LastMessageRowID = %d, want the newest row's %d", cursor, want)
	}

	resumed, err := db.MessagesSince(ctx, "fixture-root", cursor)
	if err != nil {
		t.Fatalf("MessagesSince(watermark): %v", err)
	}

	if len(resumed) != 0 {
		t.Fatalf("resuming from the watermark returned %d messages, want none", len(resumed))
	}

	child, err := db.LastMessageRowID(ctx, "m_assistant_1$$call_agent_1")
	if err != nil {
		t.Fatalf("LastMessageRowID (child): %v", err)
	}

	if child == 0 || child == cursor {
		t.Fatalf("child watermark = %d, want the child session's own newest rowid (root's is %d)", child, cursor)
	}
}

// TestLastMessageRowIDSessionWithoutMessages pins the zero cases: a session
// that exists but holds no messages, and an ID that was never written, both
// read 0 without an error — matching [DB.Messages], which returns an empty
// slice for either.
func TestLastMessageRowIDSessionWithoutMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "empty", "", "No messages yet", 0, fixtureBase, fixtureBase)
		insertSession(t, db, "written", "", "Has a message", 1, fixtureBase, fixtureBase)
		insertMessage(t, db, "m1", "written", "user", `[]`, "", "", fixtureBase)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	cursor, err := db.LastMessageRowID(ctx, "written")
	if err != nil {
		t.Fatalf("LastMessageRowID(written): %v", err)
	}

	if cursor == 0 {
		t.Fatal("LastMessageRowID(written) = 0, want the message's rowid")
	}

	for _, sessionID := range []string{"empty", "never-written"} {
		got, err := db.LastMessageRowID(ctx, sessionID)
		if err != nil {
			t.Fatalf("LastMessageRowID(%s): %v", sessionID, err)
		}

		if got != 0 {
			t.Fatalf("LastMessageRowID(%s) = %d, want 0", sessionID, got)
		}
	}
}

// TestIterMessagesSinceMatchesMessagesSince pins the iterator against the
// slice API on the since path: same rows, same order, same fields — and an
// early break leaves no error behind.
func TestIterMessagesSinceMatchesMessagesSince(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := openFixture(t, schemaCurrent)

	all, err := db.Messages(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	materialized, err := db.MessagesSince(ctx, "fixture-root", all[0].RowID)
	if err != nil {
		t.Fatalf("MessagesSince: %v", err)
	}

	var streamed []Message

	for message, err := range db.IterMessagesSince(ctx, "fixture-root", all[0].RowID) {
		if err != nil {
			t.Fatalf("IterMessagesSince: %v", err)
		}

		streamed = append(streamed, message)
	}

	if !reflect.DeepEqual(streamed, materialized) {
		t.Fatalf("IterMessagesSince streamed %+v, MessagesSince returned %+v", streamed, materialized)
	}

	broken := 0

	for _, err := range db.IterMessagesSince(ctx, "fixture-root", 0) {
		if err != nil {
			t.Fatalf("IterMessagesSince: %v", err)
		}

		broken++

		break
	}

	if broken != 1 {
		t.Fatalf("early break yielded %d messages, want 1", broken)
	}
}
