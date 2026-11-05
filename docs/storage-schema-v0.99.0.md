# Storage schema snapshot — charmbracelet/crush v0.99.0

Verified against upstream **v0.99.0** (commit `e38655a`) on 2026-11-04 by
reading the goose migrations under `internal/db/migrations/` — the same
pinned set `scripts/check-upstream-drift.sh` clones. This file is the
human-readable snapshot of what this library reads; the machine-enforced
truth lives in `schema_drift_test.go` (`TestUpstreamMigrationColumnsAreProbedOrExempt`).
The schema is also pinned mechanically since 2026-09-08:
`scripts/genschema` applies the pinned migrations to a throwaway SQLite
database and dumps the resulting CREATE statements into the checked-in
[storage-schema-v0.99.0.sql](storage-schema-v0.99.0.sql);
`scripts/check-upstream-drift.sh` diffs that snapshot on every run, and
`TestFixtureSchemaMatchesUpstreamSnapshot` (schema_snapshot_test.go)
requires the test-fixture DDL to cover every table and column of it.
Regenerate this document and the snapshot on every verified release (see
AGENTS.md cadence).

Drift since the previous snapshot (v0.92.0): two new MCP config tables,
four `prism_*` columns and a `channel` column (all read here via probed
capabilities), one messages index, and the initial migration's timestamp comments
corrected from "milliseconds" to "seconds" — the upstream fix for our
[crush#3576](https://github.com/charmbracelet/crush/pull/3576) (merged
2026-09-08) first shipped in v0.93.1. The values were always seconds; only
the lying comments changed.

## sessions

| column             | type    | added in                     | this library                                                        |
| ------------------ | ------- | ---------------------------- | ------------------------------------------------------------------- |
| id                 | TEXT PK | initial (2025-04-24)         | reads                                                               |
| parent_session_id  | TEXT    | probed capability            | reads (agent graphs; absent → flat)                                 |
| title              | TEXT    | initial                      | reads                                                               |
| message_count      | INTEGER | initial (trigger-maintained) | reads                                                               |
| prompt_tokens      | INTEGER | initial                      | reads (stats)                                                       |
| completion_tokens  | INTEGER | initial                      | reads (stats)                                                       |
| cost               | REAL    | probed capability            | reads (absent → 0)                                                  |
| updated_at         | INTEGER | initial                      | reads                                                               |
| created_at         | INTEGER | initial                      | reads                                                               |
| summary_message_id | TEXT    | 2025-05-15 migration         | probed capability — reads (`Session.SummaryMessageID`; absent → "") |
| todos              | TEXT    | 2025-08-12 migration         | reads as raw JSON (`Session.Todos`)                                 |
| channel            | TEXT    | 2026-09-21 migration         | probed capability — reads (`Session.Channel`; absent/NULL → "")     |

`parent_session_id` and `todos` ride the initial `CREATE TABLE` in the
pinned source but were added by later migrations historically; the probe
covers databases frozen before those migrations landed.

## messages

| column                    | type    | added in                      | this library                                                                   |
| ------------------------- | ------- | ----------------------------- | ------------------------------------------------------------------------------ |
| id                        | TEXT PK | initial (UUIDv4 — random!)    | reads                                                                          |
| session_id                | TEXT    | initial (FK, indexed)         | reads                                                                          |
| role                      | TEXT    | initial                       | reads                                                                          |
| parts                     | TEXT    | initial                       | reads (the `{type,data}` envelope)                                             |
| model                     | TEXT    | probed capability             | reads (absent → "")                                                            |
| created_at                | INTEGER | initial (indexed since 06-24) | reads                                                                          |
| updated_at                | INTEGER | initial (trigger-maintained)  | reads (`Message.UpdatedAt`)                                                    |
| finished_at               | INTEGER | probed capability             | reads (absent → zero time)                                                     |
| provider                  | TEXT    | 2025-06-27 migration          | reads (absent → "")                                                            |
| is_summary_message        | INTEGER | 2025-08-10 migration          | probed capability — reads (`Message.IsSummaryMessage`; absent → false)         |
| prism_model_id            | TEXT    | 2026-09-02 migration          | probed capability — reads (`Message.PrismModelID`; absent/NULL → "")           |
| prism_model_name          | TEXT    | 2026-09-02 migration          | probed capability — reads (`Message.PrismModelName`; absent/NULL → "")         |
| prism_hypercredit_savings | REAL    | 2026-09-02 migration          | probed capability — reads (`Message.PrismHypercreditSavings`; absent/NULL → 0) |
| prism_dollar_savings      | REAL    | 2026-09-02 migration          | probed capability — reads (`Message.PrismDollarSavings`; absent/NULL → 0)      |

