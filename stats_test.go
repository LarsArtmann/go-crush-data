package crushdata

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestStatsDay(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay()})
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if stats.SessionCount != 2 {
		t.Fatalf("SessionCount = %d, want 2", stats.SessionCount)
	}

	if stats.MessageCount != 11 {
		t.Fatalf("MessageCount = %d, want 11 (sum of session message_count)", stats.MessageCount)
	}

	if stats.PromptTokens != 5000 || stats.CompletionTokens != 1500 {
		t.Fatalf("tokens = %d/%d, want 5000/1500", stats.PromptTokens, stats.CompletionTokens)
	}

	if stats.CostUSD != 0.0234 {
		t.Fatalf("CostUSD = %v, want 0.0234", stats.CostUSD)
	}

	if len(stats.Models) != 1 || stats.Models[0] != "minimax/minimax-m3" {
		t.Fatalf("Models = %v", stats.Models)
	}

	if len(stats.Providers) != 0 {
		t.Fatalf("Providers = %v, want none in fixture", stats.Providers)
	}

	if len(stats.SessionTitles) != 2 {
		t.Fatalf("SessionTitles = %v", stats.SessionTitles)
	}

	// fixtureBase is 2026-08-04T12:00 UTC → hour 12.
	if stats.HourHistogram[12] != 2 {
		t.Fatalf("HourHistogram[12] = %d, want 2", stats.HourHistogram[12])
	}
}

func TestStatsOtherDayIsEmpty(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay().AddDate(0, 0, 3)})
	if err != nil {
		t.Fatal(err)
	}

	if stats.SessionCount != 0 || stats.MessageCount != 0 || stats.CostUSD != 0 {
		t.Fatalf("stats = %+v, want zero", stats)
	}

	if len(stats.ModelBreakdown) != 0 {
		t.Fatalf("ModelBreakdown = %+v, want none", stats.ModelBreakdown)
	}
}

func TestStatsAllTime(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	stats, err := db.Stats(context.Background(), StatsFilter{})
	if err != nil {
		t.Fatal(err)
	}

	if stats.SessionCount != 2 {
		t.Fatalf("SessionCount = %d, want 2 (no day filter)", stats.SessionCount)
	}

	if len(stats.ModelBreakdown) != 1 || stats.ModelBreakdown[0].Model != "minimax/minimax-m3" {
		t.Fatalf("ModelBreakdown = %+v, want one model across all time", stats.ModelBreakdown)
	}
}

func TestStatsModelBreakdownDoubleCountTrap(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "s1", "", "One model, many messages", 4, fixtureBase, fixtureBase+9)
		insertMessage(t, db, "m1", "s1", "assistant", `[]`, "model-a", "prov", fixtureBase)
		insertMessage(t, db, "m2", "s1", "assistant", `[]`, "model-a", "prov", fixtureBase+1)
		insertMessage(t, db, "m3", "s1", "assistant", `[]`, "model-a", "prov", fixtureBase+2)
		// A message without a model must not drag the session into the
		// breakdown (its tokens belong to no model attribution).
		insertMessage(t, db, "m4", "s1", "user", `[]`, "", "", fixtureBase+3)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay()})
	if err != nil {
		t.Fatal(err)
	}

	if len(stats.ModelBreakdown) != 1 {
		t.Fatalf("ModelBreakdown = %+v, want exactly one model", stats.ModelBreakdown)
	}

	breakdown := stats.ModelBreakdown[0]
	if breakdown.SessionCount != 1 {
		t.Fatalf("SessionCount = %d, want 1 — sessions must not be counted once per message", breakdown.SessionCount)
	}

	if breakdown.MessageCount != 4 {
		t.Fatalf(
			"MessageCount = %d, want 4 (every message of the attributed session, matching the historical subquery)",
			breakdown.MessageCount,
		)
	}
}

