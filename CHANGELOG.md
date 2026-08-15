# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Nothing yet.

### Fixed

- Nothing yet.

## [0.1.0] - 2026-08-15

Initial release: typed, read-only access to Crush local session data,
extracted from the reading logic previously duplicated in crush-daily and the
mindwalk fork.

### Added

- **Discovery**: `DiscoverProjects` reads the `projects.json` registry first
  (XDG path resolution with `CRUSH_GLOBAL_DATA` override), falls back to
  `crush projects --json` (payload arrives on stderr), de-duplicates the
  many-paths-to-one-`data_dir` registry quirk keeping the most recently
  accessed path, and skips projects whose `crush.db` does not exist yet.
- **Read-only database access**: `Open` uses SQLite `mode=ro` with a single
  connection and Crush's own `_txlock=immediate` hint, so reads are safe
  alongside a running Crush process. Missing databases fail with
  `ErrDatabaseNotFound`.
- **Schema-drift defense**: every `Open` probes capabilities via
  `pragma_table_info` (`sessions.cost`, `sessions.parent_session_id`,
  `messages.model`, `messages.provider`, `messages.finished_at`,
  `read_files`). Absent columns read as zero values instead of failing;
  `Schema.MissingColumns()` powers user warnings. Databases without the
  required tables fail with `ErrUnsupportedSchema`.
- **Sessions**: typed `Session` reads with `ByID`, `Day` (calendar day in
  the filter time's own location, matching historical analytics semantics),
  `ParentID`, `RootOnly`, and `Limit` filters; newest first.
- **Messages**: `Messages(sessionID)` ordered by `created_at, id`, with the
  parts JSON decoded into a sealed `Part` type — `Text`, `Reasoning`,
  `ToolCall`, `ToolResult`, `Finish`, `ShellCommand`, plus `Unknown`
  passthrough so future Crush part kinds never break reads. Malformed parts
  yield nil `Parts` for that message instead of failing the session read.
- **Read files**: `ReadFiles(sessionID)` from the optional `read_files`
  table, empty on databases that predate it.
- **Agent graphs**: `AgentGraph(rootID)` walks `parent_session_id` links in
  preorder with a depth guard; degrades to the root node on pre-column
  schemas.
- **Stats**: `Stats(filter)` day aggregates — session/message counts, token
  and cost sums, distinct models/providers, top session titles, a 24-hour
  histogram, and the per-model breakdown whose CTE prevents the
  session-per-message double count. Numbers are parity-locked against the
  original crush-daily SQL by a test.
- **Testing**: generated dual-schema fixtures (current + legacy), fuzzing
  for the parts decoder, volume stress tests, race detector, 85% coverage
  gate in CI, govulncheck.

[Unreleased]: https://github.com/LarsArtmann/go-crush-data/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/LarsArtmann/go-crush-data/releases/tag/v0.1.0
