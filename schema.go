package crushdata

import (
	"context"
	"database/sql"
	"fmt"
)

// Schema records which optional tables and columns a database carries.
// Crush adds columns and tables in migrations; databases written by older
// Crush versions lack them. Reads substitute zero values for absent columns
// instead of failing, and consumers can use [Schema.MissingColumns] (columns
// only) or [Schema.MissingCapabilities] (columns and optional tables) to
// warn about reduced coverage.
type Schema struct {
	// SessionsCost reports whether sessions.cost exists. Absent: CostUSD is
	// always 0.
	SessionsCost bool

	// SessionsParentSessionID reports whether sessions.parent_session_id
	// exists. Absent: every session is a root and [DB.AgentGraph] degrades to
	// a single node.
	SessionsParentSessionID bool

	// SessionsTodos reports whether sessions.todos exists (added upstream in
	// 2025-08-12's migration). Absent: Session.Todos is always nil.
	SessionsTodos bool

	// SessionsSummaryMessageID reports whether sessions.summary_message_id
	// exists (added upstream in 2025-05-15's migration). Absent:
	// Session.SummaryMessageID is always "".
	SessionsSummaryMessageID bool

	// MessagesModel reports whether messages.model exists. Absent: message
	// and stats model fields are empty.
	MessagesModel bool

	// MessagesProvider reports whether messages.provider exists. Absent:
	// message and stats provider fields are empty.
	MessagesProvider bool

	// MessagesFinishedAt reports whether messages.finished_at exists. Absent:
	// Message.FinishedAt is the zero time.
	MessagesFinishedAt bool

	// MessagesIsSummaryMessage reports whether messages.is_summary_message
	// exists (added upstream in 2025-08-10's migration). Absent:
	// Message.IsSummaryMessage is always false.
	MessagesIsSummaryMessage bool

	// ReadFilesTable reports whether the read_files table exists. Absent:
	// [DB.ReadFiles] returns an empty slice.
	ReadFilesTable bool
}

// MissingColumns lists the well-known columns this database lacks, in a
// stable order suitable for a user-facing warning ("upgrade Crush for full
// coverage").
func (s Schema) MissingColumns() []string {
	var missing []string

	if !s.SessionsCost {
		missing = append(missing, "sessions.cost")
	}

	if !s.SessionsParentSessionID {
		missing = append(missing, "sessions.parent_session_id")
	}

	if !s.SessionsTodos {
		missing = append(missing, "sessions.todos")
	}

	if !s.SessionsSummaryMessageID {
		missing = append(missing, "sessions.summary_message_id")
	}

	if !s.MessagesModel {
		missing = append(missing, "messages.model")
	}

	if !s.MessagesProvider {
		missing = append(missing, "messages.provider")
	}

	if !s.MessagesFinishedAt {
		missing = append(missing, "messages.finished_at")
	}

	if !s.MessagesIsSummaryMessage {
		missing = append(missing, "messages.is_summary_message")
	}

	return missing
}

// MissingCapabilities lists everything this database lacks — well-known
// columns and optional tables — in a stable order suitable for a user-facing
// warning ("upgrade Crush for full coverage"). It is a strict superset of
// [Schema.MissingColumns]: column gaps first (in MissingColumns order), then
// missing optional tables.
func (s Schema) MissingCapabilities() []string {
	missing := s.MissingColumns()

	if !s.ReadFilesTable {
		missing = append(missing, "read_files")
	}

	return missing
}

// requiredTables are the tables this library cannot function without.
//
//nolint:gochecknoglobals // a slice cannot be a constant and this one is definitionally fixed
var requiredTables = []string{sessionsTable, messagesTable}

// costColumn names the sessions cost column shared by the schema probe and
// the capability-substituted cost expressions of the sessions, stats, and
// agent-subtree queries.
const costColumn = "cost"

// todosColumn names the sessions todos column shared by the schema probe
// and the capability-substituted todos expressions of the sessions and
// agent-subtree queries.
const todosColumn = "todos"

// messagesTable names the messages table shared by the required-table
// check and the message-column probes.
const messagesTable = "messages"

// sessionsTable names the sessions table of the required-table check and
// the session-column probes.
const sessionsTable = "sessions"

// columnProbe wires one well-known column to the [Schema] field carrying its
// presence, so probeSchema stays a flat table walk instead of one branch per
// capability. Adding a probe is one row here plus the field; the drift guard
// (TestUpstreamMigrationColumnsAreProbedOrExempt) enforces the pairing.
type columnProbe struct {
	table  string
	column string
	assign func(schema *Schema, present bool)
}

//nolint:gochecknoglobals // a slice cannot be a constant and this one is definitionally fixed
var columnProbes = []columnProbe{
	{sessionsTable, costColumn, func(s *Schema, present bool) { s.SessionsCost = present }},
	{sessionsTable, "parent_session_id", func(s *Schema, present bool) { s.SessionsParentSessionID = present }},
	{sessionsTable, todosColumn, func(s *Schema, present bool) { s.SessionsTodos = present }},
	{sessionsTable, "summary_message_id", func(s *Schema, present bool) { s.SessionsSummaryMessageID = present }},
	{messagesTable, "model", func(s *Schema, present bool) { s.MessagesModel = present }},
	{messagesTable, "provider", func(s *Schema, present bool) { s.MessagesProvider = present }},
	{messagesTable, "finished_at", func(s *Schema, present bool) { s.MessagesFinishedAt = present }},
	{messagesTable, "is_summary_message", func(s *Schema, present bool) { s.MessagesIsSummaryMessage = present }},
}

// probeSchema inspects an open database and returns its capabilities.
//
// Probe failures surface as errors: a canceled context or an unreadable
// sqlite_master must not masquerade as a missing column (the difference
// between "this database predates a migration" and "this database is
// broken"). A database whose required tables are verifiably missing fails
// with an error wrapping [ErrUnsupportedSchema].
func probeSchema(ctx context.Context, db *sql.DB, path string) (Schema, error) {
	var schema Schema

	for _, probe := range columnProbes {
		present, err := columnExists(ctx, db, probe.table, probe.column)
		if err != nil {
			return Schema{}, wrapProbeError(path, err)
		}

		probe.assign(&schema, present)
	}

	readFiles, err := tableExists(ctx, db, "read_files")
	if err != nil {
		return Schema{}, wrapProbeError(path, err)
	}

	schema.ReadFilesTable = readFiles

	for _, table := range requiredTables {
		present, err := tableExists(ctx, db, table)
		if err != nil {
			return Schema{}, wrapProbeError(path, err)
		}

		if !present {
			return Schema{}, &UnsupportedSchemaError{Path: path, MissingTable: table}
		}
	}

	return schema, nil
}

// wrapProbeError names the database a probe failed against.
func wrapProbeError(path string, err error) error {
	return fmt.Errorf("probe schema of %s: %w", path, err)
}

// tableExists reports whether the named table exists.
func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	rows, err := db.QueryContext(
		ctx,
		"SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	)
	if err != nil {
		return false, fmt.Errorf("list tables: %w", err)
	}

	defer func() { _ = rows.Close() }()

	return rows.Next(), rows.Err()
}

// columnExists reports whether the named column exists on the named table,
// via the pragma_table_info table-valued function.
func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(
		ctx,
		"SELECT 1 FROM pragma_table_info(?) WHERE name = ?",
		table, column,
	)
	if err != nil {
		return false, fmt.Errorf("list columns of %s: %w", table, err)
	}

	defer func() { _ = rows.Close() }()

	return rows.Next(), rows.Err()
}