// assertCrushDailyCollectorParity requires this library's Stats numbers to
// match the exact SQL crush-daily's collector ran before this library
// existed, re-run raw against the same database. The queries are copied
// verbatim (modulo formatting) from crush-daily internal/collector/
// collector.go so a refactor here can never silently change the analytics.
// Shared by the fixture parity test and the real-data cross-check.
func assertCrushDailyCollectorParity(t *testing.T, db *DB, day time.Time) {
	t.Helper()

	stats, err := db.Stats(context.Background(), StatsFilter{Day: day})
	if err != nil {
		t.Fatal(err)
	}

	dayText := day.Format("2006-01-02")

	var (
		sessionCount     int
		messageCount     int
		promptTokens     int64
		completionTokens int64
		costUSD          float64
	)

	err = db.handle.QueryRowContext(context.Background(), `
		SELECT
			COUNT(*),
			COALESCE(SUM(message_count), 0),
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(cost), 0)
		FROM sessions
		WHERE date(created_at, 'unixepoch') = ?
	`, dayText).Scan(&sessionCount, &messageCount, &promptTokens, &completionTokens, &costUSD)
	if err != nil {
		t.Fatal(err)
	}

	if stats.SessionCount != sessionCount || stats.MessageCount != messageCount ||
		stats.PromptTokens != promptTokens || stats.CompletionTokens != completionTokens ||
		stats.CostUSD != costUSD {
		t.Fatalf(
			"session aggregates diverge: got %+v want %d/%d/%d/%d/%v",
			stats, sessionCount, messageCount, promptTokens, completionTokens, costUSD,
		)
	}

	models := collectStrings(t, db, `
		SELECT DISTINCT model FROM messages
		WHERE session_id IN (
			SELECT id FROM sessions WHERE date(created_at, 'unixepoch') = ?
		) AND model IS NOT NULL AND model != ''
	`, dayText)

	if len(models) != len(stats.Models) {
		t.Fatalf("models = %v, stats.Models = %v", models, stats.Models)
	}

	titles := collectStrings(t, db, `
		SELECT title FROM sessions
		WHERE date(created_at, 'unixepoch') = ? AND title IS NOT NULL
		ORDER BY message_count DESC
		LIMIT 20
	`, dayText)

	if len(titles) != len(stats.SessionTitles) {
		t.Fatalf("titles = %v, stats.SessionTitles = %v", titles, stats.SessionTitles)
	}

	breakdownRows, err := db.handle.QueryContext(context.Background(), `
		WITH per_session AS (
			SELECT
				m.model AS model,
				s.id AS session_id,
				s.prompt_tokens AS prompt_tokens,
				s.completion_tokens AS completion_tokens,
				s.cost AS cost,
				(SELECT COUNT(*) FROM messages m2 WHERE m2.session_id = s.id) AS messages
			FROM messages m
			JOIN sessions s ON s.id = m.session_id
			WHERE s.id IN (
				SELECT id FROM sessions WHERE date(created_at, 'unixepoch') = ?
			)
			AND m.model IS NOT NULL AND m.model != ''
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
	`, dayText)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = breakdownRows.Close() }()

	var legacy []ModelStat

	for breakdownRows.Next() {
		var stat ModelStat

		err := breakdownRows.Scan(
			&stat.Model, &stat.SessionCount, &stat.MessageCount,
			&stat.PromptTokens, &stat.CompletionTokens, &stat.CostUSD,
		)
		if err != nil {
			t.Fatal(err)
		}

		legacy = append(legacy, stat)
	}

	if err := breakdownRows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(legacy) != len(stats.ModelBreakdown) {
		t.Fatalf("breakdown rows = %d, stats.ModelBreakdown = %d", len(legacy), len(stats.ModelBreakdown))
	}

	for i := range legacy {
		if legacy[i] != stats.ModelBreakdown[i] {
			t.Fatalf("breakdown[%d] = %+v, stats = %+v", i, legacy[i], stats.ModelBreakdown[i])
		}
	}
}