Message IDs are `uuid.New()` (UUIDv4): random, so insertion order is
`rowid`, never `(created_at, id)` — this library orders by `rowid`.
v0.95.0 adds `idx_messages_role_created_at` (an index-only migration; no
column/table additions, invisible to the drift extractor by design).

## read_files (table added 2026-01-27; probed)

| column     | type    | notes                                  |
| ---------- | ------- | -------------------------------------- |
| session_id | TEXT    | PK(path, session_id), FK to sessions   |
| path       | TEXT    |                                        |
| read_at    | INTEGER | seconds; this library sorts DESC on it |

## files (initial; intentionally not read)

File snapshots: `id, session_id, path, content, version, created_at,
updated_at`, `UNIQUE(path, session_id, version)`. Written by upstream
internal/history; this library never queries it.

## mcp_disabled_servers / mcp_enabled_servers (added 2026-09-28; unread)

Single-column config tables — `name TEXT PRIMARY KEY` each — tracking
which MCP servers the user disabled/enabled. Written by upstream config
persistence; this library never queries them. Exempt in the guard list.

## Timestamps are Unix SECONDS

The initial migration's comments claimed "Unix timestamp in milliseconds";
they were wrong, and our
[crush#3576](https://github.com/charmbracelet/crush/pull/3576) comment fix
(merged 2026-09-08) shipped in v0.93.1 — the comments now correctly say
seconds. The values were always seconds: the `update_*_updated_at` triggers
write `strftime('%s', 'now')`, the read_files comment always said seconds,
and the CLI renders `time.Unix(x, 0)`. Two of six ecosystem Go readers
shipped `time.UnixMilli` date bugs trusting the old comment (both fixed
2026-09-08: openusage#357, mnemo#22). Zero/negative values read as the zero
`time.Time`.

## The parts envelope

`messages.parts` is a JSON array of `{"type": ..., "data": {...}}` entries.
Upstream v0.99.0 writes eight discriminators: `reasoning`, `text`,
`image_url`, `binary`, `tool_call`, `tool_result`, `finish`,
`shell_command` (re-verified in source at `e38655a`, internal/message).
This library decodes six to concrete types and passes `image_url`/`binary`
through as `UnknownPart`. A registry-wide census (2026-09-08, 243
databases, 5,223,870 entries) found zero discriminators outside this set;
`TestPartDiscriminatorsCensus*` is the standing tripwire.

## Registry: projects.json

`<global>/projects.json` holds `{"projects": [{"path", "data_dir",
"last_accessed"}]}` with RFC3339Nano timestamps. Global dir resolution:
`CRUSH_GLOBAL_DATA` → `$XDG_DATA_HOME/crush` (Unix) /
`%LOCALAPPDATA%\crush` (Windows) → `~/.local/share/crush`. `last_accessed`
is written by upstream `projects.Register()` as `time.Now().UTC()`, sorted
descending — the basis for this library's most-recently-accessed-wins
dedupe. Agent child session IDs look like `messageID$$toolCallID`; the
`crush projects --json` CLI prints its payload to stderr.
