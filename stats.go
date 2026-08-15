package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// StatsFilter selects the activity [DB.Stats] aggregates. The zero value
// aggregates all time.
type StatsFilter struct {
	// Day restricts aggregation to sessions whose created_at falls on this
	// calendar day, compared as text in Day's own location against Crush's
	// UTC-stored timestamps (see [SessionFilter.Day]). The zero value means
	// no day filter; the clock component is ignored.
	Day time.Time
}

// Stats aggregates activity over the sessions selected by the filter.
//
// The SQL is kept verbatim from the first implementation of this logic
// (crush-daily's collector) so numbers are directly comparable across
// consumers: session-level aggregates (message_count, prompt_tokens,
// completion_tokens, cost) come from the sessions table, while the model
// breakdown joins messages to attribute sessions to models and counts
// message rows there — the two message counts are intentionally different.
func (db *DB) Stats(ctx context.Context, filter StatsFilter) (Stats, error) {
	day := ""
	if !filter.Day.IsZero() {
		day = filter.Day.Format(time.DateOnly)
	}

	stats, err := db.scanSessionStats(ctx, day)
	if err != nil {
		return Stats{}, err
	}

	if err := db.fillModelsAndProviders(ctx, day, &stats); err != nil {
		return Stats{}, err
	}

	if err := db.fillTitlesAndHistogram(ctx, day, &stats); err != nil {
		return Stats{}, err
	}

	stats.ModelBreakdown, err = db.scanModelBreakdown(ctx, day)

	return stats, err
}