// TestStatsParityWithCrushDailySQL runs the collector-parity assertion
// against the standard fixture; TestStatsParityWithCrushDailySQLOnRealDatabase
// runs it against live data.
func TestStatsParityWithCrushDailySQL(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	assertCrushDailyCollectorParity(t, db, fixtureDay())
}

// TestStatsParityWithCrushDailySQLOnRealDatabase is the real-data half of the
// consumer contract behind the verbatim-SQL rule: this library's Stats and
// the old collector SQL must produce identical numbers on a live database,
// not just on fixtures. Day-filtered on purpose and pinned to the day of the
// newest session so the comparison is never vacuous (all-time Stats
// DISTINCTs every message row and starves on production-sized databases —
// see TestAllAPIOnRealDatabase). Skipped without real data and under -short.
func TestStatsParityWithCrushDailySQLOnRealDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("real-data sweep (slow) in -short mode")
	}

	t.Parallel()

	dataDir, ok := realDataDir(t)
	if !ok {
		t.Skip("no real data dir (set CRUSH_DATA_REAL_DATA_DIR to run against one)")
	}

	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open real database: %v", err)
	}

	defer func() { _ = db.Close() }()

	sessions, err := db.Sessions(context.Background(), SessionFilter{Limit: 1})
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Skip("real database holds no sessions — nothing to compare")
	}

	assertCrushDailyCollectorParity(t, db, sessions[0].CreatedAt)
}

func collectStrings(t *testing.T, db *DB, query string, args ...any) []string {
	t.Helper()

	rows, err := db.handle.QueryContext(context.Background(), query, args...)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = rows.Close() }()

	var values []string

	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}

		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return values
}

func TestStatsOnLegacySchema(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay()})
	if err != nil {
		t.Fatalf("Stats on legacy schema: %v", err)
	}

	if stats.SessionCount != 2 || stats.MessageCount != 11 {
		t.Fatalf("counts = %+v", stats)
	}

	if stats.CostUSD != 0 {
		t.Fatalf("CostUSD = %v, want 0 (column absent)", stats.CostUSD)
	}

	if len(stats.Models) != 0 || len(stats.Providers) != 0 {
		t.Fatalf("models/providers = %v/%v, want none (columns absent)", stats.Models, stats.Providers)
	}

	if len(stats.ModelBreakdown) != 0 {
		t.Fatalf("ModelBreakdown = %+v, want none (model column absent)", stats.ModelBreakdown)
	}
}

// TestStatsOnLegacySchemaOtherDayIsEmpty pins that the day filter works on
// legacy schema: a day with no sessions returns zero stats, not an error
// or a fallback to all sessions.
func TestStatsOnLegacySchemaOtherDayIsEmpty(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay().AddDate(0, 0, 3)})
	if err != nil {
		t.Fatalf("Stats on legacy schema other day: %v", err)
	}

	if stats.SessionCount != 0 || stats.MessageCount != 0 || stats.CostUSD != 0 {
		t.Fatalf("stats = %+v, want zero on empty day", stats)
	}

	if len(stats.SessionTitles) != 0 || len(stats.ModelBreakdown) != 0 {
		t.Fatalf("titles/breakdown = %v/%v, want none on empty day", stats.SessionTitles, stats.ModelBreakdown)
	}
}

