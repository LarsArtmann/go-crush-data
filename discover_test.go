package crushdata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

// fakeCLI writes a shell script that prints the given payload to stderr (the
// stream the real CLI uses) and returns its path.
func fakeCLI(t *testing.T, payload string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "crush-fake")

	script := "#!/bin/sh\ncat " + shellQuoteFile(t, payload, dir) + " >&2\n"
	//nolint:gosec // the fake CLI script must be executable
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

func shellQuoteFile(t *testing.T, payload, dir string) string {
	t.Helper()

	file := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(file, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}

	return file
}

func TestDiscoverProjectsCLIFallback(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	dataDir := t.TempDir()
	makeProjectDB(t, dataDir)

	payload := `{"projects":[{"path":"/repo/cli","data_dir":` + jsonString(
		dataDir,
	) + `,"last_accessed":"2026-08-15T12:00:00Z"}]}`

	projects, err := DiscoverProjects(context.Background(), DiscoverOptions{
		GlobalDataDir: globalDir,
		CLIFallback:   true,
		CLIBinary:     fakeCLI(t, payload),
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

func TestParseProjectsOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "nil input", raw: "", want: 0},
		{name: "empty projects", raw: `{"projects":[]}`, want: 0},
		{
			name: "valid",
			raw:  `{"projects":[{"path":"/foo","data_dir":"/bar","last_accessed":"2026-01-01T00:00:00Z"}]}`,
			want: 1,
		},
		{name: "malformed", raw: `{"projects":`, wantErr: true},
		{name: "wrong shape", raw: `[]`, wantErr: true},
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

func TestGlobalDataDir(t *testing.T) {
	t.Setenv("CRUSH_GLOBAL_DATA", "/custom/global")

	if got := GlobalDataDir(); got != "/custom/global" {
		t.Fatalf("GlobalDataDir = %q, want /custom/global", got)
	}
}
