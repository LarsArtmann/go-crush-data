package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

		dbPath := filepath.Join(dir, "crush.db")
		if _, statErr := os.Stat(dbPath); statErr != nil {
			continue
		}

		db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_txlock=immediate")
		if err != nil {
			fmt.Println("open-fail", dir, err)
			continue
		}

		db.SetMaxOpenConns(1)

		var maxRow int64
		if err := db.QueryRow("SELECT COALESCE(MAX(rowid),0) FROM messages").Scan(&maxRow); err != nil {
			fmt.Println("query-fail", dir, err)
			_ = db.Close()
			continue
		}

		start := time.Now()

		rows, err := db.Query(`SELECT typeof(parts), COUNT(*), MAX(LENGTH(parts))
			FROM messages WHERE rowid > ? GROUP BY 1`, maxRow-2000)
		must(err)

		typeReport := ""
		for rows.Next() {
			var (
				typeName string
				count    int
				maxLen   sql.NullInt64
			)
			must(rows.Scan(&typeName, &count, &maxLen))
			typeReport += fmt.Sprintf(" %s=%d(maxLen=%d)", typeName, count, maxLen.Int64)
		}
		must(rows.Err())

		_ = rows.Close()

		var blobSample sql.NullString
		_ = db.QueryRow(`SELECT CAST(parts AS TEXT) FROM messages
			WHERE rowid > ? AND typeof(parts)='blob' LIMIT 1`, maxRow-2000).Scan(&blobSample)
		blobPrefix := ""
		if blobSample.Valid {
			s := blobSample.String
			if len(s) > 40 {
				s = s[:40]
			}
			blobPrefix = fmt.Sprintf(" blobSample=%q", s)
		}

		_ = db.Close()

		if typeReport != " text=2000(maxLen=0)" {
			fmt.Printf("%s%s%s took %s\n", filepath.Base(filepath.Dir(dir)), typeReport, blobPrefix, time.Since(start).Round(time.Millisecond))
		}
	}
	fmt.Println("scanned", len(seen), "data dirs")
}
