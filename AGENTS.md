# AGENTS.md — go-crush-data

Typed, read-only Go library for Crush local session data (`projects.json` +
per-data-dir `crush.db`). Public MIT repo. Extraction of logic previously
duplicated in crush-daily (collector) and the mindwalk fork (adapter/crush).

## Commands

```bash
# Canonical full gate — ALWAYS with pipefail; plain pipes mask lint exit codes
set -o pipefail
go build ./... && go vet ./... && go test -race -shuffle=on ./... \
  && nix run .#lint && nix flake check && nix develop --command actionlint \
  && scripts/check-doc-links.sh

go build ./...        # no GOEXPERIMENT needed (stdlib encoding/json v1 by design)
go test ./...         # fixtures are generated per-test; no committed binary testdata
go test -race ./...   # CI runs this (CI adds -shuffle=on); add -count=2 to catch order-dependent flakes
nix flake check       # build + format
nix run .#lint        # golangci-lint (~90 linters; run per-file while writing tests)
nix run .#test        # race test via nix
scripts/check-vendor-hash.sh  # local copy of the CI go.sum↔vendorHash drift guard
scripts/check-doc-links.sh    # markdown links + file:line citations in root docs resolve (runs in CI)
```

Optional: `CRUSH_DATA_REAL_DATA_DIR=<dir> go test -run TestSessionsOnRealDatabase` opens a real crush.db read-only. Re-run it after ANY source change — not just scan/probe code (a stats.go ORDER BY once changed real-read behavior).

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

Non-Go surfaces: `scripts/check-vendor-hash.sh` (drift guard),
`.github/workflows/` (ci, release, fuzz, bench, flake-update — all
tagged to pinned action SHAs), `docs/benchmarks/baseline-benchmark-sessions.txt`
(benchstat baseline for the bench.yml trend; regenerate via
`go test -bench . -count=6 | tee …`), `example_test.go` (runnable examples).

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
- **`OpenContext(ctx, dir)` is the real API**; `Open` delegates with
  `context.Background()`. Cancelation surfaces as `context.Canceled`.
- **`Session.Todos` is `json.RawMessage`** — byte-identical pass-through of
  upstream JSON; consumers decode it themselves. NULL → nil. BREAKING vs
  the old plain string; do not re-stringify.
- **Schema probes surface errors strictly**: a corrupt/unreadable DB returns
  the probe error, NOT `ErrUnsupportedSchema` (that error means specifically
  "recognized schema, missing capability columns"). Pinned by tests.
- **Accepted art-dupl clones (do not "fix")**: (1) the `costExpr := "0…"`
  capability-gating blocks in sessions.go/stats.go — three sites need three
  different string pairs (aliased/plain/qualified), so a helper would just
  restate the conditional; (2) the `dayFilter, args := dayArgs(day)` +
  QueryContext shape shared by fillTitlesAndHistogram/fillHourHistogram —
  the shared logic already lives in `dayArgs`, and unifying the two distinct
  queries would take more parameters than the duplicated lines.

## Process rules (truth discipline)

- **Verify-then-annotate**: never write "green"/"done" into any doc before
  the verification command exited 0 in the same session. Annotations
  record verification; they do not substitute for it.
- **Diff daemon commits before trusting their messages**: parts of this
  history are auto-generated commits whose subjects misdescribe their
  diffs (one claims "No library API surface changed" over the breaking
  `Session.Todos` change). Run `git show <sha>` before repeating any claim
  taken from a subject line. CHANGELOG.md, not `git log`, is the record of
  what shipped.

## Tooling gotchas

- **go.sum and flake.nix vendorHash are coupled**: refreshing dependencies
  without updating `vendorHash` breaks `nix flake check` (has bitten once on
  a dependency refresh). After `go get` / `go mod tidy`, update the hash
  from the mismatch error's "got:" value.
- **`nix flake check` only sees git-tracked files**: untracked `.go` files
  are invisible to the flake's source filter, producing misleading
  "undefined: ..." build errors. Commit new files before judging flake health.
- **Examples/tests live in-package** (`package crushdata`): depguard forbids
  an external `_test` package importing the module. `example_test.go` is
  in-package too.
- **gosec G701 false-positives** on the sessions QueryContext taint analysis;
  covered by a `//nolint:gosec` with rationale (codebase precedent).
- **benchstat is NOT in nixpkgs**: use
  `go run golang.org/x/perf/cmd/benchstat@latest`.
- **Windows is a first-class CI leg — tests must not assume POSIX**:
  fake CLIs re-exec the test binary via `TestMain` + env vars
  (`fakeCLI` in `discover_test.go` — never a `/bin/sh` script, which would
  force a Windows skip and hide real breakage, the v0.2.0 lesson); Windows
  paths embedded in JSON need backslash escaping (`jsonString` — a raw
  `C:\Users\...` breaks the decoder; pinned by `TestJSONString`); premises
  that only hold on POSIX (chmod-000 unreadability) need an explicit
  `runtime.GOOS` guard with the reason, not a silent skip.
- **Workflow steps need `defaults.run.shell: bash`**: the windows-latest
  default is PowerShell, which splits `-flag=file` arguments differently
  and broke the coverage step once.
- **Lint per file while writing tests** (`golangci-lint run <file>_test.go`),
  not after a large batch; and never trust `cmd | tail` without `pipefail`.

## Storage facts (reverse-engineered, upstream has no docs)

- Registry: `<global>/projects.json` — `{path, data_dir, last_accessed}`;
  global dir = CRUSH_GLOBAL_DATA → XDG_DATA_HOME/crush → ~/.local/share/crush.
- DB: `<data_dir>/crush.db`, tables sessions/messages/read_files.
- Agent child session IDs look like `messageID$$toolCallID`.
- CLI `crush projects --json` prints JSON on **stderr**.
