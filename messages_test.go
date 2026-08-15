package crushdata

import (
	"context"
	"database/sql"
	"path/filepath"
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
