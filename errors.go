package crushdata

import "errors"

// Sentinel errors returned by this package. All errors wrap one of these
// where applicable; use [errors.Is] to test for them.
var (
	// ErrUnsupportedSchema is returned by [DB.Open] when a database lacks the
	// required sessions or messages tables — the file is not a Crush session
	// database (or predates anything this library can read).
	ErrUnsupportedSchema = errors.New("crush database schema not supported")

	// ErrDatabaseNotFound is returned by [DB.Open] when the data directory
	// holds no crush.db file.
	ErrDatabaseNotFound = errors.New("crush database not found")

	// ErrRegistryNotFound is returned by [DiscoverProjects] when the
	// projects.json registry is absent and no CLI fallback is configured.
	// Install Crush, run it once, or set DiscoverOptions.CLIFallback.
	ErrRegistryNotFound = errors.New("crush projects registry not found")

	// ErrSessionNotFound is returned by [DB.Session] and [DB.AgentGraph] when
	// the requested session ID does not exist in the database.
	ErrSessionNotFound = errors.New("session not found")
)