func (db *DB) scanSessionStats(ctx context.Context, day string) (Stats, error) {
	costExpr := "0"
	if db.schema.SessionsCost {
		costExpr = "cost"
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(SUM(message_count), 0),
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(%s), 0)
		FROM sessions
	`, costExpr)

	var (
		stats Stats
		args  []any
	)

	if day != "" {
		query += " WHERE date(created_at, 'unixepoch') = ?"

		args = append(args, day)
	}

	err := db.handle.QueryRowContext(ctx, query, args...).Scan(
		&stats.SessionCount,
		&stats.MessageCount,
		&stats.PromptTokens,
		&stats.CompletionTokens,
		&stats.CostUSD,
	)
	if err != nil {
		return Stats{}, fmt.Errorf("aggregate session stats in %s: %w", db.path, err)
	}

	return stats, nil
}

// fillModelsAndProviders collects the distinct models and providers used
// across the selected sessions' messages.
func (db *DB) fillModelsAndProviders(ctx context.Context, day string, stats *Stats) error {
	if db.schema.MessagesModel {
		models, err := db.distinctMessageColumns(ctx, "model", day)
		if err != nil {
			return err
		}

		stats.Models = models
	}

	if !db.schema.MessagesProvider {
		return nil
	}

	providers, err := db.distinctMessageColumns(ctx, "provider", day)
	if err != nil {
		return err
	}

	stats.Providers = providers

	return nil
}

// distinctMessageColumns collects the non-empty distinct values of one
// messages column across the day's sessions.
func (db *DB) distinctMessageColumns(ctx context.Context, column, day string) ([]string, error) {
	dayFilter := ""
	args := []any{}

	if day != "" {
		dayFilter = " AND session_id IN (SELECT id FROM sessions WHERE date(created_at, 'unixepoch') = ?)"

		args = append(args, day)
	}

	//nolint:gosec // column comes from hardcoded call sites and dayFilter is a fixed literal, not user input
	query := fmt.Sprintf(
		"SELECT DISTINCT %s FROM messages WHERE %s IS NOT NULL AND %s != ''%s",
		column, column, column, dayFilter,
	)

	rows, err := db.handle.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate distinct %s in %s: %w", column, db.path, err)
	}

	return collectRows(rows, "distinct "+column, func(rows *sql.Rows) (string, error) {
		var value string
		if err := rows.Scan(&value); err != nil {
			return "", fmt.Errorf("scan distinct %s row: %w", column, err)
		}

		return value, nil
	})
}

func (db *DB) fillTitlesAndHistogram(ctx context.Context, day string, stats *Stats) error {
	dayFilter, args := dayArgs(day)

	//nolint:gosec // dayFilter is a fixed literal, not user input
	rows, err := db.handle.QueryContext(ctx, `
		SELECT title FROM sessions
		WHERE title IS NOT NULL`+dayFilter+`
		ORDER BY message_count DESC
		LIMIT 20
	`, args...)
	if err != nil {
		return fmt.Errorf("aggregate session titles in %s: %w", db.path, err)
	}

	titles, err := collectRows(rows, "session title", func(rows *sql.Rows) (string, error) {
		var title string
		if err := rows.Scan(&title); err != nil {
			return "", fmt.Errorf("scan session title row: %w", err)
		}

		return title, nil
	})
	if err != nil {
		return err
	}

	stats.SessionTitles = titles

	return db.fillHourHistogram(ctx, day, stats)
}

func (db *DB) fillHourHistogram(ctx context.Context, day string, stats *Stats) error {
	dayFilter, args := dayArgs(day)

	//nolint:gosec // dayFilter is a fixed literal, not user input
	rows, err := db.handle.QueryContext(ctx, `
		SELECT CAST(strftime('%H', created_at, 'unixepoch') AS INTEGER) AS hour, COUNT(*) AS count
		FROM sessions
		WHERE 1=1`+dayFilter+`
		GROUP BY hour
	`, args...)
	if err != nil {
		return fmt.Errorf("aggregate hour histogram in %s: %w", db.path, err)
	}

	buckets, err := collectRows(rows, "hour histogram", scanHourBucket)
	if err != nil {
		return err
	}

	for _, bucket := range buckets {
		if bucket.hour >= 0 && bucket.hour < len(stats.HourHistogram) {
			stats.HourHistogram[bucket.hour] = bucket.count
		}
	}

	return nil
}

// hourBucket is one hour-of-day aggregate of the session histogram.
type hourBucket struct {
	hour  int
	count int
}

func scanHourBucket(rows *sql.Rows) (hourBucket, error) {
	var bucket hourBucket
	if err := rows.Scan(&bucket.hour, &bucket.count); err != nil {
		return hourBucket{}, fmt.Errorf("scan hour histogram row: %w", err)
	}

	return bucket, nil
}

// dayArgs renders the shared created_at day filter. Empty day disables it.
func dayArgs(day string) (string, []any) {
	if day == "" {
		return "", nil
	}

	return " AND date(created_at, 'unixepoch') = ?", []any{day}
}

// scanModelBreakdown aggregates per-model usage. The query joins sessions
// on messages — one row per message — but session-level aggregates
// (prompt_tokens, completion_tokens, cost) are counted once per session via
// the GROUP BY in the per_session CTE. This is the join that naively counts
// sessions once per message if written wrong; the CTE shape is the fix and
// must be preserved.
func (db *DB) scanModelBreakdown(ctx context.Context, day string) ([]ModelStat, error) {
	if !db.schema.MessagesModel {
		return nil, nil
	}

	args := []any{}
	if day != "" {
		args = append(args, day)
	}

	rows, err := db.handle.QueryContext(ctx, db.buildModelBreakdownQuery(day), args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate model breakdown in %s: %w", db.path, err)
	}

	return collectRows(rows, "model breakdown", scanModelStat)
}

// scanModelStat lifts one breakdown row into a ModelStat.
func scanModelStat(rows *sql.Rows) (ModelStat, error) {
	var stat ModelStat

	err := rows.Scan(
		&stat.Model,
		&stat.SessionCount,
		&stat.MessageCount,
		&stat.PromptTokens,
		&stat.CompletionTokens,
		&stat.CostUSD,
	)
	if err != nil {
		return ModelStat{}, fmt.Errorf("scan model breakdown row: %w", err)
	}

	return stat, nil
}

func (db *DB) buildModelBreakdownQuery(day string) string {
	costExpr := "s.cost"
	if !db.schema.SessionsCost {
		costExpr = "0"
	}

	scope := ""
	if day != "" {
		scope = "s.id IN (SELECT id FROM sessions WHERE date(created_at, 'unixepoch') = ?) AND "
	}

	return fmt.Sprintf(`
		WITH per_session AS (
			SELECT
				m.model AS model,
				s.id AS session_id,
				s.prompt_tokens AS prompt_tokens,
				s.completion_tokens AS completion_tokens,
				%s AS cost,
				(SELECT COUNT(*) FROM messages m2 WHERE m2.session_id = s.id) AS messages
			FROM messages m
			JOIN sessions s ON s.id = m.session_id
			WHERE %sm.model IS NOT NULL AND m.model != ''
			GROUP BY m.model, s.id
		)
		SELECT
			model,
			COUNT(*) AS sessions,
			SUM(messages) AS messages,
			SUM(prompt_tokens) AS prompt_tokens,
			SUM(completion_tokens) AS completion_tokens,
			SUM(cost) AS cost
		FROM per_session
		GROUP BY model
		ORDER BY cost DESC
		LIMIT 20
	`, costExpr, scope)
}
