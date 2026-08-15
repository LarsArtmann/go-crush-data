package crushdata

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite" // register the "sqlite" driver for fixture creation
)

// Schema variants exercised across the suite. Crush adds columns in
// migrations; every read must work on both shapes.
type schemaVariant string

const (
	// schemaCurrent mirrors the schema shipped by recent Crush versions.
	schemaCurrent schemaVariant = "current"

	// schemaLegacy predates the cost, parent_session_id, model, provider,
	// and finished_at migrations — the oldest databases in the wild.
	schemaLegacy schemaVariant = "legacy"
)

const currentSchemaDDL = `
	CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		parent_session_id TEXT,
		title TEXT NOT NULL,
		message_count INTEGER NOT NULL DEFAULT 0,
		prompt_tokens INTEGER NOT NULL DEFAULT 0,
		completion_tokens INTEGER NOT NULL DEFAULT 0,
		cost REAL NOT NULL DEFAULT 0.0,
		updated_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		todos TEXT
	);
	CREATE TABLE messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		parts TEXT NOT NULL DEFAULT '[]',
		model TEXT,
		provider TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		finished_at INTEGER
	);
	CREATE TABLE read_files (
		session_id TEXT NOT NULL,
		path TEXT NOT NULL,
		read_at INTEGER NOT NULL
	);
`

const legacySchemaDDL = `
	CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		message_count INTEGER NOT NULL DEFAULT 0,
		prompt_tokens INTEGER NOT NULL DEFAULT 0,
		completion_tokens INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		todos TEXT
	);
	CREATE TABLE messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		parts TEXT NOT NULL DEFAULT '[]',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
`

// fixtureDB creates a crush.db of the requested variant in a new temporary
// data directory, applies seed, and returns the data directory path.
func fixtureDB(t *testing.T, variant schemaVariant, seed func(db *sql.DB)) string {
	t.Helper()

	dataDir := t.TempDir()
	createDBAt(t, filepath.Join(dataDir, DBName), variant, seed)

	return dataDir
}

// createDBAt creates a Crush database of the requested variant at an exact
// path. The connection is closed before returning so read-only opens see a
// quiescent file.
func createDBAt(tb testing.TB, dbPath string, variant schemaVariant, seed func(db *sql.DB)) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		tb.Fatal(err)
	}

	handle, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		tb.Fatal(err)
	}

	handle.SetMaxOpenConns(1)

	ddl := currentSchemaDDL
	if variant == schemaLegacy {
		ddl = legacySchemaDDL
	}

	if _, err := handle.Exec(ddl); err != nil {
		_ = handle.Close()

		tb.Fatal(err)
	}

	if seed != nil {
		seed(handle)
	}

	if err := handle.Close(); err != nil {
		tb.Fatal(err)
	}
}

// openWritable opens a writable throwaway connection used by tests to build
// fixtures. The caller must close the handle.
func openWritable(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	handle, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}

	handle.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = handle.Close() })

	return handle
}

