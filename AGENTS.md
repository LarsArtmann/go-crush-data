# AGENTS.md — go-crush-data

Typed, read-only Go library for Crush local session data (`projects.json` +
per-data-dir `crush.db`). Public MIT repo. Extraction of logic previously
duplicated in crush-daily (collector) and the mindwalk fork (adapter/crush).

## Commands

```bash
# Canonical full gate — ALWAYS with pipefail; plain pipes mask lint exit codes
set -o pipefail
go build ./... && go vet ./... && go test -race -shuffle=on ./... \
  && go tool golangci-lint run --timeout=5m ./... && nix flake check \
  && nix develop --command actionlint && scripts/check-doc-links.sh

go build ./...        # no GOEXPERIMENT needed (stdlib encoding/json v1 by design)
go test ./...         # fixtures are generated per-test; no committed binary testdata
go test -race ./...   # CI runs this (CI adds -shuffle=on); add -count=2 to catch order-dependent flakes
go tool golangci-lint run --timeout=5m ./...  # THE lint command — byte-identical to CI's (go.mod tool pin)
nix flake check       # build + format
nix run .#lint        # convenience wrapper, DIFFERENT binary than CI's — do not gate on it alone (see tooling gotchas)
nix run .#test        # race test via nix
CRUSH_UPSTREAM_DIR=/tmp/crush-upstream scripts/check-upstream-drift.sh  # pinned-crush migrations vs capability guard + schema snapshot
scripts/check-upstream-drift.sh --self-test  # proves the drift negative path fires (doctored fixture)
scripts/check-upstream-status.sh   # filing-campaign thread states + G1/G2 verdict (T35 daily pass)
scripts/check-root-package-main.sh # no `package main` outside scripts//cmd/ (CI runs it too)
scripts/verify-all.sh              # ONE command: full gate + drift guard + real-data sweeps (summary table)
go run ./scripts/genschema -migrations <clone>/internal/db/migrations  # regen docs/storage-schema-*.sql after a pin bump
scripts/check-vendor-hash.sh  # local copy of the CI go.sum↔vendorHash drift guard (run after every go get / go mod tidy)
scripts/check-doc-links.sh    # markdown links + file:line citations in root docs resolve (runs in CI)
go test -cover ./...         # coverage number for FEATURES.md (docs-health cadence)
```

## Upstream verification cadence

On every new charmbracelet/crush **stable** release (not nightly):

1. bump `crush_ref`/`crush_sha` at the top of `scripts/check-upstream-drift.sh`
   to the new tag/commit,
2. run the script (clones the pinned release and diffs its non-initial
   migrations against the guard list in `schema_drift_test.go`),
3. on drift: probe or document-exempt every new column/table, extend the
   guard list, re-run `go test -run TestUpstreamMigrationColumnsAreProbedOrExempt .`,
4. re-run the real-data sweeps (`TestAllAPIOnRealDatabase` on the largest
   local registry DB, plus the parts-discriminator census:
   `CRUSH_DATA_REAL_REGISTRY=<global dir> go test -run
   TestPartDiscriminatorsCensusRegistry -timeout 30m .`) and update the
   last-verified tag below,
5. refresh the schema snapshot and its doc:
   `go run ./scripts/genschema -migrations <clone>/internal/db/migrations
   > docs/storage-schema-<version>.sql` (rename to the new version), extend
   the fixture DDL until `TestFixtureSchemaMatchesUpstreamSnapshot` passes,
   and update `docs/storage-schema-<version>.md`.

A weekly CI job (`.github/workflows/upstream-drift.yml`) runs the drift
script, and its `release-notice` job opens a tracking issue automatically
when a new stable release lands — the cadence has a trigger, not just a
memory.

Last verified: **v0.92.0 @ 559ec80** (2026-09-07, todos probe + guard + drift
script added; upstream stats command read — its GetUsageByModel only counts
message rows per model/provider and never sums session-level fields per
model, so our model-breakdown CTE comment stands unchanged; files table
confirmed still written by v0.92.0 via internal/history). The weekly CI job
also self-tests the guard (`--self-test`) and diffs the generated schema
snapshot. Note: our crush#3576 comment fix merged 2026-09-08 but AFTER the
v0.92.0 cut — the corrected millis comments first ship in the next release
(evidence: `docs/upstream-filings.md`).

