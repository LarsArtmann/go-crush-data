// genschema applies the pinned charmbracelet/crush goose migrations (Up
// sections only, in filename order) to a throwaway SQLite database and
// dumps the resulting schema as the CREATE statements SQLite itself
// reports. It replaces hand-maintained schema archaeology with a
// mechanical view of upstream: the dump is the checked-in snapshot
// docs/storage-schema-v0.92.0.sql, and the fixture-DDL guard test parses
// the same snapshot so test fixtures can never silently drift from the
// real schema again.
//
// Usage:
//
//	go run ./scripts/genschema -migrations /path/to/internal/db/migrations \
//	  > docs/storage-schema-v0.92.0.sql
//
// The migrations directory comes from a clone of the pinned release (see
// scripts/check-upstream-drift.sh, CRUSH_UPSTREAM_DIR).
package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "modernc.org/sqlite" // register the "sqlite" driver
)

func main() {
	migrationsDir := flag.String("migrations", "", "directory of goose .sql migrations (required)")

	flag.Parse()

	if *migrationsDir == "" {
		fail("usage: genschema -migrations <dir>")
	}

	entries, err := filepath.Glob(filepath.Join(*migrationsDir, "*.sql"))
	if err != nil || len(entries) == 0 {
		fail("no migrations found in " + *migrationsDir)
	}

	slices.Sort(entries)

	handle, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fail(err.Error())
	}

	defer func() { _ = handle.Close() }()

	handle.SetMaxOpenConns(1)

	for _, path := range entries {
		for _, statement := range upStatements(mustRead(path)) {
			if _, err := handle.ExecContext(context.Background(), statement); err != nil {
				fail(fmt.Sprintf("apply %s: %v\nstatement: %s", filepath.Base(path), err, statement))
			}
		}
	}

	dumpSchema(handle)
}

// upStatements extracts the SQL statements of a goose migration's Up
// section: everything between "-- +goose Up" and "-- +goose Down" (or EOF),
// with goose marker lines removed. StatementBegin/StatementEnd blocks are
// kept whole — triggers and multi-line bodies contain embedded semicolons —
// while loose statements are split on their terminating semicolon.
func upStatements(raw string) []string {
	inUp := false
	inBlock := false

	var (
		statements []string
		current    []string
	)

	flush := func() {
		text := strings.TrimSpace(strings.Join(current, "\n"))
		current = current[:0]

		if text != "" {
			statements = append(statements, strings.TrimSuffix(text, ";"))
		}
	}

	for line := range strings.SplitSeq(raw, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "-- +goose Up"):
			inUp = true
		case strings.HasPrefix(trimmed, "-- +goose Down"):
			flush()

			inUp = false
		case !inUp:
			continue
		case strings.HasPrefix(trimmed, "-- +goose StatementBegin"):
			flush()

			inBlock = true
		case strings.HasPrefix(trimmed, "-- +goose StatementEnd"):
			flush()

			inBlock = false
		case strings.HasPrefix(trimmed, "--"):
			// plain SQL comment, drop it
		case inBlock:
			current = append(current, line)
		case strings.HasSuffix(trimmed, ";"):
			current = append(current, line)

			flush()
		default:
			current = append(current, line)
		}
	}

	flush()

	return statements
}

// schemaObject is one row of SQLite's schema catalog.
type schemaObject struct {
	objectType string
	name       string
	sql        string
}

// dumpSchema writes every CREATE statement SQLite reports, tables first,
// each preceded by a comment naming it. The output is deterministic, so the
// checked-in snapshot can be diffed.
func dumpSchema(handle *sql.DB) {
	ctx := context.Background()

	rows, err := handle.QueryContext(ctx, `
		SELECT type, name, sql FROM sqlite_master
		WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%'
		ORDER BY CASE type WHEN 'table' THEN 0 WHEN 'index' THEN 1 ELSE 2 END, name
	`)
	if err != nil {
		fail(fmt.Sprintf("read sqlite_master: %v", err))
	}

	defer func() { _ = rows.Close() }()

	writer := bufio.NewWriter(os.Stdout)

	defer func() { _ = writer.Flush() }()

	for rows.Next() {
		var object schemaObject
		if err := rows.Scan(&object.objectType, &object.name, &object.sql); err != nil {
			fail(fmt.Sprintf("scan sqlite_master row: %v", err))
		}

		fmt.Fprintf(writer, "-- %s: %s\n%s;\n\n", object.objectType, object.name, strings.TrimSpace(object.sql))
	}

	if err := rows.Err(); err != nil {
		fail(fmt.Sprintf("walk sqlite_master: %v", err))
	}
}

func mustRead(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		fail(err.Error())
	}

	return string(raw)
}

func fail(message string) {
	fmt.Fprintf(os.Stderr, "genschema: %s\n", message)
	os.Exit(1)
}