// insertSession inserts one session row on the current schema.
func insertSession(
	tb testing.TB,
	db *sql.DB,
	id, parentID, title string,
	messageCount int,
	createdAt, updatedAt int64,
) {
	tb.Helper()

	_, err := db.Exec(
		`INSERT INTO sessions (id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, nullable(parentID), title, messageCount, 0, 0, 0.0, updatedAt, createdAt,
	)
	if err != nil {
		tb.Fatal(err)
	}
}

// insertLegacySession inserts one session row on the legacy schema.
func insertLegacySession(
	t *testing.T,
	db *sql.DB,
	id, title string,
	messageCount int,
	createdAt, updatedAt int64,
) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO sessions (id, title, message_count, prompt_tokens, completion_tokens, updated_at, created_at)
		 VALUES (?, ?, ?, 0, 0, ?, ?)`,
		id, title, messageCount, updatedAt, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// insertMessage inserts one message row with model/provider on the current
// schema. model/provider may be "" for NULL.
func insertMessage(
	tb testing.TB,
	db *sql.DB,
	id, sessionID, role, parts, model, provider string,
	createdAt int64,
) {
	tb.Helper()

	_, err := db.Exec(
		`INSERT INTO messages (id, session_id, role, parts, model, provider, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, role, parts, nullable(model), nullable(provider), createdAt, createdAt,
	)
	if err != nil {
		tb.Fatal(err)
	}
}

// nullable maps "" to SQL NULL, the shape Crush writes for absent values.
func nullable(value string) any {
	if value == "" {
		return nil
	}

	return value
}

// seedFixtures mirrors a realistic session: a root conversation with agent,
// read, write, bash, reasoning, and shell-command parts plus a sub-agent
// child session. Timestamps are unix seconds.
const fixtureBase = int64(1785585600) // 2026-08-04 ~12:00 UTC

func seedFixture(db *sql.DB) {
	seedSessionWithEconomics(db, "fixture-root", "", "Fixture root session", 10, 5000, 1500, 0.0234,
		fixtureBase, fixtureBase+9)
	seedSessionWithEconomics(db, "m_assistant_1$$call_agent_1", "fixture-root", "Agent: read server", 1, 0, 0, 0.0,
		fixtureBase+1, fixtureBase+3)

	messages := []struct {
		id, sessionID, role, parts, model string
		createdAt                         int64
	}{
		{
			"m_user", "fixture-root", "user",
			`[{"data":{"text":"Read internal/server/server.go."},"type":"text"},{"data":{"reason":"stop","time":0},"type":"finish"}]`,
			"", fixtureBase,
		},
		{
			"m_assistant_1", "fixture-root", "assistant",
			`[{"data":{"text":"Reading the server."},"type":"text"},{"data":{"finished":true,"id":"call_agent_1","input":"{\"task_title\":\"read server\",\"agent_type\":\"explore\",\"message\":\"look at server.go\"}","name":"agent","provider_executed":false},"type":"tool_call"}]`,
			"minimax/minimax-m3", fixtureBase + 1,
		},
		{
			"m_tool_agent", "fixture-root", "tool",
			`[{"data":{"content":"{\"agent_id\":\"m_assistant_1\",\"nickname\":\"explore\",\"task_name\":\"read server\"}","is_error":false,"name":"agent","tool_call_id":"call_agent_1"},"type":"tool_result"}]`,
			"", fixtureBase + 2,
		},
		{
			"m_assistant_2", "fixture-root", "assistant",
			`[{"data":{"thinking":"considering which file to read","started_at":100,"finished_at":107},"type":"reasoning"},{"data":{"finished":true,"id":"call_read_1","input":"{\"file_path\":\"/repo/main.go\"}","name":"read","provider_executed":false},"type":"tool_call"}]`,
			"minimax/minimax-m3", fixtureBase + 3,
		},
		{
			"m_tool_read", "fixture-root", "tool",
			`[{"data":{"content":"package main\n","is_error":false,"name":"read","tool_call_id":"call_read_1"},"type":"tool_result"}]`,
			"", fixtureBase + 4,
		},
		{
			"m_assistant_3", "fixture-root", "assistant",
			`[{"data":{"text":"Now editing."},"type":"text"},{"data":{"finished":true,"id":"call_write_1","input":"{\"file_path\":\"/repo/server.go\"}","name":"write","provider_executed":false},"type":"tool_call"}]`,
			"minimax/minimax-m3", fixtureBase + 5,
		},
		{
			"m_tool_write", "fixture-root", "tool",
			`[{"data":{"content":"File written.","is_error":true,"name":"write","tool_call_id":"call_write_1"},"type":"tool_result"}]`,
			"", fixtureBase + 6,
		},
		{
			"m_assistant_4", "fixture-root", "assistant",
			`[{"data":{"command":"go test ./...","output":"ok","exit_code":0},"type":"shell_command"},{"data":{"finished":true,"id":"call_bash_1","input":"{\"command\":\"go test ./...\"}","name":"bash","provider_executed":false},"type":"tool_call"}]`,
			"minimax/minimax-m3", fixtureBase + 7,
		},
		{
			"m_tool_bash", "fixture-root", "tool",
			`[{"data":{"content":"ok\n","is_error":false,"name":"bash","tool_call_id":"call_bash_1"},"type":"tool_result"}]`,
			"", fixtureBase + 8,
		},
		{
			"m_user_2", "fixture-root", "user",
			`[{"data":{"text":"Looks good!"},"type":"text"},{"data":{"reason":"stop","time":0},"type":"finish"}]`,
			"", fixtureBase + 9,
		},
		{
			"m_child_1", "m_assistant_1$$call_agent_1", "assistant",
			`[{"data":{"text":"Reading the file."},"type":"text"}]`,
			"minimax/minimax-m3", fixtureBase + 2,
		},
		{
			"m_future", "fixture-root", "user",
			`[{"data":{"blob":"base64"},"type":"image_url"},{"data":{"payload":"..."},"type":"brand_new_part"}]`,
			"", fixtureBase + 9,
		},
	}

	for _, msg := range messages {
		var model any
		if msg.model != "" {
			model = msg.model
		}

		_, err := db.Exec(
			`INSERT INTO messages (id, session_id, role, parts, model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			msg.id, msg.sessionID, msg.role, msg.parts, model, msg.createdAt, msg.createdAt,
		)
		if err != nil {
			panic(err)
		}
	}

	if _, err := db.Exec(
		`INSERT INTO read_files (session_id, path, read_at) VALUES ('fixture-root', '/repo/main.go', ?)`,
		fixtureBase+4,
	); err != nil {
		panic(err)
	}
}

// seedSessionWithEconomics inserts a session row including token and cost
// columns.
func seedSessionWithEconomics(
	db *sql.DB,
	id, parentID, title string,
	messageCount int,
	promptTokens, completionTokens int64,
	cost float64,
	createdAt, updatedAt int64,
) {
	var parent any
	if parentID != "" {
		parent = parentID
	}

	_, err := db.Exec(
		`INSERT INTO sessions (id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, parent, title, messageCount, promptTokens, completionTokens, cost, updatedAt, createdAt,
	)
	if err != nil {
		panic(err)
	}
}

// fixtureDay is the UTC calendar day of fixtureBase.
func fixtureDay() time.Time {
	return time.Unix(fixtureBase, 0).UTC()
}

// openFixture opens the standard seeded fixture of the requested variant.
func openFixture(t *testing.T, variant schemaVariant) *DB {
	t.Helper()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), variant, func(db *sql.DB) {
		if variant == schemaLegacy {
			seedLegacyFixture(db)
		} else {
			seedFixture(db)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open fixture (%s): %v", variant, err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}

// seedLegacyFixture mirrors seedFixture on the legacy schema: same sessions
// and message bodies, minus the columns that did not exist yet.
func seedLegacyFixture(db *sql.DB) {
	seedLegacySessionFull(db, "fixture-root", "Fixture root session", 10, 5000, 1500,
		fixtureBase, fixtureBase+9)
	seedLegacySessionFull(db, "m_child", "Agent: read server", 1, 0, 0,
		fixtureBase+1, fixtureBase+3)

	messages := []struct {
		id, sessionID, role, parts string
		createdAt                  int64
	}{
		{
			"m_user", "fixture-root", "user",
			`[{"data":{"text":"Read server.go."},"type":"text"}]`, fixtureBase,
		},
		{
			"m_assistant_1", "fixture-root", "assistant",
			`[{"data":{"text":"Reading."},"type":"text"}]`, fixtureBase + 1,
		},
		{
			"m_child_1", "m_child", "assistant",
			`[{"data":{"text":"Reading the file."},"type":"text"}]`, fixtureBase + 2,
		},
	}

	for _, msg := range messages {
		if _, err := db.Exec(
			`INSERT INTO messages (id, session_id, role, parts, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			msg.id, msg.sessionID, msg.role, msg.parts, msg.createdAt, msg.createdAt,
		); err != nil {
			panic(err)
		}
	}
}

func seedLegacySessionFull(
	db *sql.DB,
	id, title string,
	messageCount int,
	promptTokens, completionTokens int64,
	createdAt, updatedAt int64,
) {
	_, err := db.Exec(
		`INSERT INTO sessions (id, title, message_count, prompt_tokens, completion_tokens, updated_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, title, messageCount, promptTokens, completionTokens, updatedAt, createdAt,
	)
	if err != nil {
		panic(err)
	}
}
