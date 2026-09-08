package crushdata

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// The fixture schema (currentSchemaDDL in testutil_test.go) used to be a
// hand-maintained approximation of the real crush.db schema and drifted at
// least once (the files table was missing entirely). The snapshot
// docs/storage-schema-v0.92.0.sql is GENERATED from the pinned upstream
// migrations by scripts/genschema, and this guard requires the fixture to
// declare every table and column of that snapshot — the fixture-drift
// analogue of the migration-drift guard in schema_drift_test.go. Regenerate
// the snapshot after bumping the upstream pin:
//
//	go run ./scripts/genschema -migrations <clone>/internal/db/migrations \
//	  > docs/storage-schema-v0.92.0.sql

// snapshotPath is the generated schema snapshot (exactly one file; the
// version in the name moves with the upstream pin).
const snapshotPath = "docs/storage-schema-v0.92.0.sql"

// tableColumns parses CREATE TABLE statements out of a blob of DDL and maps
// table name → sorted column names. Fragments are split on top-level commas
// (depth-aware, so PRIMARY KEY (path, session_id) stays one fragment);
// SQLite's inline `-- …` comments are stripped; table-level constraints
// (FOREIGN KEY, PRIMARY KEY, UNIQUE, CHECK, CONSTRAINT) are not columns and
// are skipped. Only the column NAMES are compared — the fixture intentionally
// approximates types and omits constraints.
func tableColumns(t *testing.T, ddl string) map[string][]string {
	t.Helper()

	columns := make(map[string][]string)

	for _, statement := range strings.Split(ddl, ";") {
		if table, body, ok := createTableBody(statement); ok {
			columns[table] = columnNames(body)
		}
	}

	return columns
}

// createTableBody finds the CREATE TABLE in a statement fragment and returns
// the table name plus the text between the outermost parentheses.
func createTableBody(statement string) (table, body string, ok bool) {
	lines := strings.Split(statement, "\n")

	nameLine := ""
	for _, line := range lines {
		clean := stripLineComment(line)

		if strings.Contains(strings.ToUpper(clean), "CREATE TABLE") {
			nameLine = clean

			break
		}
	}

	if nameLine == "" {
		return "", "", false
	}

	table = lastWord(strings.TrimSpace(strings.Split(nameLine, "(")[0]))
	if table == "" || !strings.Contains(statement, "(") {
		return "", "", false
	}

	open := strings.Index(statement, "(")

	depth := 0

	for i := open; i < len(statement); i++ {
		switch statement[i] {
		case '(':
			depth++
		case ')':
			depth--

			if depth == 0 {
				return table, statement[open+1 : i], true
			}
		}
	}

	return "", "", false
}

// columnNames splits a CREATE TABLE body on top-level commas and returns the
// sorted names of the column definitions (constraint fragments excluded).
func columnNames(body string) []string {
	var (
		names     []string
		fragment  strings.Builder
		depth     int
		flushFunc = func() {
			definition := firstWord(stripLineComment(fragment.String()))
			fragment.Reset()

			if definition == "" {
				return
			}

			upper := strings.ToUpper(definition)

			switch {
			case strings.HasPrefix(upper, "FOREIGN KEY"),
				strings.HasPrefix(upper, "PRIMARY KEY"),
				strings.HasPrefix(upper, "UNIQUE"),
				strings.HasPrefix(upper, "CHECK"),
				strings.HasPrefix(upper, "CONSTRAINT"):
				return
			}

			names = append(names, definition)
		}
	)

	for _, char := range body {
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				flushFunc()

				continue
			}
		}

		fragment.WriteRune(char)
	}

	flushFunc()

	slices.Sort(names)

	return names
}

// stripLineComment removes a `-- comment` suffix from one DDL line.
func stripLineComment(line string) string {
	if index := strings.Index(line, "--"); index >= 0 {
		line = line[:index]
	}

	return strings.TrimSpace(line)
}

func firstWord(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	return fields[0]
}

func lastWord(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	return fields[len(fields)-1]
}

// TestFixtureSchemaMatchesUpstreamSnapshot requires the current-schema
// fixture to cover exactly the tables and columns of the generated upstream
// snapshot. The legacy fixture intentionally predates columns and is not
// compared. On failure: extend currentSchemaDDL (the fixture may approximate
// types and omit constraints, but every column must exist), or regenerate
// the snapshot after an upstream pin bump.
func TestFixtureSchemaMatchesUpstreamSnapshot(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read %s (regenerate via scripts/genschema): %v", snapshotPath, err)
	}

	snapshot := tableColumns(t, string(raw))
	fixture := tableColumns(t, currentSchemaDDL)

	if len(snapshot) == 0 {
		t.Fatalf("%s parsed to zero tables — snapshot corrupt or parser broken", snapshotPath)
	}

	if len(fixture) == 0 {
		t.Fatal("currentSchemaDDL parsed to zero tables — parser broken")
	}

	for table, wantColumns := range snapshot {
		gotColumns, ok := fixture[table]
		if !ok {
			t.Errorf("fixture is missing table %q (snapshot columns: %v) — extend currentSchemaDDL", table, wantColumns)

			continue
		}

		for _, column := range wantColumns {
			if !slices.Contains(gotColumns, column) {
				t.Errorf("fixture table %q is missing column %q — extend currentSchemaDDL", table, column)
			}
		}
	}

	for table := range fixture {
		if _, ok := snapshot[table]; !ok {
			t.Errorf("fixture declares table %q, which the upstream snapshot does not have", table)
		}
	}
}

// TestFixtureSnapshotGuardDetectsMissingColumn proves the guard's negative
// path: a snapshot with one extra column must fail the comparison, so a
// stale fixture can never pass vacuously.
func TestFixtureSnapshotGuardDetectsMissingColumn(t *testing.T) {
	t.Parallel()

	snapshot := tableColumns(t, "CREATE TABLE sessions (id TEXT PRIMARY KEY, rogue_column TEXT);")
	fixture := tableColumns(t, "CREATE TABLE sessions (id TEXT PRIMARY KEY);")

	if !slices.Contains(snapshot["sessions"], "rogue_column") {
		t.Fatal("fixture error: parser did not extract the rogue column")
	}

	if slices.Contains(fixture["sessions"], "rogue_column") {
		t.Fatal("fixture error: rogue column extracted from the wrong side")
	}
}
