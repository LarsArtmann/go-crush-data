package crushdata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// writeRegistry writes a projects.json registry into globalDir.
func writeRegistry(t *testing.T, globalDir string, content string) {
	t.Helper()

	if err := os.MkdirAll(globalDir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(globalDir, RegistryName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// makeProjectDB creates a non-empty crush.db inside dataDir.
func makeProjectDB(t *testing.T, dataDir string) {
	t.Helper()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, nil)
}

func TestDiscoverProjectsRegistryFirst(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	dataDir := t.TempDir()
	makeProjectDB(t, dataDir)

	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/repo/proj","data_dir":`+jsonString(dataDir)+`,"last_accessed":"2026-08-15T10:00:00Z"}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(projects))
	}

	if projects[0].Path != "/repo/proj" || projects[0].DataDir != dataDir {
		t.Fatalf("project = %+v", projects[0])
	}

	want := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	if !projects[0].LastAccessed.Equal(want) {
		t.Fatalf("LastAccessed = %v, want %v", projects[0].LastAccessed, want)
	}
}

// jsonString quotes s as a JSON string with escaped backslashes (Windows-ish
// paths appear on all platforms in tests).
func jsonString(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}

// TestDiscoverProjectsDedupeSharedDataDir covers the registry quirk: many
// paths routing to one shared data_dir collapse to a single project, keeping
// the most recently accessed path.
func TestDiscoverProjectsDedupeSharedDataDir(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	sharedDir := t.TempDir()
	otherDir := t.TempDir()
	makeProjectDB(t, sharedDir)
	makeProjectDB(t, otherDir)

	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/home/lars/projects/sub-a","data_dir":`+jsonString(sharedDir)+`,"last_accessed":"2026-08-14T10:00:00Z"},
		{"path":"/home/lars/projects/sub-b","data_dir":`+jsonString(sharedDir)+`,"last_accessed":"2026-08-15T09:00:00Z"},
		{"path":"/home/lars/projects/sub-c","data_dir":`+jsonString(sharedDir)+`,"last_accessed":"2026-08-13T09:00:00Z"},
		{"path":"/repo/other","data_dir":`+jsonString(otherDir)+`,"last_accessed":"2026-08-15T11:00:00Z"}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("projects = %d, want 2 (shared dir deduped): %+v", len(projects), projects)
	}

	for _, project := range projects {
		if project.DataDir != sharedDir {
			continue
		}

		if project.Path != "/home/lars/projects/sub-b" {
			t.Fatalf("shared dir kept path %q, want newest-accessed sub-b", project.Path)
		}
	}
}

// TestDiscoverProjectsDedupeEqualLastAccessedKeepsFirst pins the tie-break:
// when two registry paths share a data_dir AND a last_accessed timestamp,
// the first registry entry wins (no reorder based on path).
func TestDiscoverProjectsDedupeEqualLastAccessedKeepsFirst(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	sharedDir := t.TempDir()
	makeProjectDB(t, sharedDir)

	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/home/lars/projects/first","data_dir":`+jsonString(sharedDir)+`,"last_accessed":"2026-08-15T10:00:00Z"},
		{"path":"/home/lars/projects/second","data_dir":`+jsonString(sharedDir)+`,"last_accessed":"2026-08-15T10:00:00Z"}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(projects))
	}

	if projects[0].Path != "/home/lars/projects/first" {
		t.Fatalf("Path = %q, want the first registry entry on ties", projects[0].Path)
	}
}

func TestDiscoverProjectsSkipsMissingDatabases(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()

	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/repo/fresh","data_dir":"/nonexistent/data-dir","last_accessed":"2026-08-15T10:00:00Z"}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 0 {
		t.Fatalf("projects = %+v, want none (no crush.db yet)", projects)
	}
}

func TestDiscoverProjectsEmptyAndNullDataDir(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()

	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/repo/no-dir","data_dir":"","last_accessed":"2026-08-15T10:00:00Z"}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 0 {
		t.Fatalf("projects = %+v, want none", projects)
	}
}

func TestDiscoverProjectsRegistryMissing(t *testing.T) {
	t.Parallel()

	_, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: t.TempDir()})
	if !errors.Is(err, ErrRegistryNotFound) {
		t.Fatalf("err = %v, want ErrRegistryNotFound", err)
	}
}

