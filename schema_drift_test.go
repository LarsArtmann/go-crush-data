package crushdata

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
)

// dropCapabilityFixture builds the standard seeded current-schema fixture and
// then executes dropStmt (a full DROP TABLE / ALTER TABLE … DROP COLUMN
// statement), mimicking a database written by a Crush version that predates
// that migration. The connection is closed before returning so read-only
// opens see a quiescent file.
func dropCapabilityFixture(t *testing.T, dropStmt string) *DB {
	t.Helper()

	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, DBName)

	createDBAt(t, dbPath, schemaCurrent, seedFixture)

	handle, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := handle.ExecContext(context.Background(), dropStmt); err != nil {
		_ = handle.Close()

		t.Fatalf("drop capability: %v", err)
	}

	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open degraded fixture (dropped %s): %v", dropStmt, err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}

// exerciseReadAPIs runs every public read path against db, failing on the
// first error. It is the behavioral half of the capability-coverage guard:
// a column the library SELECTs but does not probe turns exactly one of these
// calls into a "no such column" failure.
func exerciseReadAPIs(t *testing.T, db *DB) {
	t.Helper()

	ctx := context.Background()

	if _, err := db.Sessions(ctx, SessionFilter{}); err != nil {
		t.Fatalf("Sessions: %v", err)
	}

	if _, err := db.Stats(ctx, StatsFilter{}); err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if _, err := db.Stats(ctx, StatsFilter{Day: fixtureDay()}); err != nil {
		t.Fatalf("Stats(day): %v", err)
	}

	if _, err := db.Session(ctx, "fixture-root"); err != nil {
		t.Fatalf("Session: %v", err)
	}

	messages, err := db.Messages(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(messages) == 0 {
		t.Fatal("Messages returned no rows for the seeded root session")
	}

	for message, err := range db.IterMessages(ctx, "fixture-root") {
		if err != nil {
			t.Fatalf("IterMessages: %v", err)
		}

		if message.ID == "" {
			t.Fatal("IterMessages yielded an empty ID")
		}
	}

	if _, err := db.AgentGraph(ctx, "fixture-root"); err != nil {
		t.Fatalf("AgentGraph: %v", err)
	}

	if _, err := db.ReadFiles(ctx, "fixture-root"); err != nil {
		t.Fatalf("ReadFiles: %v", err)
	}
}

// TestPreTodosSchemaDegradesGracefully pins the fix for the frozen-database
// compatibility bug: upstream added sessions.todos in 2025-08-12, and a
// database that predates the migration (and was never reopened by a newer
// Crush) must not make Sessions, Session, or AgentGraph fail with
// "no such column: todos" — the todos capability substitutes NULL instead.
func TestPreTodosSchemaDegradesGracefully(t *testing.T) {
	t.Parallel()

	db := dropCapabilityFixture(t, "ALTER TABLE sessions DROP COLUMN todos")

	if missing := db.Schema().MissingColumns(); !slices.Contains(missing, "sessions.todos") {
		t.Fatalf("MissingColumns = %v, want it to report sessions.todos", missing)
	}

	ctx := context.Background()

	sessions, err := db.Sessions(ctx, SessionFilter{})
	if err != nil {
		t.Fatalf("Sessions on a pre-todos schema: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("Sessions returned no rows")
	}

	for _, session := range sessions {
		if session.Todos != nil {
			t.Fatalf("session %s Todos = %s, want nil on a pre-todos schema", session.ID, session.Todos)
		}
	}

	graph, err := db.AgentGraph(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("AgentGraph on a pre-todos schema: %v", err)
	}

	for _, node := range graph.Nodes {
		if node.Session.Todos != nil {
			t.Fatalf("graph node %s Todos = %s, want nil on a pre-todos schema", node.Session.ID, node.Session.Todos)
		}
	}

	session, err := db.Session(ctx, "fixture-root")
	if err != nil {
		t.Fatalf("Session on a pre-todos schema: %v", err)
	}

	if session.Todos != nil {
		t.Fatalf("Session Todos = %s, want nil on a pre-todos schema", session.Todos)
	}
}

// TestUpstreamMigrationColumnsAreProbedOrExempt is the capability-coverage
// guard: every column and table a non-initial upstream migration adds must
// be either substituted by a probed capability or deliberately unread.
// Dropping each one from the full fixture and running the whole read surface
// is the behavioral check; the MissingColumns/Schema assertion proves the
// probe exists (a future SELECT that reads a currently-unread column without
// adding a probe fails exactly here).
//
// The list is derived from the pinned upstream migration set
// (charmbracelet/crush v0.92.0, commit 559ec80); refresh it — and check for
// upstream drift — with scripts/check-upstream-drift.sh. Columns of the
// initial migration (20250424200609) are deliberately absent: they are the
// stable identifiers every read requires, and dropping one SHOULD break
// reads.
func TestUpstreamMigrationColumnsAreProbedOrExempt(t *testing.T) {
	t.Parallel()

	type migration struct {
		table  string
		column string // "" for a whole-table migration
		file   string
		probed bool
	}

	// Currently-unread columns stay in the list with probed=false: dropping
	// them must change nothing, and the moment a query starts reading one,
	// its row must gain probed=true plus a Schema field — this test makes
	// that omission loud. summary_message_id and is_summary_message were the
	// first to make that journey (2026-09-08).
	migrations := []migration{
		{
			table:  "sessions",
			column: "summary_message_id",
			file:   "20250515105448_add_summary_message_id.sql",
			probed: true,
		},
		{
			table:  "messages",
			column: "provider",
			file:   "20250627000000_add_provider_to_messages.sql",
			probed: true,
		},
		{
			table:  "messages",
			column: "is_summary_message",
			file:   "20250810000000_add_is_summary_message.sql",
			probed: true,
		},
		{
			table:  "sessions",
			column: "todos",
			file:   "20250812000000_add_todos_to_sessions.sql",
			probed: true,
		},
		{
			table:  "read_files",
			file:   "20260127000000_add_read_files_table.sql",
			probed: true,
		},
	}

	for _, migration := range migrations {
		t.Run(migration.table+"."+migration.column, func(t *testing.T) {
			t.Parallel()

			var dropStmt string

			switch migration.column {
			case "":
				dropStmt = "DROP TABLE " + migration.table
			default:
				dropStmt = "ALTER TABLE " + migration.table + " DROP COLUMN " + migration.column
			}

			db := dropCapabilityFixture(t, dropStmt)

			exerciseReadAPIs(t, db)

			if !migration.probed {
				return
			}

			switch migration.column {
			case "":
				if db.Schema().ReadFilesTable {
					t.Fatal("Schema.ReadFilesTable = true, want false after dropping read_files")
				}
			default:
				want := migration.table + "." + migration.column

				if missing := db.Schema().MissingColumns(); !slices.Contains(missing, want) {
					t.Fatalf(
						"MissingColumns = %v, want %q listed after dropping it — add a Schema capability probe for %s",
						missing, want, want,
					)
				}
			}
		})
	}
}
