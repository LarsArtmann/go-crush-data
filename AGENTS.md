# AGENTS.md — go-crush-data

Typed, read-only Go library for Crush local session data (`projects.json` +
per-data-dir `crush.db`). Public MIT repo. Extraction of logic previously
duplicated in crush-daily (collector) and the mindwalk fork (adapter/crush).

## Commands

```bash
go build ./...        # no GOEXPERIMENT needed (stdlib encoding/json v1 by design)
go test ./...         # fixtures are generated per-test; no committed binary testdata
go test -race ./...   # CI runs this
nix flake check       # build + format
nix run .#lint        # golangci-lint
nix run .#test        # race test via nix
```

Optional: `CRUSH_DATA_REAL_DATA_DIR=<dir> go test -run TestSessionsOnRealDatabase` opens a real crush.db read-only.

## Architecture (single root package `crushdata`)

| File | Role |
|---|---|
| discover.go | projects.json registry + `crush projects --json` CLI fallback (stderr capture) + dedupe (many paths → one data_dir) |
| db.go | read-only open (`mode=ro&_txlock=immediate`, 1 conn) + ErrDatabaseNotFound/ErrUnsupportedSchema |
| schema.go | capability probing via pragma_table_info — THE drift defense |
| sessions.go | SessionFilter{ByID, Day, ParentID, RootOnly, Limit} + capability-substituted SQL |
| parts.go | sealed Part interface: Text/Reasoning/ToolCall/ToolResult/Finish/ShellCommand/Unknown |
| rows.go | `collectRows[T]` generic: iterate rows, scan each into T, collect, verify `rows.Err()` — the one row-collection path every query uses |
| messages.go | Messages(sessionID) ordered `created_at, id`; malformed parts → nil Parts (tolerant); ReadFiles |
| agents.go | AgentGraph via parent_session_id recursion (preorder, depth cap 64, flat fallback pre-column) |
| stats.go | day aggregates; model-breakdown CTE has the double-count trap — see comment there |

## Critical decisions

- **stdlib `encoding/json` v1, NOT v2**: jsonv2 still requires
  GOEXPERIMENT=jsonv2 in Go 1.26; a public library must not force the flag on
  consumers (mindwalk upstream would break). Revisit when jsonv2 graduates.
- **Day filters format the time in its own location** (`f.Day.Format(DateOnly)`),
  matching crush-daily's historical semantics for number parity.
- **Stats SQL is ported verbatim** from crush-daily's collector; the parity
  test (`TestStatsParityWithCrushDailySQL`) re-runs the old SQL and requires
  identical numbers. Do not "improve" the SQL without updating that contract.
- **Timestamps are unix seconds** (crush migration comment lies about
  milliseconds). Zero/negative → zero time.Time.
- Sessions dedupe keeps the most-recently-accessed Path per DataDir.
- **Accepted art-dupl clones (do not "fix")**: (1) the `costExpr := "0…"`
  capability-gating blocks in sessions.go/stats.go — three sites need three
  different string pairs (aliased/plain/qualified), so a helper would just
  restate the conditional; (2) the `dayFilter, args := dayArgs(day)` +
  QueryContext shape shared by fillTitlesAndHistogram/fillHourHistogram —
  the shared logic already lives in `dayArgs`, and unifying the two distinct
  queries would take more parameters than the duplicated lines.

## Storage facts (reverse-engineered, upstream has no docs)

- Registry: `<global>/projects.json` — `{path, data_dir, last_accessed}`;
  global dir = CRUSH_GLOBAL_DATA → XDG_DATA_HOME/crush → ~/.local/share/crush.
- DB: `<data_dir>/crush.db`, tables sessions/messages/read_files.
- Agent child session IDs look like `messageID$$toolCallID`.
- CLI `crush projects --json` prints JSON on **stderr**.