func TestDiscoverProjectsRegistryMalformed(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	writeRegistry(t, globalDir, `{not json`)

	_, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err == nil || errors.Is(err, ErrRegistryNotFound) {
		t.Fatalf("err = %v, want a parse error distinct from registry-missing", err)
	}
}

// fakeCLIEnv marks the process as a fake-CLI child (set by fakeCLI in the
// parent); the payload and exit code travel in sibling variables.
const (
	fakeCLIEnv        = "GO_CRUSH_DATA_FAKE_CLI"
	fakeCLIPayloadEnv = fakeCLIEnv + "_PAYLOAD"
	fakeCLIExitEnv    = fakeCLIEnv + "_EXIT"
)

// TestMain doubles as the fake CLI. DiscoverProjects execs the path fakeCLI
// returns — this very test binary — so the child re-enters TestMain, sees
// the marker env var, prints the requested payload to stderr, and exits
// with the requested code, never reaching m.Run.
//
// Decision (2026-08-16): re-exec the test binary instead of `go build`-ing
// a testdata helper — no toolchain invocation inside tests, no second
// binary to keep cross-platform, and argv needs no changes because the
// payload rides in env vars.
func TestMain(m *testing.M) {
	if os.Getenv(fakeCLIEnv) != "" {
		_, _ = os.Stderr.WriteString(os.Getenv(fakeCLIPayloadEnv))

		code := 0

		if s := os.Getenv(fakeCLIExitEnv); s != "" {
			if parsed, err := strconv.Atoi(s); err == nil {
				code = parsed
			}
		}

		os.Exit(code)
	}

	os.Exit(m.Run())
}

// fakeCLI configures the test binary itself (via TestMain) to stand in for
// the crush CLI: it prints payload to stderr — the stream the real CLI
// uses — and exits with the given code; exitCode != 0 pins the CLI-fallback
// error path (partial JSON printed, then failure). Re-execing the test
// binary instead of writing a /bin/sh script keeps the fixture
// cross-platform: Windows has no /bin/sh, which is why these tests used to
// skip there and the v0.2.0 Windows breakage went unnoticed until a tagged
// release. Payload and exit code travel in process env vars (t.Setenv), so
// tests using fakeCLI must not call t.Parallel.
func fakeCLI(t *testing.T, payload string, exitCode int) string {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv(fakeCLIEnv, "1")
	t.Setenv(fakeCLIPayloadEnv, payload)
	t.Setenv(fakeCLIExitEnv, strconv.Itoa(exitCode))

	return exe
}

func TestDiscoverProjectsCLIFallback(t *testing.T) { //nolint:paralleltest // t.Setenv
	globalDir := t.TempDir()
	dataDir := t.TempDir()
	makeProjectDB(t, dataDir)

	payload := `{"projects":[{"path":"/repo/cli","data_dir":` + jsonString(
		dataDir,
	) + `,"last_accessed":"2026-08-15T12:00:00Z"}]}`

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: globalDir,
		CLIFallback:   true,
		CLIBinary:     fakeCLI(t, payload, 0),
	})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 1 || projects[0].Path != "/repo/cli" {
		t.Fatalf("projects = %+v, want the CLI-discovered project", projects)
	}
}

// TestDiscoverProjectsCLIFallbackToleratesLogNoise pins the real-world CLI
// contract: the crush CLI may print log or warning lines around its JSON
// payload on stderr; discovery must still decode the payload.
func TestDiscoverProjectsCLIFallbackToleratesLogNoise(t *testing.T) { //nolint:paralleltest // t.Setenv
	globalDir := t.TempDir()
	dataDir := t.TempDir()
	makeProjectDB(t, dataDir)

	payload := "INFO crush v0.9.0 starting\n" +
		`{"projects":[{"path":"/repo/cli","data_dir":` + jsonString(dataDir) + `,"last_accessed":"2026-08-15T12:00:00Z"}]}` +
		"\nWARN deprecated flag --json\n"

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: globalDir,
		CLIFallback:   true,
		CLIBinary:     fakeCLI(t, payload, 0),
	})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 1 || projects[0].Path != "/repo/cli" {
		t.Fatalf("projects = %+v, want the CLI-discovered project", projects)
	}
}

