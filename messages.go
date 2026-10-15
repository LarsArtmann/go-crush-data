package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"slices"
)

// Messages returns one session's messages in insertion order (rowid —
// the order Crush wrote the rows), with their parts decoded into typed
// [Part] values.
//
// Insertion order is also the correct tiebreak for the second-precision
// created_at timestamps: Crush generates message IDs with uuid.New()
// (UUIDv4, random), so ordering by (created_at, id) would be deterministic
// but arbitrary within a second — a same-second tool_result could sort
// before its tool_call. rowid is the insertion counter, unique, and
// inversion-free against created_at on real databases.
//
// Parts decoding is tolerant: a single malformed part degrades to
// [UnknownPart] carrying its discriminator and raw payload, so the
// well-formed siblings survive, and a message whose parts JSON is empty or
// wholly unparseable gets nil Parts — either way one corrupted row never
// hides the rest of the session. Callers needing strict all-or-nothing
// validation can use [DecodeParts] on the raw column via their own query.
//
// Summary messages (is_summary_message = 1, written when Crush compacts a
// session's context into a replacement summary) are returned like any other
// row: they carry real content and model attribution, and filtering them
// would break number parity with the collector SQL this library ports
// verbatim. Their [Message.IsSummaryMessage] flag (and the session's
// [Session.SummaryMessageID] pointer) lets consumers skip or specially
// render them. Pinned by TestSummaryMessagesAreCounted.
func (db *DB) Messages(ctx context.Context, sessionID string) ([]Message, error) {
	return db.readMessages(ctx, sessionID, db.buildMessagesQuery(), sessionID)
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
	return db.streamMessages(ctx, sessionID, db.buildMessagesQuery(), sessionID)
}

// MessagesSince returns the session's messages inserted after afterRowID, in
// insertion order — the keyset page an incremental consumer reads once it has
// handled everything up to a cursor. afterRowID 0 returns the whole session,
// row for row what [DB.Messages] returns; the RowID of the last message a
// consumer handled returns exactly what Crush appended since.
//
// The cursor is a rowid and not a timestamp, deliberately. A
// `created_at > watermark` filter loses data permanently: created_at holds
// whole seconds, so every message written in the cursor's own second — the
// common case in a live session, where a poll resumes mid-second — is
// excluded, and because the cursor only moves forward those rows are never
// read again. A rowid is unique and monotonic, so a strict `rowid > cursor`
// can neither skip a row nor repeat one, and it is the same column the
// ordering uses, so the page boundary and the order cannot disagree.
// [MessageRowID] spells the distinction out in the type system.
//
// Cursors are per session: a rowid is unique per database, but the cursor
// only means anything next to the session it was read from. Consumers that
// withhold rows — holding back a tool call whose result has not arrived, for
// instance — resume from the rowid of the last message they released rather
// than the last one they read; the withheld rows come back on the next read,
// which is what makes the withholding safe instead of lossy.
//
// Parts decoding, summary-message handling, and tolerance toward malformed
// rows are exactly as documented on [DB.Messages].
func (db *DB) MessagesSince(ctx context.Context, sessionID string, afterRowID MessageRowID) ([]Message, error) {
	return db.readMessages(ctx, sessionID, db.buildMessagesSinceQuery(), sessionID, afterRowID)
}

// IterMessagesSince returns the same rows as [DB.MessagesSince], in the same
// order, as an iterator for pages too large to materialize as one slice:
//
//	for message, err := range db.IterMessagesSince(ctx, sessionID, cursor) {
//		if err != nil {
//			break // handle err; iteration stops here
//		}
//		// ...
//	}
//
// Its failure and early-break behavior is [DB.IterMessages]'s: failures are
// yielded as (Message{}, err) and stop the iteration, breaking out releases
// the rows, and each range runs the query anew.
func (db *DB) IterMessagesSince(
	ctx context.Context,
	sessionID string,
	afterRowID MessageRowID,
) iter.Seq2[Message, error] {
	return db.streamMessages(ctx, sessionID, db.buildMessagesSinceQuery(), sessionID, afterRowID)
}

