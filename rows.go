package crushdata

import (
	"database/sql"
	"fmt"
)

// collectRows gathers one T per row via scanRow and verifies iteration ran
// to completion; rows is closed on return. what names the row set in the
// iteration error so a failure says which query ended early.
func collectRows[T any](rows *sql.Rows, what string, scanRow func(*sql.Rows) (T, error)) ([]T, error) {
	defer func() { _ = rows.Close() }()

	var values []T

	for rows.Next() {
		value, err := scanRow(rows)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s rows: %w", what, err)
	}

	return values, nil
}