// TestDiscoverProjectsUnreadableRegistryFallsBackToCLI pins the fallback
// trigger: a registry that exists but cannot be read (permissions) falls
// through to the CLI exactly like a missing one. Skipped when running as
// root, which reads chmod-000 files regardless.
func TestDiscoverProjectsUnreadableRegistryFallsBackToCLI(t *testing.T) { //nolint:paralleltest // t.Setenv
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod-000 files are still readable")
	}

	globalDir := t.TempDir()
	dataDir := t.TempDir()
	makeProjectDB(t, dataDir)

	registryPath := filepath.Join(globalDir, RegistryName)
	if err := os.WriteFile(registryPath, []byte(`{"projects":[]}`), 0o000); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chmod(registryPath, 0o600) })

	dataDirJSON := jsonString(dataDir)
	payload := `{"projects":[{"path":"/repo/cli","data_dir":` + dataDirJSON +
		`,"last_accessed":"2026-08-15T12:00:00Z"}]}`

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: globalDir,
		CLIFallback:   true,
		CLIBinary:     fakeCLI(t, payload, 0),
	})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 1 || projects[0].Path != "/repo/cli" {
		t.Fatalf("projects = %+v, want the CLI-discovered project", projects)
	}
}

func TestDiscoverProjectsCLIFailure(t *testing.T) {
	t.Parallel()

	_, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: t.TempDir(),
		CLIFallback:   true,
		CLIBinary:     "/nonexistent/crush-binary",
	})
	if err == nil {
		t.Fatal("err = nil, want exec failure")
	}
}

// TestDiscoverProjectsCLIExitNonzeroWithPartialJSON pins the error path: a
// CLI that exits nonzero after printing partial JSON to stderr must surface
// as an error, not silently return the partial payload's projects.
func TestDiscoverProjectsCLIExitNonzeroWithPartialJSON(t *testing.T) { //nolint:paralleltest // t.Setenv
	globalDir := t.TempDir()

	payload := `{"projects":[{"path":"/repo/partial","data_dir":"/nonexistent","last_accessed":"x"}`

	_, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: globalDir,
		CLIFallback:   true,
		CLIBinary:     fakeCLI(t, payload, 1),
	})
	if err == nil {
		t.Fatal("err = nil, want exit-nonzero error (partial JSON must not silently succeed)")
	}
}

func TestParseProjectsOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "nil input", raw: "", want: 0},
		{name: "json null", raw: `null`, want: 0},
		{name: "empty projects", raw: `{"projects":[]}`, want: 0},
		{
			name: "valid",
			raw:  `{"projects":[{"path":"/foo","data_dir":"/bar","last_accessed":"2026-01-01T00:00:00Z"}]}`,
			want: 1,
		},
		{name: "malformed", raw: `{"projects":`, wantErr: true},
		{name: "wrong shape", raw: `[]`, wantErr: true},
		{
			name: "log noise around payload",
			raw:  "INFO starting\n" + `{"projects":[]}` + "\nWARN done\n",
			want: 0,
		},
		{name: "noise without any object", raw: "no json here at all\n", wantErr: true},
		{name: "noise with stray braces only", raw: "loaded {config} from disk\n", wantErr: true},
		{
			name: "brace inside JSON string value",
			raw:  `{"projects":[{"path":"/repo/{name}","data_dir":"/d","last_accessed":"2026-01-01T00:00:00Z"}]}`,
			want: 1,
		},
		{
			name: "closing brace inside JSON string with noise",
			raw:  "INFO starting\n" + `{"projects":[{"path":"test}","data_dir":"/d","last_accessed":"x"}]}` + "\n",
			want: 1,
		},
		{
			name: "brace in trailing noise after payload",
			raw:  "INFO starting\n" + `{"projects":[]}` + "\nWARN cache {flushed}\n",
			// extractJSONObject spans to the LAST '}', so brace-bearing
			// trailing noise extends the substring past the payload and
			// json.Unmarshal rejects it — the documented limitation.
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			projects, err := ParseProjectsOutput([]byte(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("err = nil, want error")
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseProjectsOutput: %v", err)
			}

			if len(projects) != tt.want {
				t.Fatalf("projects = %d, want %d", len(projects), tt.want)
			}
		})
	}
}

