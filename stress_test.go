package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

// TestSessionsLargeDatabase is a correctness smoke test under volume: 500
// sessions must list without error and in full.
func TestSessionsLargeDatabase(t *testing.T) {
	t.Parallel()

	const sessionCount = 500

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		for i := range sessionCount {
			id := fmt.Sprintf("sess-large-%04d", i)
			insertSession(t, db, id, "", fmt.Sprintf("Session %d", i), 0, fixtureBase+int64(i), fixtureBase+int64(i))
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	sessions, err := db.Sessions(context.Background(), SessionFilter{})
	if err != nil {
		t.Fatalf("Sessions on %d rows: %v", sessionCount, err)
	}

	if len(sessions) != sessionCount {
		t.Fatalf("sessions = %d, want %d", len(sessions), sessionCount)
	}
}

// TestMessagesLargeHistory parses a 1000-message session, interleaving tool
// calls and results, and must decode every row without error.
func TestMessagesLargeHistory(t *testing.T) {
	t.Parallel()

	const messageCount = 1000

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "sess-many-msgs", "", "Many messages", messageCount, fixtureBase, fixtureBase+messageCount)

		for i := range messageCount {
			role := "tool"

			parts := fmt.Sprintf(
				`[{"data":{"content":"ok","is_error":false,"name":"read","tool_call_id":"call_%04d"},"type":"tool_result"}]`,
				i-1,
			)
			if i%2 == 0 {
				role = "assistant"

				parts = fmt.Sprintf(
					`[{"data":{"text":"step %d"},"type":"text"},{"data":{"finished":true,"id":"call_%04d","input":"{\"file_path\":\"/repo/file.go\"}","name":"read"},"type":"tool_call"}]`,
					i,
					i,
				)
			}

			insertMessage(
				t,
				db,
				fmt.Sprintf("msg-%04d", i),
				"sess-many-msgs",
				role,
				parts,
				"test-model",
				"",
				fixtureBase+int64(i),
			)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	messages, err := db.Messages(context.Background(), "sess-many-msgs")
	if err != nil {
		t.Fatalf("Messages on %d rows: %v", messageCount, err)
	}

	if len(messages) != messageCount {
		t.Fatalf("messages = %d, want %d", len(messages), messageCount)
	}

	decoded := 0

	for _, message := range messages {
		decoded += len(message.Parts)
	}

	if decoded < messageCount {
		t.Fatalf("decoded parts = %d, want at least one per message", decoded)
	}
}

func BenchmarkSessionsList(b *testing.B) {
	dataDir := b.TempDir()
	createDBAt(b, dataDir+"/"+DBName, schemaCurrent, func(db *sql.DB) {
		for i := range 2000 {
			insertSession(
				b,
				db,
				fmt.Sprintf("bench-sess-%04d", i),
				"",
				"bench",
				0,
				fixtureBase+int64(i),
				fixtureBase+int64(i),
			)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	b.ReportAllocs()

	for b.Loop() {
		if _, err := db.Sessions(context.Background(), SessionFilter{}); err != nil {
			b.Fatal(err)
		}
	}
}
