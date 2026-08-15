// Package crushdata provides typed, read-only access to the local session
// data written by charmbracelet/crush (https://github.com/charmbracelet/crush).
//
// Crush stores its data in two places: a project registry at
// ~/.local/share/crush/projects.json and one SQLite database (crush.db) per
// project data directory. Neither location has a documented or stable schema;
// this library centralizes the reverse-engineered knowledge so every consumer
// does not have to repeat it.
//
// # Schema drift
//
// The crush.db schema changes across Crush versions. Columns such as
// sessions.cost, messages.model, messages.provider, and
// sessions.parent_session_id were added in later migrations and may be absent
// from older databases. [DB.Open] probes the schema once and every read
// adapts to what it finds: absent columns are reported as zero values instead
// of failing, and [DB.Schema] exposes the detected capabilities so callers can
// warn their users about reduced coverage. Databases that lack the required
// sessions or messages tables fail [DB.Open] with [ErrUnsupportedSchema].
//
// # Read-only access
//
// Databases are opened with SQLite's mode=ro flag and a single connection, so
// reads are safe to run alongside a live Crush process. The library never
// writes.
//
// # Timestamps
//
// Crush stores timestamps as Unix seconds in the UTC time zone. This library
// converts them to time.Time in UTC. Day filters ([SessionFilter.Day] and
// [StatsFilter.Day]) compare against the UTC calendar day of created_at.
package crushdata