func TestParseTimeTolerance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want bool // want zero time
	}{
		{name: "empty", raw: "", want: true},
		{name: "garbage", raw: "not-a-time", want: true},
		{name: "valid nano", raw: "2026-08-15T18:01:14.491439042Z", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseTime(tt.raw).IsZero(); got != tt.want {
				t.Fatalf("parseTime(%q).IsZero() = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

// TestWindowsLocalAppData pins the Windows fallback resolution: LOCALAPPDATA
// wins when set; otherwise it is derived from USERPROFILE. The function only
// reads environment variables, so the pin runs on every platform.
func TestWindowsLocalAppData(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\me\AppData\Local`)

	if got := windowsLocalAppData(); got != `C:\Users\me\AppData\Local` {
		t.Fatalf("windowsLocalAppData = %q, want LOCALAPPDATA verbatim", got)
	}

	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("USERPROFILE", `C:\Users\me`)

	want := `C:\Users\me` + string(filepath.Separator) + "AppData" + string(filepath.Separator) + "Local"
	if got := windowsLocalAppData(); got != want {
		t.Fatalf("windowsLocalAppData = %q, want %q derived from USERPROFILE", got, want)
	}

	t.Setenv("USERPROFILE", "")

	if got := windowsLocalAppData(); got != "" {
		t.Fatalf("windowsLocalAppData = %q, want empty when nothing resolves", got)
	}
}

func TestGlobalDataDir(t *testing.T) {
	t.Setenv("CRUSH_GLOBAL_DATA", "/custom/global")

	if got := GlobalDataDir(); got != "/custom/global" {
		t.Fatalf("GlobalDataDir = %q, want /custom/global", got)
	}
}

// TestDiscoverProjectsDedupeZeroTimestampLoses pins the documented claim:
// when two registry entries share a DataDir and one has a zero timestamp
// (missing or unparseable last_accessed), the non-zero entry wins.
func TestDiscoverProjectsDedupeZeroTimestampLoses(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	sharedDir := t.TempDir()
	makeProjectDB(t, sharedDir)

	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/repo/zero-ts","data_dir":`+jsonString(sharedDir)+`,"last_accessed":""},
		{"path":"/repo/real-ts","data_dir":`+jsonString(sharedDir)+`,"last_accessed":"2026-08-15T10:00:00Z"}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("projects = %d, want 1 (deduped)", len(projects))
	}

	if projects[0].Path != "/repo/real-ts" {
		t.Fatalf("Path = %q, want /repo/real-ts (non-zero timestamp wins)", projects[0].Path)
	}
}

// TestDiscoverProjectsDedupeZeroTimestampOnlyEntryStillAppears pins that a
// zero-timestamp entry is not silently dropped when it is the only entry for
// its DataDir.
func TestDiscoverProjectsDedupeZeroTimestampOnlyEntryStillAppears(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	dataDir := t.TempDir()
	makeProjectDB(t, dataDir)

	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/repo/zero-only","data_dir":`+jsonString(dataDir)+`,"last_accessed":""}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(projects))
	}

	if projects[0].Path != "/repo/zero-only" {
		t.Fatalf("Path = %q, want /repo/zero-only", projects[0].Path)
	}
}

// TestDiscoverProjectsOrderedByDataDir pins the documented ordering: results
// are sorted by DataDir ascending across multiple projects.
func TestDiscoverProjectsOrderedByDataDir(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	dirA := t.TempDir()
	dirB := t.TempDir()
	dirC := t.TempDir()
	makeProjectDB(t, dirA)
	makeProjectDB(t, dirB)
	makeProjectDB(t, dirC)

	// Register in non-alphabetical order.
	writeRegistry(t, globalDir, `{"projects":[
		{"path":"/repo/charlie","data_dir":`+jsonString(dirC)+`,"last_accessed":"2026-08-15T10:00:00Z"},
		{"path":"/repo/alpha","data_dir":`+jsonString(dirA)+`,"last_accessed":"2026-08-15T10:00:00Z"},
		{"path":"/repo/bravo","data_dir":`+jsonString(dirB)+`,"last_accessed":"2026-08-15T10:00:00Z"}
	]}`)

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{GlobalDataDir: globalDir})
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(projects) != 3 {
		t.Fatalf("projects = %d, want 3", len(projects))
	}

	if projects[0].DataDir != dirA || projects[1].DataDir != dirB || projects[2].DataDir != dirC {
		got := []string{projects[0].DataDir, projects[1].DataDir, projects[2].DataDir}
		t.Fatalf("DataDirs not sorted ascending: %v", got)
	}
}
