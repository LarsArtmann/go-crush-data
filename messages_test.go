package crushdata

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMessagesOrdersByCreatedAtThenID(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	messages, err := db.Messages(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(messages) != 11 {
		t.Fatalf("messages = %d, want 11", len(messages))
	}

	if messages[0].ID != "m_user" || messages[len(messages)-1].ID != "m_user_2" {
		t.Fatalf("order broken: first=%s last=%s", messages[0].ID, messages[len(messages)-1].ID)
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
// an empty path column do not appear in the result.
func TestReadFilesFiltersEmptyPaths(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "session-with-empty-paths", "", "Has empty paths", 1, fixtureBase, fixtureBase)

		for _, path := range []string{"", "/repo/a.go", "", "/repo/b.go", ""} {
			if _, err := db.ExecContext(
				context.Background(),
				`INSERT INTO read_files (session_id, path, read_at) VALUES (?, ?, ?)`,
				"session-with-empty-paths", path, fixtureBase+4,
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

	if paths[0] != "/repo/a.go" || paths[1] != "/repo/b.go" {
		t.Fatalf("paths = %v, want [/repo/a.go /repo/b.go]", paths)
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
