package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "file:"+os.Args[1]+"?mode=ro")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&total); err != nil {
		panic(err)
	}

	var inversions int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM messages m1
		JOIN messages m2 ON m2.rowid = (
			SELECT MIN(m3.rowid) FROM messages m3
			WHERE m3.session_id = m1.session_id AND m3.rowid > m1.rowid
		)
		WHERE m2.created_at < m1.created_at
	`).Scan(&inversions); err != nil {
		panic(err)
	}

	var ties int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT session_id, created_at, COUNT(*) c FROM messages
			GROUP BY session_id, created_at HAVING c > 1
		)
	`).Scan(&ties); err != nil {
		panic(err)
	}

	fmt.Printf("messages=%d same-second-groups=%d adjacent-inversions=%d\n", total, ties, inversions)
}