// TestStatsDayFilterUsesFilterLocation pins the day-filter semantics: the
// day string is rendered in the filter value's own location and compared as
// text against the UTC-stored created_at date. A zone whose calendar date
// differs from the UTC date near midnight therefore matches differently —
// the historical crush-daily semantics this library preserves.
func TestStatsDayFilterUsesFilterLocation(t *testing.T) {
	t.Parallel()

	// 2026-08-04T23:30:00Z: UTC calendar day 08-04, but 08-05 in UTC+2.
	lateUnix := fixtureBase + 11*3600 + 30*60

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "late", "", "Late session", 1, lateUnix, lateUnix)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	utcDay := time.Unix(lateUnix, 0).UTC()

	utcStats, err := db.Stats(context.Background(), StatsFilter{Day: utcDay})
	if err != nil {
		t.Fatalf("Stats (UTC day): %v", err)
	}

	if utcStats.SessionCount != 1 {
		t.Fatalf("SessionCount (UTC day) = %d, want 1", utcStats.SessionCount)
	}

	plusTwo := time.Unix(lateUnix, 0).In(time.FixedZone("UTC+2", 2*3600))
	if plusTwo.Format(time.DateOnly) == utcDay.Format(time.DateOnly) {
		t.Fatal("fixture error: zone shift must change the calendar date")
	}

	zoneStats, err := db.Stats(context.Background(), StatsFilter{Day: plusTwo})
	if err != nil {
		t.Fatalf("Stats (UTC+2 day): %v", err)
	}

	if zoneStats.SessionCount != 0 {
		t.Fatalf(
			"SessionCount (UTC+2 day) = %d, want 0: the day string is the zone's calendar date, compared against the UTC date",
			zoneStats.SessionCount,
		)
	}
}

// TestApplyHourBucketsDropsOutOfRangeHours pins the histogram guard: hours
// outside [0, 24) must be ignored rather than panic, since a corrupted
// aggregate cannot be allowed to take down a whole Stats read.
func TestApplyHourBucketsDropsOutOfRangeHours(t *testing.T) {
	t.Parallel()

	var histogram [24]int

	applyHourBuckets(&histogram, []hourBucket{
		{hour: 0, count: 3},
		{hour: 12, count: 5},
		{hour: 23, count: 7},
		{hour: -1, count: 99},
		{hour: 24, count: 99},
		{hour: 100, count: 99},
	})

	if histogram[0] != 3 || histogram[12] != 5 || histogram[23] != 7 {
		t.Fatalf("histogram = %v, want in-range buckets written", histogram)
	}

	for _, hour := range []int{1, 11, 13, 22} {
		if histogram[hour] != 0 {
			t.Fatalf("histogram[%d] = %d, want 0", hour, histogram[hour])
		}
	}
}

// TestStatsModelsProvidersOrderedAscending pins that Models and Providers
// are returned in ascending alphabetical order (ORDER BY was added to make
// the DISTINCT output deterministic; previously the order was undefined).
func TestStatsModelsProvidersOrderedAscending(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "multi-model-session", "", "Multi-model", 5, fixtureBase, fixtureBase)

		models := []string{"zeta/large", "alpha/small", "mid/medium", "alpha/small", "zeta/large"}
		providers := []string{"openrouter", "anthropic", "openai", "anthropic", "openrouter"}

		for i, msg := range fixtureMessages[:5] {
			insertMessage(t, db, msg.id, "multi-model-session", msg.role, msg.parts,
				models[i], providers[i], fixtureBase+int64(i))
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay()})
	if err != nil {
		t.Fatal(err)
	}

	wantModels := []string{"alpha/small", "mid/medium", "zeta/large"}
	if len(stats.Models) != len(wantModels) {
		t.Fatalf("Models = %v, want %v", stats.Models, wantModels)
	}

	for i, want := range wantModels {
		if stats.Models[i] != want {
			t.Fatalf("Models[%d] = %q, want %q (ascending)", i, stats.Models[i], want)
		}
	}

	wantProviders := []string{"anthropic", "openai", "openrouter"}
	if len(stats.Providers) != len(wantProviders) {
		t.Fatalf("Providers = %v, want %v", stats.Providers, wantProviders)
	}

	for i, want := range wantProviders {
		if stats.Providers[i] != want {
			t.Fatalf("Providers[%d] = %q, want %q (ascending)", i, stats.Providers[i], want)
		}
	}
}

