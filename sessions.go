package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SessionFilter narrows [DB.Sessions]. The zero value returns every session,
// newest first.
type SessionFilter struct {
	// ByID restricts results to the session with this exact ID. Combine it
	// with nothing else; the zero value means no ID filter.
	ByID string

	// Day restricts results to sessions whose created_at falls on this
	// calendar day, compared as text in Day's own location against Crush's
	// UTC-stored timestamps. Pass a time in the zone you want the day
	// boundary drawn in (usually local midnight). The zero value means no
	// day filter; the clock component is ignored.
	Day time.Time

	// ParentID restricts results to children of this session ID. It cannot
	// be combined with RootOnly.
	ParentID string

	// RootOnly keeps only sessions without a parent (the sessions a user
	// started; auxiliary agent sessions are excluded). On schemas that lack
	// the parent_session_id column every session is a root, so the filter is
	// a no-op. It cannot be combined with ParentID.
	RootOnly bool

	// Limit caps the number of rows returned. Zero or negative means no
	// limit.
	Limit int
}

// Sessions lists sessions matching the filter, ordered by UpdatedAt
// descending. Filters are validated before any query runs: combining
// ParentID with RootOnly is a caller error.
func (db *DB) Sessions(ctx context.Context, filter SessionFilter) ([]Session, error) {
	if err := filter.validate(); err != nil {
		return nil, err
	}

	if filter.ParentID != "" && !db.schema.SessionsParentSessionID {
		return []Session{}, nil
	}

	rows, err := db.handle.QueryContext(ctx, db.buildSessionsQuery(filter), filter.args()...)
	if err != nil {
		return nil, fmt.Errorf("list sessions in %s: %w", db.path, err)
	}

	defer func() { _ = rows.Close() }()

	return scanSessions(rows)
}

// Session returns the session with the given ID, or [ErrSessionNotFound]
// wrapped with the ID when it does not exist.
func (db *DB) Session(ctx context.Context, id string) (Session, error) {
	//nolint:exhaustruct // the ID is the only relevant filter here
	rows, err := db.handle.QueryContext(ctx, db.buildSessionsQuery(SessionFilter{ByID: id}), id)
	if err != nil {
		return Session{}, fmt.Errorf("get session %s from %s: %w", id, db.path, err)
	}

	defer func() { _ = rows.Close() }()

	sessions, err := scanSessions(rows)
	if err != nil {
		return Session{}, err
	}

	if len(sessions) == 0 {
		return Session{}, fmt.Errorf("%w: %s in %s", ErrSessionNotFound, id, db.path)
	}

	return sessions[0], nil
}

func (f SessionFilter) validate() error {
	if f.ParentID != "" && f.RootOnly {
		return errConflictingFilter
	}

	return nil
}

func (f SessionFilter) args() []any {
	var args []any

	if !f.Day.IsZero() {
		args = append(args, f.Day.Format(time.DateOnly))
	}

	if f.ParentID != "" {
		args = append(args, f.ParentID)
	}

	if f.ByID != "" {
		args = append(args, f.ByID)
	}

	return args
}

// buildSessionsQuery constructs the sessions SELECT, substituting literal
// defaults for columns the database predates. Stable identifiers come from
// Crush's initial schema; cost and parent_session_id arrived in later
// migrations.
func (db *DB) buildSessionsQuery(filter SessionFilter) string {
	parentExpr := "NULL AS parent_session_id"
	if db.schema.SessionsParentSessionID {
		parentExpr = "parent_session_id"
	}

	costExpr := "0 AS cost"
	if db.schema.SessionsCost {
		costExpr = "cost"
	}

	query := fmt.Sprintf(
		"SELECT id, title, %s, message_count, prompt_tokens, completion_tokens, %s, updated_at, created_at, todos FROM sessions",
		parentExpr,
		costExpr,
	)

	var conditions []string

	if filter.ByID != "" {
		conditions = append(conditions, "id = ?")
	}

	if !filter.Day.IsZero() {
		conditions = append(conditions, "date(created_at, 'unixepoch') = ?")
	}

	if filter.ParentID != "" {
		conditions = append(conditions, "parent_session_id = ?")
	} else if filter.RootOnly && db.schema.SessionsParentSessionID {
		conditions = append(conditions, "parent_session_id IS NULL")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY updated_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	return query
}

// scanSessions lifts every row into a []Session.
func scanSessions(rows *sql.Rows) ([]Session, error) {
	var sessions []Session

	for rows.Next() {
		var (
			session       Session
			parent, todos sql.NullString
			createdAtUnix int64
			updatedAtUnix int64
		)

		err := rows.Scan(
			&session.ID,
			&session.Title,
			&parent,
			&session.MessageCount,
			&session.PromptTokens,
			&session.CompletionTokens,
			&session.CostUSD,
			&updatedAtUnix,
			&createdAtUnix,
			&todos,
		)
		if err != nil {
			return nil, fmt.Errorf("scan session row: %w", err)
		}

		session.ParentSessionID = parent.String
		session.CreatedAt = unixTime(createdAtUnix)
		session.UpdatedAt = unixTime(updatedAtUnix)
		session.Todos = todos.String

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session rows: %w", err)
	}

	return sessions, nil
}

// unixTime converts a Crush unix-seconds timestamp; non-positive values map
// to the zero time.
func unixTime(seconds int64) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}

	return time.Unix(seconds, 0).UTC()
}
