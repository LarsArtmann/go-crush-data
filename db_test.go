package crushdata

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCurrentSchema(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	if got := db.Schema(); !got.SessionsCost || !got.SessionsParentSessionID ||
		!got.MessagesModel || !got.MessagesProvider || !got.MessagesFinishedAt || !got.ReadFilesTable {
		t.Fatalf("Schema = %+v, want all capabilities on the current schema", got)
	}

	if len(db.Schema().MissingColumns()) != 0 {
		t.Fatalf("MissingColumns = %v, want none", db.Schema().MissingColumns())
	}
}

func TestOpenLegacySchema(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	schema := db.Schema()
	if schema.SessionsCost || schema.SessionsParentSessionID || schema.MessagesModel ||
		schema.MessagesProvider || schema.MessagesFinishedAt || schema.ReadFilesTable {
		t.Fatalf("Schema = %+v, want all capabilities off on the legacy schema", schema)
	}

	want := []string{
		"sessions.cost",
		"sessions.parent_session_id",
		"messages.model",
		"messages.provider",
		"messages.finished_at",
	}
	got := schema.MissingColumns()

	if len(got) != len(want) {
		t.Fatalf("MissingColumns = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MissingColumns = %v, want %v", got, want)
		}
	}
}

func TestOpenMissingDataDir(t *testing.T) {
	t.Parallel()

	_, err := Open(filepath.Join(t.TempDir(), "does-not-exist"))
	if !errors.Is(err, ErrDatabaseNotFound) {
		t.Fatalf("err = %v, want ErrDatabaseNotFound", err)
	}
}

func TestOpenEmptyDataDir(t *testing.T) {
	t.Parallel()

	_, err := Open(t.TempDir())
	if !errors.Is(err, ErrDatabaseNotFound) {
		t.Fatalf("err = %v, want ErrDatabaseNotFound", err)
	}
}

func TestOpenEmptyDatabaseFile(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, DBName), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dataDir)
	if !errors.Is(err, ErrDatabaseNotFound) {
		t.Fatalf("err = %v, want ErrDatabaseNotFound", err)
	}
}

func TestOpenDatabaseIsDirectory(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dataDir, DBName), 0o750); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dataDir)
	if !errors.Is(err, ErrDatabaseNotFound) {
		t.Fatalf("err = %v, want ErrDatabaseNotFound", err)
	}
}

func TestOpenEmptyDataDirArgument(t *testing.T) {
	t.Parallel()

	if _, err := Open(""); err == nil {
		t.Fatal("err = nil, want error for empty dataDir")
	}
}

// TestOpenUnsupportedSchema feeds a foreign SQLite database and one missing
// the messages table; both must fail with ErrUnsupportedSchema.
func TestOpenUnsupportedSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		create func(t *testing.T, dbPath string)
	}{
		{
			name: "foreign database",
			create: func(t *testing.T, dbPath string) {
				t.Helper()

				createRawDBAt(t, dbPath, "CREATE TABLE unrelated (id TEXT)")
			},
		},
		{
			name: "sessions only",
			create: func(t *testing.T, dbPath string) {
				t.Helper()

				createRawDBAt(t, dbPath, `CREATE TABLE sessions (id TEXT PRIMARY KEY)`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dataDir := t.TempDir()
			dbPath := filepath.Join(dataDir, DBName)
			tt.create(t, dbPath)

			db, err := Open(dataDir)
			if !errors.Is(err, ErrUnsupportedSchema) {
				t.Fatalf("err = %v, want ErrUnsupportedSchema", err)
			}

			if db != nil {
				t.Fatal("db != nil on failure")
			}
		})
	}
}

// createRawDBAt creates a SQLite database with arbitrary DDL.
func createRawDBAt(t *testing.T, dbPath, ddl string) {
	t.Helper()

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	db := openWritable(t, dbPath)

	if _, err := db.ExecContext(context.Background(), ddl); err != nil {
		t.Fatal(err)
	}
}

// TestOpenIsReadOnly proves reads leave the database byte-identical: the
// SHA-256 of crush.db (plus any WAL/SHM sidecars) is unchanged after a full
// read pass.
func TestOpenIsReadOnly(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)
	path := db.Path()

	before := hashTree(t, path)

	if _, err := db.Sessions(context.Background(), SessionFilter{}); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Messages(context.Background(), "fixture-root"); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay()}); err != nil {
		t.Fatal(err)
	}

	_ = db.Close()

	if after := hashTree(t, path); before != after {
		t.Fatal("database bytes changed during read-only usage")
	}
}

// hashTree hashes crush.db plus its WAL and SHM sidecars, ignoring missing
// files.
func hashTree(t *testing.T, dbPath string) string {
	t.Helper()

	hash := sha256.New()

	for _, suffix := range []string{"", "-wal", "-shm"} {
		//nolint:gosec // reading the test's own fixture files
		data, err := os.ReadFile(dbPath + suffix)
		if err != nil {
			continue
		}

		hash.Write(data)
	}

	return string(hash.Sum(nil))
}