Optional: `CRUSH_DATA_REAL_DATA_DIR=<dir> go test -run 'TestAllAPIOnRealDatabase|TestStatsParityWithCrushDailySQLOnRealDatabase|TestPartDiscriminatorsCensusShape'` opens a real crush.db read-only (`TestAllAPIOnRealDatabase` sweeps every read API; its Stats is day-filtered by design — all-time Stats DISTINCTs the whole messages table and starves on production-sized DBs under a live writer; skipped under `-short`). Or run everything at once via `scripts/verify-all.sh`. Re-run after ANY source change — not just scan/probe code (a stats.go ORDER BY once changed real-read behavior).

## Docs-health cadence (T33)

Run a docs-health pass (TODO_LIST ↔ FEATURES ↔ CHANGELOG ↔ reality sync;
annotate stale status reports) after **every release** and after **any
session that touches 50+ list items**. The 2026-08-16 proposal for this
rule never landed, and the 2026-09-08 audit found a Critical split brain
that had lived 3 weeks after the last "full sync" — the absence of a
cadence rule was the root cause. A pass takes minutes; skipping it is how
documentation rots silently.

## Architecture (single root package `crushdata`)

| File        | Role                                                                                                                                                                                                                                                                                                                                                                                                |
| ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| discover.go | projects.json registry + `crush projects --json` CLI fallback (stderr capture) + dedupe (many paths → one data_dir)                                                                                                                                                                                                                                                                                 |
| db.go       | read-only open (`mode=ro&_txlock=immediate` + `busy_timeout(5000)` + `query_only(1)` pragmas, 1 conn) + ErrDatabaseNotFound/ErrUnsupportedSchema                                                                                                                                                                                                                                                  |
| schema.go   | capability probing via pragma_table_info — THE drift defense                                                                                                                                                                                                                                                                                                                                        |
| sessions.go | SessionFilter{ByID, Day, ParentID, RootOnly, Limit} + capability-substituted SQL                                                                                                                                                                                                                                                                                                                    |
| parts.go    | sealed Part interface: Text/Reasoning/ToolCall/ToolResult/Finish/ShellCommand/Unknown; strict `DecodeParts` vs tolerant `decodeParts` (bad entry → UnknownPart)                                                                                                                                                                                                                                     |
| rows.go     | `collectRows[T]` generic: iterate rows, scan each into T, collect, verify `rows.Err()` — the one row-collection path every query uses                                                                                                                                                                                                                                                               |
| messages.go | Messages(sessionID) ordered by insertion (`rowid` — NOT `created_at, id`: message IDs are UUIDv4 random, so id order is arbitrary within a same-second created_at tie; verified inversion-free against created_at on real DBs); tolerant part decode (bad entry → UnknownPart, unparseable array → nil Parts); IterMessages (iter.Seq2) streams the same rows via the shared scanMessage; ReadFiles |
| agents.go   | AgentGraph: ONE `WITH RECURSIVE` query per subtree (CTE generates rows through depth 65 so the cap still errors) + in-memory preorder; depth cap 64; flat fallback pre-column                                                                                                                                                                                                                       |
| stats.go    | day aggregates; model-breakdown CTE has the double-count trap — see comment there                                                                                                                                                                                                                                                                                                                   |
| todos.go    | Todo/TodoStatus + DecodeTodos: decodes Session.Todos raw JSON; shape pinned by a real-data census                                                                                                                                                                                                                                                                                                   |

