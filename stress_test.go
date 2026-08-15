package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
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

// TestSessionsConcurrentWithWALWriter pins the live-database contract: a
// read-only handle keeps listing sessions without error while a separate
// WAL-mode writer commits new rows into the same file — the situation this
// library exists for (reading while Crush runs).
func TestSessionsConcurrentWithWALWriter(t *testing.T) {
	t.Parallel()

	const written = 50

	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, DBName)

	createDBAt(t, dataDir+"/"+DBName, schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "seed", "", "Seed", 0, fixtureBase, fixtureBase)
	})

	writer := openWritable(t, dbPath)

	if _, err := writer.ExecContext(context.Background(), "PRAGMA journal_mode=WAL"); err != nil {
		t.Fatal(err)
	}

	if _, err := writer.ExecContext(context.Background(), "PRAGMA busy_timeout=5000"); err != nil {
		t.Fatal(err)
	}

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	writeDone := make(chan error, 1)

	go func() {
		var writeErr error

		for i := range written {
			_, writeErr = writer.ExecContext(
				context.Background(),
				`INSERT INTO sessions (id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				fmt.Sprintf("wal-%04d", i),
				nil,
				"WAL session",
				0,
				0,
				0,
				0.0,
				fixtureBase+int64(i)+1,
				fixtureBase+int64(i)+1,
			)
			if writeErr != nil {
				break
			}

			// Guarantee reader/writer overlap regardless of machine speed.
			time.Sleep(200 * time.Microsecond)
		}

		writeDone <- writeErr
	}()

	reads := 0

	for {
		sessions, err := db.Sessions(context.Background(), SessionFilter{})
		if err != nil {
			t.Fatalf("concurrent read failed after %d successful reads: %v", reads, err)
		}

		reads++

		if len(sessions) == 0 {
			t.Fatal("reader saw zero sessions mid-write; snapshot must at least contain the seed row")
		}

		select {
		case writeErr := <-writeDone:
			if writeErr != nil {
				t.Fatalf("writer failed: %v", writeErr)
			}

			sessions, err := db.Sessions(context.Background(), SessionFilter{})
			if err != nil {
				t.Fatalf("final read: %v", err)
			}

			if len(sessions) != written+1 {
				t.Fatalf("sessions = %d, want %d after writer finished", len(sessions), written+1)
			}

			if reads < 2 {
				t.Fatalf("reads = %d, want overlap between reader and writer", reads)
			}

			return
		default:
		}
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

// TestMessagesConcurrentWithWALWriter pins the same live-database contract
// as TestSessionsConcurrentWithWALWriter but for Messages: a read-only
// handle keeps reading messages without error while a WAL-mode writer
// inserts new message rows into the same file.
func TestMessagesConcurrentWithWALWriter(t *testing.T) {
	t.Parallel()

	const written = 30

	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, DBName)

	createDBAt(t, dbPath, schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "seed-session", "", "Seed", 2, fixtureBase, fixtureBase)
		insertMessage(
			t, db, "seed-msg-1", "seed-session", "user",
			fixturePartsUser, "", "", fixtureBase,
		)
		insertMessage(
			t, db, "seed-msg-2", "seed-session", "assistant",
			fixturePartsAgentCall, fixtureModel, "", fixtureBase+1,
		)
	})

	writer := openWritable(t, dbPath)

	if _, err := writer.ExecContext(context.Background(), "PRAGMA journal_mode=WAL"); err != nil {
		t.Fatal(err)
	}

	if _, err := writer.ExecContext(context.Background(), "PRAGMA busy_timeout=5000"); err != nil {
		t.Fatal(err)
	}

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	writeDone := make(chan error, 1)

	go func() {
		var writeErr error

		for i := range written {
			_, writeErr = writer.ExecContext(
				context.Background(),
				`INSERT INTO messages (id, session_id, role, parts, model, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				fmt.Sprintf("wal-msg-%04d", i),
				"seed-session",
				"assistant",
				`[{"data":{"text":"step"},"type":"text"}]`,
				fixtureModel,
				fixtureBase+int64(i)+2,
				fixtureBase+int64(i)+2,
			)
			if writeErr != nil {
				break
			}

			time.Sleep(200 * time.Microsecond)
		}

		writeDone <- writeErr
	}()

	reads := 0

	for {
		messages, err := db.Messages(context.Background(), "seed-session")
		if err != nil {
			t.Fatalf("concurrent Messages read failed after %d reads: %v", reads, err)
		}

		reads++

		if len(messages) < 2 {
			t.Fatal("reader saw fewer than seed messages mid-write")
		}

		select {
		case writeErr := <-writeDone:
			if writeErr != nil {
				t.Fatalf("writer failed: %v", writeErr)
			}

			messages, err := db.Messages(context.Background(), "seed-session")
			if err != nil {
				t.Fatalf("final Messages read: %v", err)
			}

			if len(messages) != written+2 {
				t.Fatalf("messages = %d, want %d after writer finished", len(messages), written+2)
			}

			if reads < 2 {
				t.Fatalf("reads = %d, want overlap between reader and writer", reads)
			}

			return
		default:
		}
	}
}

func BenchmarkMessages(b *testing.B) {
	dataDir := b.TempDir()

	createDBAt(b, dataDir+"/"+DBName, schemaCurrent, func(db *sql.DB) {
		insertSession(b, db, "bench-msgs", "", "bench", 2000, fixtureBase, fixtureBase)

		for i := range 2000 {
			insertMessage(
				b, db,
				fmt.Sprintf("bench-msg-%04d", i),
				"bench-msgs",
				"assistant",
				`[{"data":{"text":"step"},"type":"text"}]`,
				fixtureModel, "",
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
		if _, err := db.Messages(context.Background(), "bench-msgs"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAgentGraph(b *testing.B) {
	dataDir := b.TempDir()

	createDBAt(b, dataDir+"/"+DBName, schemaCurrent, func(db *sql.DB) {
		insertSession(b, db, "root", "", "Root", 1, fixtureBase, fixtureBase)

		for i := range 100 {
			childID := fmt.Sprintf("child-%04d", i)
			insertSession(
				b, db,
				childID, "root",
				fmt.Sprintf("Child %d", i),
				0, fixtureBase+int64(i)+1, fixtureBase+int64(i)+1,
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
		if _, err := db.AgentGraph(context.Background(), "root"); err != nil {
			b.Fatal(err)
		}
	}
}
