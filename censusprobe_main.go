package main

import (
	"context"
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		start := time.Now()

		result := make(chan string, 1)

		go func() {
			db, openErr := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_txlock=immediate")
			if openErr != nil {
				result <- "open-fail " + openErr.Error()
				return
			}

			db.SetMaxOpenConns(1)

			var maxRowID int64
			if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(rowid), 0) FROM messages").Scan(&maxRowID); err != nil {
				result <- "maxrow-fail " + err.Error()
				_ = db.Close()
				return
			}

			rows, queryErr := db.QueryContext(ctx, `
				SELECT parts FROM messages
				WHERE rowid > ? AND parts IS NOT NULL AND parts != '[]'
				ORDER BY rowid DESC`, maxRowID-50000)
			if queryErr != nil {
				result <- "query-fail " + queryErr.Error()
				_ = db.Close()
				return
			}

			bytesRead, entries, unparseable, rowCount := 0, 0, 0, 0
			for rows.Next() {
				var raw []byte
				if err := rows.Scan(&raw); err != nil {
					result <- "scan-fail " + err.Error()
					_ = rows.Close()
					_ = db.Close()
					return
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

			_ = rows.Close()
			_ = db.Close()

			result <- fmt.Sprintf("rows=%d entries=%d unparseable=%d bytes=%dMB", rowCount, entries, unparseable, bytesRead>>20)
		}()

		var report string
		select {
		case report = <-result:
		case <-ctx.Done():
			report = "TIMEOUT >30s"
		}

		elapsed := time.Since(start).Round(time.Millisecond)
		if elapsed > 2*time.Second || report[:7] == "TIMEOUT" || report[:5] == "query" || report[:4] == "open" || report[:6] == "maxrow" {
			fmt.Printf("%-60s %-30s %s\n", dir, report, elapsed)
		}

		cancel()
	}
	fmt.Println("scanned", len(seen), "data dirs")
}
