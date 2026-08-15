package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // the driver the examples use to build throwaway fixtures
)

// setupExampleData creates a throwaway data directory holding a minimal
// crush.db (sessions + messages only — the shape of an old Crush version)
// with two sessions and one message, plus a projects.json registry pointing
// at it. It returns the registry directory to pass as DiscoverOptions
// .GlobalDataDir.
//
// Real programs skip all of this: Crush has already written both files and
// discovery finds them via GlobalDataDir.
func setupExampleData() string {
	root, err := os.MkdirTemp("", "crushdata-example")
	if err != nil {
		panic(err)
	}

	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.Join(dataDir, DBName))
	if err != nil {
		panic(err)
	}

	defer func() { _ = db.Close() }()

	for _, statement := range []string{
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY, title TEXT NOT NULL,
			message_count INTEGER NOT NULL DEFAULT 0,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL, created_at INTEGER NOT NULL, todos TEXT)`,
		`CREATE TABLE messages (
			id TEXT PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL,
			parts TEXT NOT NULL DEFAULT '[]',
			created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`,
		`INSERT INTO sessions VALUES
			('s1', 'Refactor the collector', 1, 120, 80, 1785585700, 1785585600, NULL),
			('s2', 'Write the README',      0,   0,  0, 1785585900, 1785585800, NULL)`,
		`INSERT INTO messages VALUES
			('m1', 's1', 'user',
			 '[{"type":"text","data":{"text":"List the fixtures."}}]',
			 1785585600, 1785585600)`,
	} {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			panic(err)
		}
	}

	globalDir := filepath.Join(root, "global")
	registry := `{"projects":[{` +
		`"path":"/home/me/project",` +
		`"data_dir":` + quoteJSON(dataDir) + `,` +
		`"last_accessed":"2026-08-04T12:00:00Z"}]}`

	if err := os.MkdirAll(globalDir, 0o750); err != nil {
		panic(err)
	}

	if err := os.WriteFile(filepath.Join(globalDir, RegistryName), []byte(registry), 0o600); err != nil {
		panic(err)
	}

	return globalDir
}

// quoteJSON renders s as a JSON string literal.
func quoteJSON(s string) string {
	return `"` + s + `"`
}

// ExampleDiscoverProjects lists every project Crush knows about.
func ExampleDiscoverProjects() {
	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: setupExampleData(),
	})
	if err != nil {
		panic(err)
	}

	for _, project := range projects {
		fmt.Printf("%s -> %s\n", project.Path, filepath.Base(project.DataDir))
		fmt.Printf("last used %s\n", project.LastAccessed.Format(time.DateOnly))
	}

	// Output:
	// /home/me/project -> data
	// last used 2026-08-04
}

// ExampleDB_Sessions opens a database read-only and lists its sessions,
// newest activity first.
func ExampleDB_Sessions() {
	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: setupExampleData(),
	})
	if err != nil {
		panic(err)
	}

	db, err := Open(projects[0].DataDir)
	if err != nil {
		panic(err)
	}

	defer func() { _ = db.Close() }()

	// Old databases report the columns they lack; warn or degrade gracefully.
	for _, missing := range db.Schema().MissingColumns() {
		fmt.Printf("note: %s absent, some fields will be zero\n", missing)
	}

	sessions, err := db.Sessions(context.Background(), SessionFilter{})
	if err != nil {
		panic(err)
	}

	for _, session := range sessions {
		fmt.Printf("%s %q (%d messages)\n", session.ID, session.Title, session.MessageCount)
	}

	// Output:
	// note: sessions.cost absent, some fields will be zero
	// note: sessions.parent_session_id absent, some fields will be zero
	// note: messages.model absent, some fields will be zero
	// note: messages.provider absent, some fields will be zero
	// note: messages.finished_at absent, some fields will be zero
	// s2 "Write the README" (0 messages)
	// s1 "Refactor the collector" (1 messages)
}

// ExampleDB_Messages walks one session's messages and switches on the
// decoded part types.
func ExampleDB_Messages() {
	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: setupExampleData(),
	})
	if err != nil {
		panic(err)
	}

	db, err := Open(projects[0].DataDir)
	if err != nil {
		panic(err)
	}

	defer func() { _ = db.Close() }()

	messages, err := db.Messages(context.Background(), "s1")
	if err != nil {
		panic(err)
	}

	for _, message := range messages {
		fmt.Printf("[%s] %s\n", message.Role, message.ID)

		for _, part := range message.Parts {
			switch typed := part.(type) {
			case TextPart:
				fmt.Printf("  text: %s\n", typed.Text)
			case ToolCallPart:
				fmt.Printf("  tool call: %s\n", typed.Name)
			case ToolResultPart:
				fmt.Printf("  tool result for %s (error: %v)\n", typed.Name, typed.IsError)
			case ReasoningPart:
				fmt.Printf("  reasoning: %d bytes\n", len(typed.Thinking))
			default:
				fmt.Printf("  other part: %T\n", part)
			}
		}
	}

	// Output:
	// [user] m1
	//   text: List the fixtures.
}

// ExampleDB_Stats aggregates one calendar day of activity.
func ExampleDB_Stats() {
	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: setupExampleData(),
	})
	if err != nil {
		panic(err)
	}

	db, err := Open(projects[0].DataDir)
	if err != nil {
		panic(err)
	}

	defer func() { _ = db.Close() }()

	day := time.Unix(1785585600, 0).UTC()

	stats, err := db.Stats(context.Background(), StatsFilter{Day: day})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d sessions, %d messages, %d prompt tokens\n",
		stats.SessionCount, stats.MessageCount, stats.PromptTokens)

	for hour, count := range stats.HourHistogram {
		if count > 0 {
			fmt.Printf("%02d:00 %d session(s)\n", hour, count)
		}
	}

	// Output:
	// 2 sessions, 1 messages, 120 prompt tokens
	// 12:00 2 session(s)
}
