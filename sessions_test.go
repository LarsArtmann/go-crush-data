package crushdata

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionsListsRootsAndChildren(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	sessions, err := db.Sessions(context.Background(), SessionFilter{})
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("sessions = %d, want 2 (root + child)", len(sessions))
	}

	byID := map[string]Session{}
	for _, session := range sessions {
		byID[session.ID] = session
	}

	root := byID["fixture-root"]
	if root.Title != "Fixture root session" {
		t.Fatalf("root.Title = %q", root.Title)
	}

	if root.ParentSessionID != "" {
		t.Fatalf("root.ParentSessionID = %q, want empty", root.ParentSessionID)
	}

	if root.MessageCount != 10 || root.PromptTokens != 5000 || root.CompletionTokens != 1500 {
		t.Fatalf("root counts = %+v", root)
	}

	if root.CostUSD != 0.0234 {
		t.Fatalf("root.CostUSD = %v, want 0.0234", root.CostUSD)
	}

	child := byID["m_assistant_1$$call_agent_1"]
	if child.ParentSessionID != "fixture-root" {
		t.Fatalf("child.ParentSessionID = %q, want fixture-root", child.ParentSessionID)
	}
}

func TestSessionsOrdersNewestFirst(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	sessions, err := db.Sessions(context.Background(), SessionFilter{})
	if err != nil {
		t.Fatal(err)
	}

	if sessions[0].ID != "fixture-root" {
		t.Fatalf("first session = %q, want fixture-root (newest updated_at)", sessions[0].ID)
	}
}

func TestSessionsRootOnly(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	sessions, err := db.Sessions(context.Background(), SessionFilter{RootOnly: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 1 || sessions[0].ID != "fixture-root" {
		t.Fatalf("sessions = %+v, want only fixture-root", sessions)
	}
}

func TestSessionsParentFilter(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	sessions, err := db.Sessions(context.Background(), SessionFilter{ParentID: "fixture-root"})
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 1 || sessions[0].ID != "m_assistant_1$$call_agent_1" {
		t.Fatalf("sessions = %+v, want only the child", sessions)
	}
}

func TestSessionsParentFilterOnLegacySchema(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	sessions, err := db.Sessions(context.Background(), SessionFilter{ParentID: "fixture-root"})
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 0 {
		t.Fatalf("sessions = %+v, want none (no parent links exist)", sessions)
	}
}

func TestSessionsRootOnlyOnLegacySchemaIsNoop(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	sessions, err := db.Sessions(context.Background(), SessionFilter{RootOnly: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 2 {
		t.Fatalf("sessions = %d, want 2 (every session is a root on legacy schemas)", len(sessions))
	}
}

func TestSessionsFilterCombinationRejected(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	if _, err := db.Sessions(context.Background(), SessionFilter{ParentID: "x", RootOnly: true}); err == nil {
		t.Fatal("err = nil, want ParentID+RootOnly rejected")
	}
}

func TestSessionsDayFilter(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	onDay, err := db.Sessions(context.Background(), SessionFilter{Day: fixtureDay()})
	if err != nil {
		t.Fatal(err)
	}

	if len(onDay) != 2 {
		t.Fatalf("sessions on day = %d, want 2", len(onDay))
	}

	otherDay := fixtureDay().AddDate(0, 0, 1)
	offDay, err := db.Sessions(context.Background(), SessionFilter{Day: otherDay})
	if err != nil {
		t.Fatal(err)
	}

	if len(offDay) != 0 {
		t.Fatalf("sessions on next day = %d, want 0", len(offDay))
	}
}

// TestSessionsDayFilterUsesDayLocation verifies the day string is formatted
// in Day's own location: a session created at 23:30 UTC is excluded from a
// +02:00 local day whose midnight instant is 22:00 UTC of the previous day.
func TestSessionsDayFilterUsesDayLocation(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	createdAt := time.Date(2026, 8, 14, 23, 30, 0, 0, time.UTC).Unix()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "late-night", "", "Late night", 1, createdAt, createdAt)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	localZone := time.FixedZone("plus-two", 2*60*60)
	localMidnight := time.Date(2026, 8, 15, 0, 0, 0, 0, localZone)

	// The local day formats as 2026-08-15; the row's UTC date is 2026-08-14.
	sessions, err := db.Sessions(context.Background(), SessionFilter{Day: localMidnight})
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 0 {
		t.Fatalf("sessions = %d, want 0 (row is on UTC 2026-08-14, filter asked for 2026-08-15)", len(sessions))
	}

	// The same instant viewed in UTC is the day the row belongs to.
	sessions, err = db.Sessions(context.Background(), SessionFilter{Day: localMidnight.In(time.UTC)})
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1 (UTC day 2026-08-14 contains the row)", len(sessions))
	}
}

func TestSessionsLimit(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	sessions, err := db.Sessions(context.Background(), SessionFilter{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(sessions))
	}
}

func TestSessionByID(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	session, err := db.Session(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("Session: %v", err)
	}

	if session.ID != "fixture-root" || session.Title != "Fixture root session" {
		t.Fatalf("session = %+v", session)
	}
}

func TestSessionByIDNotFound(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	_, err := db.Session(context.Background(), "no-such-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionsLegacySchemaZeroSubstitution(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	sessions, err := db.Sessions(context.Background(), SessionFilter{})
	if err != nil {
		t.Fatalf("Sessions on legacy schema: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(sessions))
	}

	for _, session := range sessions {
		if session.CostUSD != 0 {
			t.Fatalf("session %s CostUSD = %v, want 0 (column absent)", session.ID, session.CostUSD)
		}

		if session.ParentSessionID != "" {
			t.Fatalf("session %s ParentSessionID = %q, want empty (column absent)", session.ID, session.ParentSessionID)
		}
	}
}

func TestSessionTimestampsAreUTC(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	session, err := db.Session(context.Background(), "fixture-root")
	if err != nil {
		t.Fatal(err)
	}

	if got := session.CreatedAt; got.Unix() != fixtureBase || got.Location() != time.UTC {
		t.Fatalf("CreatedAt = %v, want unix %d in UTC", got, fixtureBase)
	}
}

// TestSessionsOnRealDatabase opens the developer's (or this repository's)
// own real crush database when present, proving the library against live
// data. Skipped when no local database exists.
func TestSessionsOnRealDatabase(t *testing.T) {
	t.Parallel()

	dataDir := os.Getenv("CRUSH_DATA_REAL_DATA_DIR")
	if dataDir == "" {
		candidate, err := filepath.Abs(filepath.Join("..", ".crush"))
		if err != nil {
			t.Skip("no real data dir")
		}

		if _, statErr := os.Stat(filepath.Join(candidate, DBName)); statErr != nil {
			t.Skip("no real data dir (set CRUSH_DATA_REAL_DATA_DIR to run against one)")
		}

		dataDir = candidate
	}

	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open real database: %v", err)
	}

	defer func() { _ = db.Close() }()

	sessions, err := db.Sessions(context.Background(), SessionFilter{Limit: 5})
	if err != nil {
		t.Fatalf("Sessions on real database: %v", err)
	}

	t.Logf("real database schema: %+v, sessions read: %d", db.Schema(), len(sessions))
}