// TestStatsCapsAt20 pins the documented result caps on Stats.SessionTitles
// and Stats.ModelBreakdown: with more than 20 sessions and models, both
// slices truncate at exactly 20 entries — silently, as their field docs
// promise — keeping the top sessions by message count and the top models
// by cost.
func TestStatsCapsAt20(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		for i := range 25 {
			id := fmt.Sprintf("s-%02d", i)
			seedSessionWithEconomics(
				db, id, "", fmt.Sprintf("Session %d", i),
				i+1, 0, 0, float64(i+1), fixtureBase, fixtureBase,
			)
			insertMessage(t, db, "m-"+id, id, "assistant", "[]", fmt.Sprintf("model-%02d", i), "", fixtureBase)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay()})
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if len(stats.SessionTitles) != 20 {
		t.Fatalf("SessionTitles = %d entries, want exactly 20 (documented cap)", len(stats.SessionTitles))
	}

	if stats.SessionTitles[0] != "Session 24" {
		t.Fatalf("SessionTitles[0] = %q, want the most-active session first", stats.SessionTitles[0])
	}

	if len(stats.ModelBreakdown) != 20 {
		t.Fatalf("ModelBreakdown = %d entries, want exactly 20 (documented cap)", len(stats.ModelBreakdown))
	}

	if stats.ModelBreakdown[0].Model != "model-24" {
		t.Fatalf("ModelBreakdown[0].Model = %q, want the highest-cost model first", stats.ModelBreakdown[0].Model)
	}
}

// TestSummaryMessagesAreCounted pins the is_summary_message decision (the
// messages.go/stats.go doc comments restate it): summary rows — upstream
// writes them via message.Create with IsSummaryMessage when compacting a
// session's context — carry real content and model attribution, so they are
// counted everywhere regular messages are. Here the summary row is the ONLY
// attributed message of its session: any filtering would silently empty
// Models and ModelBreakdown. Excluding summaries would also break number
// parity with the crush-daily collector SQL (assertCrushDailyCollectorParity),
// whose queries never filtered the flag either.
func TestSummaryMessagesAreCounted(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(handle *sql.DB) {
		insertSession(t, handle, "s1", "", "Compacted session", 2, fixtureBase, fixtureBase+9)
		insertMessage(t, handle, "m1", "s1", "user", `[{"type":"text","data":{"text":"long prefix"}}]`,
			"", "", fixtureBase)

		_, err := handle.ExecContext(context.Background(), `
			INSERT INTO messages (id, session_id, role, parts, model, provider, created_at, updated_at, is_summary_message)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
		`, "m2", "s1", "assistant", `[{"type":"text","data":{"text":"summary of the prefix"}}]`,
			fixtureModel, "", fixtureBase+1, fixtureBase+1)
		if err != nil {
			t.Fatal(err)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	messages, err := db.Messages(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2 (the summary row is returned like any other)", len(messages))
	}

	if text, ok := messages[1].Parts[0].(TextPart); !ok || text.Text != "summary of the prefix" {
		t.Fatalf("messages[1].Parts[0] = %#v, want the summary TextPart", messages[1].Parts[0])
	}

	stats, err := db.Stats(context.Background(), StatsFilter{Day: fixtureDay()})
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if len(stats.Models) != 1 || stats.Models[0] != fixtureModel {
		t.Fatalf("Models = %v, want the summary row's %q counted", stats.Models, fixtureModel)
	}

	if len(stats.ModelBreakdown) != 1 || stats.ModelBreakdown[0].Model != fixtureModel {
		t.Fatalf("ModelBreakdown = %+v, want the summary-attributed session", stats.ModelBreakdown)
	}

	if stats.ModelBreakdown[0].MessageCount != 2 {
		t.Fatalf(
			"MessageCount = %d, want 2 (all messages of the attributed session, matching the historical subquery)",
			stats.ModelBreakdown[0].MessageCount,
		)
	}
}
