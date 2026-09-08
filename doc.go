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
// sessions or messages tables fail [DB.Open] with [ErrUnsupportedSchema];
// databases whose schema cannot be probed at all (a canceled context, a file
// that is not a SQLite database) fail with the underlying probe error —
// "unreadable" and "too old" are different diagnoses. [DB.OpenContext] runs
// the same probes under a caller-supplied context.
//
// # Read-only access
//
// Databases are opened with SQLite's mode=ro flag and a single connection, so
// reads are safe to run alongside a live Crush process. The library never
// writes.
//
// # Timestamps
//
// Crush stores timestamps as Unix seconds. This library converts them to
// time.Time in UTC. Day filters ([SessionFilter.Day] and [StatsFilter.Day])
// match the UTC calendar day of created_at against the filter value formatted
// in its own location: pass a time in the zone whose day boundary you want
// (usually local midnight). See [SessionFilter.Day] for the exact semantics.
//
// # The parts envelope
//
// A message's content is stored as a JSON array in the messages.parts
// column. Each entry is an envelope of the shape {"type": ..., "data": {...}}
// where type selects the part kind and data carries its payload. Crush
// v0.92.0 writes eight discriminators:
//
//	text          → [TextPart]
//	reasoning     → [ReasoningPart]
//	tool_call     → [ToolCallPart]
//	tool_result   → [ToolResultPart]
//	finish        → [FinishPart]
//	shell_command → [ShellCommandPart]
//	image_url     → [UnknownPart] (attachment pass-through; not decoded)
//	binary        → [UnknownPart] (attachment pass-through; not decoded)
//
// Decoding is deliberately tolerant: an entry with an unknown discriminator
// or a malformed payload degrades to [UnknownPart] carrying the discriminator
// and raw payload instead of failing the whole message (strict all-or-nothing
// decoding is available via [DecodeParts]). Typeless entries (empty type with
// null data) are skipped — Crush has always written them. The known set is
// pinned against real databases by the census tripwires in realdata_test.go,
// which fail loudly when an upstream release adds a ninth discriminator.
package crushdata
