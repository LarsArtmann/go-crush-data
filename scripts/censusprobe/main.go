// censusprobe is a one-off scratch probe (concurrent-session census work):
// scans every registry data dir for messages.parts storage types (text vs
// blob) over the newest 2000 rows, reporting anything that is not pure
// text — a canary for upstream parts-format drift (e.g. compression).
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
	raw, err := os.ReadFile("/home/lars/.local/share/crush/projects.json")
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
		probe(dir)
	}

	fmt.Println("scanned", len(seen), "data dirs")
}

// probe reports on one data dir; defers are scoped per call so file
// handles do not accumulate across the registry walk.
func probe(dir string) {
	dbPath := filepath.Join(dir, "crush.db")
	if _, statErr := os.Stat(dbPath); statErr != nil {
		return
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_txlock=immediate")
	if err != nil {
		fmt.Println("open-fail", dir, err)

		return
	}

	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()

	var maxRow int64
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(rowid),0) FROM messages").Scan(&maxRow); err != nil {
		fmt.Println("query-fail", dir, err)

		return
	}

	start := time.Now()

	rows, err := db.QueryContext(ctx, `SELECT typeof(parts), COUNT(*), MAX(LENGTH(parts))
		FROM messages WHERE rowid > ? GROUP BY 1`, maxRow-2000)
	must(err)

	defer func() { _ = rows.Close() }()

	var typeReport strings.Builder

	for rows.Next() {
		var (
			typeName string
			count    int
			maxLen   sql.NullInt64
		)

		must(rows.Scan(&typeName, &count, &maxLen))
		fmt.Fprintf(&typeReport, " %s=%d(maxLen=%d)", typeName, count, maxLen.Int64)
	}
	must(rows.Err())

	var blobSample sql.NullString
	_ = db.QueryRowContext(ctx, `SELECT CAST(parts AS TEXT) FROM messages
		WHERE rowid > ? AND typeof(parts)='blob' LIMIT 1`, maxRow-2000).Scan(&blobSample)

	blobPrefix := ""
	if blobSample.Valid {
		s := blobSample.String
		if len(s) > 40 {
			s = s[:40]
		}

		blobPrefix = fmt.Sprintf(" blobSample=%q", s)
	}

	if typeReport.String() != " text=2000(maxLen=0)" {
		fmt.Printf("%s%s%s took %s\n", filepath.Base(filepath.Dir(dir)), typeReport.String(), blobPrefix,
			time.Since(start).Round(time.Millisecond))
	}
}
