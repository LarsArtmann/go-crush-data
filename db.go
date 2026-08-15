package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // register the "sqlite" driver
)

// UnsupportedSchemaError reports a database that lacks a required table and
// therefore cannot be read. Test with [errors.Is](err, [ErrUnsupportedSchema]).
type UnsupportedSchemaError struct {
	Path         string
	MissingTable string
}

// Error implements the error interface.
func (e *UnsupportedSchemaError) Error() string {
	return ErrUnsupportedSchema.Error() +
		fmt.Sprintf(": %s is missing the %q table", e.Path, e.MissingTable)
}

// Is makes the error match [ErrUnsupportedSchema] via errors.Is.
func (e *UnsupportedSchemaError) Is(target error) bool {
	return target == ErrUnsupportedSchema
}

// DB is a read-only handle to one Crush session database. It is safe for
// concurrent use and must be closed with [DB.Close].
type DB struct {
	path   string
	handle *sql.DB
	schema Schema
}

// Open opens the crush.db under dataDir in read-only mode: SQLite's mode=ro
// flag avoids Crush's data-dir lock and migrations, and a single connection
// keeps reads predictable while a live Crush process writes. Opening never
// mutates the database.
//
// A missing crush.db fails with [ErrDatabaseNotFound]. A database without
// the required sessions and messages tables fails with
// [ErrUnsupportedSchema].
func Open(dataDir string) (*DB, error) {
	if dataDir == "" {
		return nil, errEmptyDataDir
	}

	path := filepath.Join(dataDir, DBName)

	if err := checkDatabaseFile(path); err != nil {
		return nil, err
	}

	handle, err := openSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("open crush database at %s: %w", path, err)
	}

	schema, err := probeSchema(context.Background(), handle, path)
	if err != nil {
		_ = handle.Close()

		return nil, err
	}

	return &DB{path: path, handle: handle, schema: schema}, nil
}

// checkDatabaseFile validates that path refers to a readable database file.
func checkDatabaseFile(path string) error {
	//nolint:gosec // path is caller-supplied by design: reading local files at arbitrary paths is this library's purpose
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseNotFound, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%w: %s is a directory", ErrDatabaseNotFound, path)
	}

	if info.Size() == 0 {
		return fmt.Errorf("%w: %s is empty", ErrDatabaseNotFound, path)
	}

	return nil
}

// openSQLite opens the database in read-only mode with a single connection.
//
// mode=ro tells the driver to use SQLite's read-only flag; the
// _txlock=immediate hint matches Crush's own open path so concurrent readers
// do not see torn WAL pages from the writer.
func openSQLite(path string) (*sql.DB, error) {
	params := url.Values{}
	params.Set("mode", "ro")
	params.Set("_txlock", "immediate")
	dsn := fmt.Sprintf("file:%s?%s", path, params.Encode())

	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite handle for %s: %w", path, err)
	}

	handle.SetMaxOpenConns(1)

	return handle, nil
}

// Close releases the database connection. Calling Close on an already-closed
// DB is an error; a DB must not be used after Close.
func (db *DB) Close() error {
	if err := db.handle.Close(); err != nil {
		return fmt.Errorf("close crush database at %s: %w", db.path, err)
	}

	return nil
}

// Schema returns the capabilities detected when the database was opened.
func (db *DB) Schema() Schema {
	return db.schema
}

// Path returns the filesystem path of the underlying crush.db.
func (db *DB) Path() string {
	return db.path
}
