package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func census(db *sql.DB, label, query string, args ...any) {
	start := time.Now()

	rows, err := db.Query(query, args...)
	must(err)

	entries := 0

	for rows.Next() {
		var (
			kind  string
			count int
		)

		must(rows.Scan(&kind, &count))
		entries += count
		fmt.Printf("  %-16s %d\n", kind, count)
	}

	must(rows.Err())
	fmt.Printf("%s: %d entries in %s\n\n", label, entries, time.Since(start))
}

func main() {
	db, err := sql.Open("sqlite", "file:/home/lars/projects/.crush/crush.db?mode=ro&_txlock=immediate")
	must(err)

	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	var (
		total      int
		minTsSql   sql.NullInt64
		maxTsSql   sql.NullInt64
	)
	must(db.QueryRow("SELECT COUNT(*), MIN(created_at), MAX(created_at) FROM messages").Scan(&total, &minTsSql, &maxTsSql))
	fmt.Println("messages:", total, "created_at range:", minTsSql.Int64, "-", maxTsSql.Int64,
		"(", time.Unix(minTsSql.Int64, 0).UTC().Format(time.DateOnly), "→", time.Unix(maxTsSql.Int64, 0).UTC().Format(time.DateOnly), ")")

	cutoff := time.Now().AddDate(0, 0, -90).Unix()

	var recent int
	must(db.QueryRow("SELECT COUNT(*) FROM messages WHERE created_at >= ?", cutoff).Scan(&recent))
	fmt.Println("messages in last 90d:", recent)

	census(db, "rowid<=5000 census",
		`SELECT COALESCE(json_extract(entry.value,'$.type'),''), COUNT(*)
		 FROM messages, json_each(messages.parts) AS entry
		 WHERE messages.rowid <= 5000 AND json_valid(messages.parts) GROUP BY 1`)

	census(db, "90-day-window census",
		`SELECT COALESCE(json_extract(entry.value,'$.type'),''), COUNT(*)
		 FROM messages, json_each(messages.parts) AS entry
		 WHERE messages.created_at >= ? AND json_valid(messages.parts) GROUP BY 1`, cutoff)
}