// LastMessageRowID returns the rowid of the session's newest message — the
// cursor that means "everything written so far is accounted for", for a
// consumer that starts tailing a session whose history it does not want to
// replay. Pair it with [DB.MessagesSince]; 0 means the session has no
// messages yet, and reads as "no history" there too.
//
// It is a rowid, so it is exact where a timestamp watermark is not: the value
// moves only when Crush appends a row, and no two rows share it. The newest
// row may still be one Crush is streaming, though — the cursor is where the
// writer is, not a promise that the turn is complete. A consumer that must
// not act on a partial turn holds its own cursor at the last row it considers
// complete; rowids already written never move, so the difference is a re-read,
// not a gap.
//
// A session ID that does not exist reads 0 rather than failing, matching
// [DB.Messages], which returns an empty slice for it. The read is a covering
// index seek on messages(session_id) — confirmed with EXPLAIN QUERY PLAN on a
// real database — so holding one cursor per session costs an index lookup,
// not a scan of the session's rows.
func (db *DB) LastMessageRowID(ctx context.Context, sessionID string) (MessageRowID, error) {
	var rowID int64

	err := db.handle.QueryRowContext(ctx, lastMessageRowIDQuery, sessionID).Scan(&rowID)
	if err != nil {
		return 0, fmt.Errorf("read the newest message rowid of session %s from %s: %w", sessionID, db.path, err)
	}

	return MessageRowID(rowID), nil
}

// readMessages runs one messages query and lifts every row into a Message;
// sessionID attributes the rows and names them in the error. The query and
// its bind arguments always travel together, so a page's WHERE can never
// drift from the args it was built for.
func (db *DB) readMessages(
	ctx context.Context,
	sessionID, query string,
	args ...any,
) ([]Message, error) {
	rows, err := db.handle.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read messages of session %s from %s: %w", sessionID, db.path, err)
	}

	return collectRows(rows, "message", func(rows *sql.Rows) (Message, error) {
		return scanMessage(rows, sessionID)
	})
}

// streamMessages is [DB.readMessages] as an iterator: the shared body of
// [DB.IterMessages] and [DB.IterMessagesSince].
func (db *DB) streamMessages(
	ctx context.Context,
	sessionID, query string,
	args ...any,
) iter.Seq2[Message, error] {
	return func(yield func(Message, error) bool) {
		rows, err := db.handle.QueryContext(ctx, query, args...)
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
		updatedAtUnix   int64
		finishedAt      sql.NullInt64
		isSummary       sql.NullInt64
	)

	err := rows.Scan(
		&message.ID,
		&message.Role,
		&parts,
		&model,
		&provider,
		&createdAtUnix,
		&updatedAtUnix,
		&finishedAt,
		&isSummary,
	)
	if err != nil {
		return Message{}, fmt.Errorf("scan message row: %w", err)
	}

	message.SessionID = sessionID
	message.Model = model.String
	message.Provider = provider.String
	message.CreatedAt = unixTime(createdAtUnix)
	message.UpdatedAt = unixTime(updatedAtUnix)
	message.FinishedAt = unixTime(finishedAt.Int64)
	message.IsSummaryMessage = isSummary.Int64 != 0

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

	isSummaryExpr := "0 AS is_summary_message"
	if db.schema.MessagesIsSummaryMessage {
		isSummaryExpr = "is_summary_message"
	}

	return fmt.Sprintf(
		"SELECT id, role, parts, %s, %s, created_at, updated_at, %s, %s FROM messages WHERE session_id = ? ORDER BY rowid",
		modelExpr,
		providerExpr,
		finishedExpr,
		isSummaryExpr,
	)
}

// ReadFiles returns the paths the agent actually opened during a session,
// most recently read first — the same ordering upstream's read_files listing
// uses (ORDER BY read_at DESC). Databases that predate the table return an
// empty slice.
func (db *DB) ReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	if !db.schema.ReadFilesTable {
		return []string{}, nil
	}

	rows, err := db.handle.QueryContext(
		ctx,
		"SELECT path FROM read_files WHERE session_id = ? ORDER BY read_at DESC",
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
