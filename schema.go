package crushdata

import (
	"context"
	"database/sql"
	"fmt"
)

// Schema records which optional tables and columns a database carries.
// Crush adds columns and tables in migrations; databases written by older
// Crush versions lack them. Reads substitute zero values for absent columns
// instead of failing, and consumers can use [Schema.MissingColumns] to warn
// about reduced coverage.
type Schema struct {
	// SessionsCost reports whether sessions.cost exists. Absent: CostUSD is
	// always 0.
	SessionsCost bool

	// SessionsParentSessionID reports whether sessions.parent_session_id
	// exists. Absent: every session is a root and [DB.AgentGraph] degrades to
	// a single node.
	SessionsParentSessionID bool

	// MessagesModel reports whether messages.model exists. Absent: message
	// and stats model fields are empty.
	MessagesModel bool

	// MessagesProvider reports whether messages.provider exists. Absent:
	// message and stats provider fields are empty.
	MessagesProvider bool

	// MessagesFinishedAt reports whether messages.finished_at exists. Absent:
	// Message.FinishedAt is the zero time.
	MessagesFinishedAt bool

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

	if !s.MessagesModel {
		missing = append(missing, "messages.model")
	}

	if !s.MessagesProvider {
		missing = append(missing, "messages.provider")
	}

	if !s.MessagesFinishedAt {
		missing = append(missing, "messages.finished_at")
	}

	return missing
}

// requiredTables are the tables this library cannot function without.
//
//nolint:gochecknoglobals // a slice cannot be a constant and this one is definitionally fixed
var requiredTables = []string{"sessions", "messages"}

// costColumn names the sessions cost column shared by the schema probe and
// the capability-substituted cost expressions of the sessions, stats, and
// agent-subtree queries.
const costColumn = "cost"

// probeSchema inspects an open database and returns its capabilities.
//
// Probe failures surface as errors: a canceled context or an unreadable
// sqlite_master must not masquerade as a missing column (the difference
// between "this database predates a migration" and "this database is
// broken"). A database whose required tables are verifiably missing fails
// with an error wrapping [ErrUnsupportedSchema].
func probeSchema(ctx context.Context, db *sql.DB, path string) (Schema, error) {
	var (
		schema Schema
		err    error
	)

	if schema.SessionsCost, err = columnExists(ctx, db, "sessions", costColumn); err != nil {
		return Schema{}, wrapProbeError(path, err)
	}

	if schema.SessionsParentSessionID, err = columnExists(
		ctx, db, "sessions", "parent_session_id",
	); err != nil {
		return Schema{}, wrapProbeError(path, err)
	}

	if schema.MessagesModel, err = columnExists(ctx, db, "messages", "model"); err != nil {
		return Schema{}, wrapProbeError(path, err)
	}

	if schema.MessagesProvider, err = columnExists(ctx, db, "messages", "provider"); err != nil {
		return Schema{}, wrapProbeError(path, err)
	}

	if schema.MessagesFinishedAt, err = columnExists(
		ctx, db, "messages", "finished_at",
	); err != nil {
		return Schema{}, wrapProbeError(path, err)
	}

	if schema.ReadFilesTable, err = tableExists(ctx, db, "read_files"); err != nil {
		return Schema{}, wrapProbeError(path, err)
	}

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
