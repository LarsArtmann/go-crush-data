package crushdata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

// TestCollectRowsSurfacesIterationError pins the rows.Err branch: when the
// context is cancelled mid-iteration, the failure surfaces as an "iterate
// ... rows" error wrapping the context error instead of a silent short read.
func TestCollectRowsSurfacesIterationError(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "s", "", "Session", 0, fixtureBase, fixtureBase)
	})

	handle := openWritable(t, filepath.Join(dataDir, DBName))

	if _, err := handle.ExecContext(
		context.Background(),
		`WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM seq WHERE n < 20000)
		 INSERT INTO read_files (session_id, path, read_at)
		 SELECT 's', 'file-' || n, 0 FROM seq`,
	); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := handle.QueryContext(ctx, "SELECT path FROM read_files")
	if err != nil {
		t.Fatal(err)
	}

	scanned := 0

	_, err = collectRows(rows, "cancelled read_files", func(rows *sql.Rows) (string, error) {
		scanned++

		if scanned == 1 {
			cancel()
		}

		var path string
		if err := rows.Scan(&path); err != nil {
			return "", fmt.Errorf("scan cancelled row: %w", err)
		}

		return path, nil
	})
	if err == nil {
		t.Fatal("err = nil, want iteration error after mid-iteration cancel")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}

	if scanned >= 20000 {
		t.Fatalf("scanned = %d, want the iteration to stop early", scanned)
	}
}
