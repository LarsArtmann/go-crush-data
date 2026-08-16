package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"slices"
)

// Messages returns one session's messages ordered by creation time (and ID
// as the tiebreaker, since timestamps have second precision), with their
// parts decoded into typed [Part] values.
//
// Parts decoding is tolerant: a single malformed part degrades to
// [UnknownPart] carrying its discriminator and raw payload, so the
// well-formed siblings survive, and a message whose parts JSON is empty or
// wholly unparseable gets nil Parts — either way one corrupted row never
// hides the rest of the session. Callers needing strict all-or-nothing
// validation can use [DecodeParts] on the raw column via their own query.
func (db *DB) Messages(ctx context.Context, sessionID string) ([]Message, error) {
	rows, err := db.handle.QueryContext(ctx, db.buildMessagesQuery(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("read messages of session %s from %s: %w", sessionID, db.path, err)
	}

	return collectRows(rows, "message", func(rows *sql.Rows) (Message, error) {
		return scanMessage(rows, sessionID)
	})
}

// IterMessages returns the same messages as [DB.Messages], in the same
// order, as an iterator for sessions too large to materialize as one
// slice:
//
//	for message, err := range db.IterMessages(ctx, sessionID) {
//		if err != nil {
//			break // handle err; iteration stops here
//		}
//		// ...
//	}
//
// Because an iterator cannot return an error up front, failures — a failed
// query, a canceled context, a scan error, or an iteration error — are
// yielded as (Message{}, err) and iteration stops. Breaking out of the
// loop stops the iteration and releases the underlying rows. Each range
// over the returned sequence runs the query anew, so storing it and
// ranging twice reads the session twice.
func (db *DB) IterMessages(ctx context.Context, sessionID string) iter.Seq2[Message, error] {
	return func(yield func(Message, error) bool) {
		rows, err := db.handle.QueryContext(ctx, db.buildMessagesQuery(), sessionID)
		if err != nil {
			//nolint:exhaustruct // error yield: the message is intentionally zero
			yield(Message{}, fmt.Errorf("read messages of session %s from %s: %w", sessionID, db.path, err))

			return
		}

		defer func() { _ = rows.Close() }()

		for rows.Next() {
			message, err := scanMessage(rows, sessionID)
			if err != nil {
				//nolint:exhaustruct // error yield: the message is intentionally zero
				yield(Message{}, err)

				return
			}

			if !yield(message, nil) {
				return
			}
		}

		if err := rows.Err(); err != nil {
			//nolint:exhaustruct // error yield: the message is intentionally zero
			yield(Message{}, fmt.Errorf("iterate message rows: %w", err))
		}
	}
}

// scanMessage lifts one row into a Message, decoding its parts tolerantly
// — the shared row logic of [DB.Messages] and [DB.IterMessages].
func scanMessage(rows *sql.Rows, sessionID string) (Message, error) {
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
		return Message{}, fmt.Errorf("scan message row: %w", err)
	}

	message.SessionID = sessionID
	message.Model = model.String
	message.Provider = provider.String
	message.CreatedAt = unixTime(createdAtUnix)
	message.FinishedAt = unixTime(finishedAt.Int64)

	decoded, err := decodeParts(parts, false)
	if err != nil {
		decoded = nil
	}

	message.Parts = decoded

	return message, nil
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

	values, err := collectRows(rows, "read_files", func(rows *sql.Rows) (string, error) {
		var path string
		if err := rows.Scan(&path); err != nil {
			return "", fmt.Errorf("scan read_files row: %w", err)
		}

		return path, nil
	})
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(values, func(path string) bool { return path == "" }), nil
}
