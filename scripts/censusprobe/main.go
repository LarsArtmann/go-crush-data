// censusprobe is machine-local scratch tooling for parts-envelope census
// investigations: it walks every database in a local Crush registry with a
// per-DB 30s budget and prints any that are slow or failing. It is not part
// of the library or any CI contract — keep package-main files under
// scripts/, never in the repo root (a root-level package main breaks the
// single-package build).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LarsArtmann/go-crush-data"
	_ "modernc.org/sqlite"
)

type registryEntry struct {
	DataDir string `json:"data_dir"`
}

type registryFile struct {
	Projects []registryEntry `json:"projects"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	registryPath := os.Getenv("CENSUSPROBE_REGISTRY")
	if registryPath == "" {
		registryPath = filepath.Join(crushdata.GlobalDataDir(), crushdata.RegistryName)
	}

	raw, err := os.ReadFile(registryPath)
	must(err)

	var registry registryFile
	must(json.Unmarshal(raw, &registry))

	seen := map[string]bool{}

	for _, project := range registry.Projects {
		dir := project.DataDir
		if dir == "" || seen[dir] {
			continue
		}

		seen[dir] = true
		probeDir(dir)
	}

	fmt.Println("scanned", len(seen), "data dirs")
}

// probeDir runs the per-dir census with a 30s timeout and prints the
// report line when the dir was slow or the probe failed.
func probeDir(dir string) {
	dbPath := filepath.Join(dir, crushdata.DBName)
	if _, statErr := os.Stat(dbPath); statErr != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	start := time.Now()

	result := make(chan string, 1)

	go func() { result <- censusDir(ctx, dbPath) }()

	var report string

	select {
	case report = <-result:
	case <-ctx.Done():
		report = "TIMEOUT >30s"
	}

	elapsed := time.Since(start).Round(time.Millisecond)
	slow := elapsed > 2*time.Second
	failed := strings.HasPrefix(report, "TIMEOUT") || strings.HasPrefix(report, "query") ||
		strings.HasPrefix(report, "open") || strings.HasPrefix(report, "maxrow") ||
		strings.HasPrefix(report, "rowserr") || strings.HasPrefix(report, "scan-fail")
	if slow || failed {
		fmt.Printf("%-60s %-30s %s\n", dir, report, elapsed)
	}

	cancel()
}

// censusDir walks the newest messages.parts rows of one crush.db and
// returns a one-line report; the goroutine wrapper in main enforces the
// 30s timeout per dir.
func censusDir(ctx context.Context, dbPath string) string {
	db, openErr := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_txlock=immediate")
	if openErr != nil {
		return "open-fail " + openErr.Error()
	}

	db.SetMaxOpenConns(1)

	var maxRowID int64
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(rowid), 0) FROM messages").Scan(&maxRowID); err != nil {
		_ = db.Close()

		return "maxrow-fail " + err.Error()
	}

	rows, queryErr := db.QueryContext(ctx, `
		SELECT parts FROM messages
		WHERE rowid > ? AND parts IS NOT NULL AND parts != '[]'
		ORDER BY rowid DESC`, maxRowID-50000)
	if queryErr != nil {
		_ = db.Close()

		return "query-fail " + queryErr.Error()
	}

	bytesRead, entries, unparseable, rowCount := 0, 0, 0, 0

	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			_ = rows.Close()
			_ = db.Close()

			return "scan-fail " + err.Error()
		}

		bytesRead += len(raw)
		rowCount++

		var parsed []struct {
			Type string `json:"type"`
		}

		if err := json.Unmarshal(raw, &parsed); err != nil {
			unparseable++
		} else {
			entries += len(parsed)
		}

		if bytesRead >= 256<<20 {
			break
		}
	}

	rowsErr := rows.Err()
	_ = rows.Close()

	if rowsErr != nil {
		_ = db.Close()

		return "rowserr " + rowsErr.Error()
	}
	_ = db.Close()

	return fmt.Sprintf("rows=%d entries=%d unparseable=%d bytes=%dMB", rowCount, entries, unparseable, bytesRead>>20)
}
