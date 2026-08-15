package crushdata

import (
	"context"
	"database/sql"
	"fmt"
)

// Messages returns one session's messages ordered by creation time (and ID
// as the tiebreaker, since timestamps have second precision), with their
// parts decoded into typed [Part] values.
//
// A message whose parts JSON is empty or malformed is returned with nil
// Parts instead of failing the read: one corrupted row must not hide the
// rest of the session. Callers needing strict validation can use
// [DecodeParts] on the raw column via their own query.
func (db *DB) Messages(ctx context.Context, sessionID string) ([]Message, error) {
	rows, err := db.handle.QueryContext(ctx, db.buildMessagesQuery(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("read messages of session %s from %s: %w", sessionID, db.path, err)
	}

	defer func() { _ = rows.Close() }()

	var messages []Message

	for rows.Next() {
		var (
			message         Message
			parts           string
			model, provider sql.NullString
			createdAtUnix   int64
			finishedAt      sql.NullInt64
		)

		err := rows.Scan(
			&message.ID,
			&message.Role,
			&parts,
			&model,
			&provider,
			&createdAtUnix,
			&finishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}

		message.SessionID = sessionID
		message.Model = model.String
		message.Provider = provider.String
		message.CreatedAt = unixTime(createdAtUnix)
		message.FinishedAt = unixTime(finishedAt.Int64)

		decoded, err := DecodeParts(parts)
		if err != nil {
			decoded = nil
		}

		message.Parts = decoded

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message rows: %w", err)
	}

	return messages, nil
}

// buildMessagesQuery constructs the messages SELECT, substituting NULL for
// columns the database predates.
func (db *DB) buildMessagesQuery() string {
	modelExpr := "NULL AS model"
	if db.schema.MessagesModel {
		modelExpr = "model"
	}

	providerExpr := "NULL AS provider"
	if db.schema.MessagesProvider {
		providerExpr = "provider"
	}

	finishedExpr := "NULL AS finished_at"
	if db.schema.MessagesFinishedAt {
		finishedExpr = "finished_at"
	}

	return fmt.Sprintf(
		"SELECT id, role, parts, %s, %s, created_at, %s FROM messages WHERE session_id = ? ORDER BY created_at, id",
		modelExpr, providerExpr, finishedExpr,
	)
}

// ReadFiles returns the paths the agent actually opened during a session,
// according to the read_files table. Databases that predate the table return
// an empty slice.
func (db *DB) ReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	if !db.schema.ReadFilesTable {
		return []string{}, nil
	}

	rows, err := db.handle.QueryContext(
		ctx,
		"SELECT path FROM read_files WHERE session_id = ?",
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("read files of session %s from %s: %w", sessionID, db.path, err)
	}

	defer func() { _ = rows.Close() }()

	var paths []string

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan read_files row: %w", err)
		}

		if path != "" {
			paths = append(paths, path)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate read_files rows: %w", err)
	}

	return paths, nil
}