Non-Go surfaces: `scripts/check-vendor-hash.sh` (drift guard),
`scripts/check-upstream-drift.sh` (migration-set + schema-snapshot guard,
with `--self-test` doctored fixture), `scripts/genschema` (generates the
schema snapshot from pinned migrations), `scripts/check-upstream-status.sh`
(filing-campaign monitor), `scripts/check-upstream-release.sh` (opens the
verification-cadence issue on new stable releases; driven by the
upstream-drift workflow's release-notice job), `scripts/check-root-package-main.sh`
(CI build-breaker guard), `scripts/verify-all.sh` (single-command full
verification), `.github/workflows/` (ci, release, fuzz, bench,
flake-update, upstream-drift — all tagged to pinned action SHAs),
`docs/benchmarks/baseline-benchmarks.txt`
(benchstat baseline for the bench.yml trend; regenerate via
`go test -bench . -count=6 | tee …`), `example_test.go` (runnable examples),
`docs/storage-schema-v0.92.0.md` + `.sql` (per-release verified schema
snapshot; guarded by `TestFixtureSchemaMatchesUpstreamSnapshot`),
`docs/upstream-filings.md` (ledger of everything filed on other repos),
`docs/ecosystem-implementation-review.md` (source-verified comparison of
every known Go tool reading crush data — 2 of 6 inherited the
milliseconds-comment lie as `time.UnixMilli` date bugs; both fixed by
our one-line PRs openusage#357 + mnemo#22 on 2026-09-08), and
`docs/upstream-read-access-discussion-draft.md` (authored record of the
posted charmbracelet/crush discussion #3740 — Ideas, 2026-09-08; do not
re-post). `scripts/censusprobe/` is machine-local scratch tooling (T10
parts-type canary; prints by design, registry path defaults to
GlobalDataDir, override via CENSUSPROBE_REGISTRY) — it is
path-excluded in `.golangci.yml` for print/style rules; do NOT "fix" its
fmt.Print calls, and never put `package main` files in the repo root
(they break the single-package `crushdata` build —
`scripts/check-root-package-main.sh` and CI now catch this).

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
  capability-gating blocks in sessions.go/stats.go/agents.go — the sites
  need different string pairs (aliased/plain/qualified), so a helper would
  just restate the conditional; (2) the `dayFilter, args := dayArgs(day)` +
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
- **Concurrent sessions are normal** (an auto-commit daemon plus multiple
  agents work this tree simultaneously). Before editing a file, re-read it
  (`git status` + view) — a stale read is the #1 edit-failure cause, and a
  foreign diff is not yours to revert. Make surgical, additive edits;
  never `git restore`/revert work you did not author; when two sessions
  converge on the same task, keep the better result and integrate rather
  than overwrite. `scripts/verify-all.sh` re-establishes a known-green
  baseline after any interleaved foreign change.
- **External-filing checklist** (every PR/Issue/Discussion/comment on
  another repo; the filing campaign's hard-won rules):
  1. verify-then-file — confirm every claim against source in THIS session
     before writing it into the body;
  2. no forward promises in bodies ("will follow up with X" rot the moment
     priorities shift — link what exists or say nothing);
  3. edit-after-posting is a required step: re-read the posted body and fix
     rendering/description drift immediately;
  4. re-verify every cross-claim (links, states of other PRs, numbers) in
     the FINAL body immediately before posting;
  5. record what was filed where in `docs/upstream-filings.md`, and track
     responses via `scripts/check-upstream-status.sh` (TODO_LIST T35).

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
- **Two lint binaries diverge: `nix run .#lint` is NOT CI's lint.** CI runs
  `go tool golangci-lint run --timeout=5m ./...` (go.mod `tool` pin);
  the nix lint app is a different build. Seen 2026-09-08: nix lint reported
  0 issues while all three CI legs went red on 3 goconst findings
  (`todos`/`messages` literals from the todos-probe work). Gate on the
  go-tool command — it is byte-identical to CI. This also resolves the old
  "goconst occurrence counting" mystery (2026-08-16 07:47 report f/10):
  it was binary/version skew, not counting mechanics; note goconst's
  per-string counts shift as you extract constants (fixing `messages`
  pushed `sessions` over the threshold in the same run).
- **Lint per file while writing tests** (`golangci-lint run <file>_test.go`),
  not after a large batch — but NOT for multi-file test packages: per-file
  linting splits cross-file helpers (typecheck "undefined: openFixture"
  false positives), so lint the whole package when helpers are shared;
  per-file is safe for non-test files. And never trust `cmd | tail`
  without `pipefail`.
- **golangci-lint's cache can serve ANOTHER tree's findings**: linting a
  copy of the repo (same module path) right after linting the original
  reported the original's absolute paths and findings as if they were the
  copy's (seen 2026-08-16: a cache-hit G701 that a clean-cache rerun could
  not reproduce). `golangci-lint cache clean` before judging lint results
  across trees or claiming reproducibility.
- **gosec suppressions live in `.golangci.yml` exclusions, never line
  nolints**: gosec's G7xx taint findings fire non-deterministically across
  runs and platforms (root cause: `maxCallerEdges` + unordered CHA edges,
  tracked upstream as securego/gosec#1712; our data point:
  securego/gosec/issues/1712#issuecomment-5305416042), and a sleeping
  finding turns its `//nolint:gosec` "unused" (nolintlint → red leg).
  Config-level file+rule-ID exclusions are platform-stable; keep the
  rationale as a plain comment at the site.

## Storage facts (reverse-engineered, upstream has no docs)

**Verified against charmbracelet/crush v0.92.0 (commit 559ec80, checked
2026-09-07) by reading upstream source**: every fact below matches the
upstream internal/db/migrations, internal/projects/projects.go,
internal/config/load.go (GlobalConfigData), internal/session/session.go
(Todo/TodoStatus/CreateAgentToolSessionID), and internal/message/message.go
(marshalParts/unmarshalParts). Upstream now pins its schema in-source via
goose migrations + sqlc (sqlc.yaml under internal/db), which supersedes
migration-comment archaeology — the comments still claim milliseconds while
the update_sessions_updated_at trigger writes strftime('%s','now') (seconds)
and the CLI renders time.Unix(CreatedAt, 0).

- Registry: `<global>/projects.json` — `{projects: [{path, data_dir,
  last_accessed}]}` (RFC3339Nano timestamps); global dir =
  CRUSH_GLOBAL_DATA → XDG_DATA_HOME/crush → ~/.local/share/crush (Windows:
  %LOCALAPPDATA%\crush with USERPROFILE fallback), same order upstream uses.
- DB: `<data_dir>/crush.db`, tables sessions/messages/read_files **plus
  `files`** (initial migration: id, session_id, path, content, version,
  created_at, updated_at — file snapshots, intentionally not exposed here;
  the fixture DDL now declares it, guarded against the generated snapshot).
  Sessions also carry `summary_message_id`; messages carry
  `is_summary_message` — both unread by this library (additive, harmless).
- Parts envelope: `[{"type":..., "data":{...}}]` with 8 upstream
  discriminators: reasoning, text, image_url, binary, tool_call,
  tool_result, finish, shell_command. image_url/binary pass through as
  `UnknownPart`.
- Agent child session IDs look like `messageID$$toolCallID`.
- CLI `crush projects --json` prints JSON on **stderr**.
- Todos column: JSON array of `{content, status, active_form}` items;
  statuses pending/in_progress/completed. Pinned by a census of 71,747
  items across all 287 DBs in the local registry (2026-08-16) — zero
  malformed, zero extra keys. When Crush changes the shape,
  `TestDecodeTodosCensusShape` is the tripwire.
- Parts census (2026-09-08): across 243 readable registry DBs (50k-row /
  256MiB ceilings per DB) — 5,223,870 entries, zero unparseable, zero
  discriminators outside the 8 above (image_url never observed in recent
  windows; shell_command is rare at 5). Standing tripwire:
  `TestPartDiscriminatorsCensusShape` (per-DB) and
  `TestPartDiscriminatorsCensusRegistry` (whole registry, env
  `CRUSH_DATA_REAL_REGISTRY`, per-DB 60s timeout). Slow-DB note: a running
  mindwalk holds ~250 registry DBs open; census calls overlapping its scan
  windows stall (14 skipped 2026-09-08) — transient contention, not DB
  pathologies (the same DBs census in under a second when idle, and
  busy_timeout(5000) bounds pure lock waits).
- Real consumer adoption: `crush-daily` decodes per-session todos
  (pending/in_progress/completed counts in `ProjectDailySummary.TodoStats`)
  and iterates messages via `DB.IterMessages` to count part kinds
  (`TextPart`/`ReasoningPart`/`ToolCallPart`/`ToolResultPart`/`FinishPart`/
  `ShellCommandPart`/`UnknownPart`) into
  `ProjectDailySummary.MessagePartStats`. Census-pinned shape is therefore
  exercised against production data, not just synthetic.
- Data-dir quirk seen once (2026-07-28, trashed 2026-09-08): 0-byte files
  named `crush.db?_loc=auto` / `crush.db?_loc=auto&_time_format=sqlite` —
  some Go tool passed a mattn-style DSN string as a literal file path
  (SQLite only parses `?params` with a `file:` prefix, so the whole string
  became the filename). Harmless to this library, which builds proper URIs
  (`file:%s?%s`, db.go); trash on sight.
