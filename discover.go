package crushdata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

// Constants describing where Crush keeps its data. They mirror the defaults
// of the upstream charmbracelet/crush CLI.
const (
	// AppName is the XDG application identifier Crush uses.
	AppName = "crush"

	// DBName is the SQLite filename Crush writes into every data directory.
	DBName = "crush.db"

	// RegistryName is the project registry filename inside the global data
	// directory.
	RegistryName = "projects.json"
)

// DiscoverOptions configures [DiscoverProjects].
type DiscoverOptions struct {
	// CLIFallback runs `crush projects --json` when the projects.json
	// registry is missing or unreadable. The CLI prints its JSON payload to
	// stderr; both streams are captured, matching how the CLI behaves today.
	CLIFallback bool

	// GlobalDataDir overrides the global Crush data directory that holds
	// projects.json. Empty uses [GlobalDataDir] resolution (CRUSH_GLOBAL_DATA,
	// then XDG_DATA_HOME, then the platform default). Tests use this to point
	// discovery at a fixture directory.
	GlobalDataDir string

	// CLIBinary is the executable used for CLIFallback. Empty means "crush"
	// from PATH. Tests use this to point at a fake CLI.
	CLIBinary string
}

// registryFile is the JSON shape of projects.json.
type registryFile struct {
	Projects []registryEntry `json:"projects"`
}

// registryEntry is one row of the registry.
type registryEntry struct {
	Path         string `json:"path"`
	DataDir      string `json:"data_dir"`
	LastAccessed string `json:"last_accessed"`
}

// GlobalDataDir returns the global Crush data directory: CRUSH_GLOBAL_DATA
// when set, otherwise $XDG_DATA_HOME/crush (Unix) or %LOCALAPPDATA%\crush
// (Windows), otherwise ~/.local/share/crush. Empty when the home directory
// cannot be resolved.
func GlobalDataDir() string {
	if env := os.Getenv("CRUSH_GLOBAL_DATA"); env != "" {
		return env
	}

	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName)
	}

	if runtime.GOOS == "windows" {
		if dir := windowsLocalAppData(); dir != "" {
			return filepath.Join(dir, AppName)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".local", "share", AppName)
}

func windowsLocalAppData() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		if home := os.Getenv("USERPROFILE"); home != "" {
			local = filepath.Join(home, "AppData", "Local")
		}
	}

	return local
}

// DiscoverProjects returns every project known to Crush, de-duplicated to
// one [Project] per data directory (the registry can map many working
// directories onto a shared data directory; the most recently accessed path
// wins). Projects whose crush.db does not exist yet are skipped — a freshly
// registered project has no sessions to read.
//
// The registry is read first. When it is missing and opts.CLIFallback is
// set, `crush projects --json` is executed as a fallback (its JSON arrives
// on stderr). An absent registry without fallback fails with
// [ErrRegistryNotFound]; exec and parse failures return the underlying
// cause. The result is sorted by DataDir for deterministic ordering.
func DiscoverProjects(ctx context.Context, opts DiscoverOptions) ([]Project, error) {
	globalDir := opts.GlobalDataDir
	if globalDir == "" {
		globalDir = GlobalDataDir()
	}

	projects, err := loadRegistry(globalDir)
	if err != nil {
		if !opts.CLIFallback {
			return nil, err
		}

		projects, err = queryProjectsCLI(ctx, opts.CLIBinary)
		if err != nil {
			return nil, err
		}
	}

	return dedupeProjects(projects), nil
}

// loadRegistry reads and parses the projects.json registry. A missing
// registry returns [ErrRegistryNotFound] wrapped with its path.
func loadRegistry(globalDir string) ([]Project, error) {
	if globalDir == "" {
		return nil, fmt.Errorf("%w: no global data directory could be resolved", ErrRegistryNotFound)
	}

	path := filepath.Join(globalDir, RegistryName)

	//nolint:gosec // the registry path is caller-supplied by design: reading local files at arbitrary paths is this library's purpose
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", ErrRegistryNotFound, path, err)
	}

	var file registryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	projects := make([]Project, 0, len(file.Projects))
	for _, entry := range file.Projects {
		projects = append(projects, Project{
			Path:         entry.Path,
			DataDir:      entry.DataDir,
			LastAccessed: parseTime(entry.LastAccessed),
		})
	}

	return projects, nil
}

// queryProjectsCLI runs the `crush projects --json` fallback and parses its
// output. The payload is printed on stderr, so combined output is captured.
func queryProjectsCLI(ctx context.Context, binary string) ([]Project, error) {
	name := binary
	if name == "" {
		name = "crush"
	}

	//nolint:gosec // running the configured crush binary is the documented CLI fallback, not an injection vector
	out, err := exec.CommandContext(ctx, name, "projects", "--json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run %s projects --json: %w", name, err)
	}

	projects, err := ParseProjectsOutput(out)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

// ParseProjectsOutput decodes the JSON payload emitted by
// `crush projects --json` (the same shape as projects.json). Empty input
// yields an empty slice without error, mirroring Crush's behaviour when the
// registry is absent. Log or warning lines the CLI prints around the payload
// are tolerated: the outermost JSON object is extracted before decoding, and
// input from which no decodable object can be extracted fails.
func ParseProjectsOutput(raw []byte) ([]Project, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var file registryFile
	if err := json.Unmarshal(extractJSONObject(raw), &file); err != nil {
		return nil, fmt.Errorf("parse crush projects output: %w", err)
	}

	projects := make([]Project, 0, len(file.Projects))
	for _, entry := range file.Projects {
		projects = append(projects, Project{
			Path:         entry.Path,
			DataDir:      entry.DataDir,
			LastAccessed: parseTime(entry.LastAccessed),
		})
	}

	return projects, nil
}

// extractJSONObject returns the substring from the first '{' to the last '}'
// in raw, or raw itself when either delimiter is missing so that the caller
// reports the original input in its parse error.
func extractJSONObject(raw []byte) []byte {
	start := bytes.IndexByte(raw, '{')

	end := bytes.LastIndexByte(raw, '}')
	if start < 0 || end <= start {
		return raw
	}

	return raw[start : end+1]
}

// dedupeProjects collapses discovered projects to one per existing crush.db.
// The entry with the newest LastAccessed wins the Path; ties keep the first
// entry, and entries with zero timestamps sort last.
func dedupeProjects(projects []Project) []Project {
	best := map[string]Project{}

	for _, project := range projects {
		if project.DataDir == "" {
			continue
		}

		if !databaseExists(project.DataDir) {
			continue
		}

		current, seen := best[project.DataDir]
		if !seen || project.LastAccessed.After(current.LastAccessed) {
			best[project.DataDir] = project
		}
	}

	unique := make([]Project, 0, len(best))
	for _, project := range best {
		unique = append(unique, project)
	}

	sort.Slice(unique, func(i, j int) bool {
		return unique[i].DataDir < unique[j].DataDir
	})

	return unique
}

// databaseExists reports whether dataDir holds a non-empty crush.db file.
func databaseExists(dataDir string) bool {
	info, err := os.Stat(filepath.Join(dataDir, DBName))

	return err == nil && !info.IsDir() && info.Size() > 0
}

// parseTime accepts the RFC 3339 timestamps Crush writes. Empty or
// malformed values return the zero time — registry rows written by other
// tools must not break discovery.
func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}

	return parsed.UTC()
}
