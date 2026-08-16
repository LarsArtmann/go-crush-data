# Features

Honest inventory of what this library does, by status. Code is the source of
truth; every row cites its evidence. Update rows in place when status changes.

| Status | Feature | Evidence |
|---|---|---|
| FULLY_FUNCTIONAL | Project discovery from the projects.json registry, deduplicated to one project per data directory (newest access wins, ties keep the first entry) | `discover.go:237` `dedupeProjects` |
| FULLY_FUNCTIONAL | `crush projects --json` CLI fallback; tolerates log noise around the JSON payload | `discover.go:165` `queryProjectsCLI`, `discover.go:223` `extractJSONObject` |
| FULLY_FUNCTIONAL | `ParseProjectsOutput` — public parser for raw CLI output | `discover.go:191` |
| FULLY_FUNCTIONAL | Read-only open (`mode=ro`, single connection, `_txlock=immediate`); byte-identical after reads | `db.go:108` `openSQLite`, `db_test.go` `TestOpenIsReadOnly` |
| FULLY_FUNCTIONAL | `OpenContext` — open with a caller context bounding the schema probes | `db.go:58` |
| FULLY_FUNCTIONAL | Schema capability probing (cost, parent_session_id, model, provider, finished_at, read_files) with strict error surfacing | `schema.go:82` `probeSchema` |
| FULLY_FUNCTIONAL | `Schema.MissingColumns` — user-facing drift warning list | `schema.go:44` |
| FULLY_FUNCTIONAL | Session listing with filters: ByID, Day (filter-location semantics), ParentID, RootOnly, Limit | `sessions.go:45` |
| FULLY_FUNCTIONAL | Session fetch by ID (`ErrSessionNotFound`) | `sessions.go:66` |
| FULLY_FUNCTIONAL | `Session.Todos` as raw JSON (`json.RawMessage`, nil for NULL) | `types.go:39`, `sessions.go:187` |
| FULLY_FUNCTIONAL | `DecodeTodos` — typed todos decoding (census-pinned shape, drift-tolerant) | `todos.go:47`, `todos_test.go` `TestDecodeTodosCensusShape` |
| FULLY_FUNCTIONAL | Message listing ordered by `created_at, id` | `messages.go:21` |
| FULLY_FUNCTIONAL | `DB.IterMessages` — streaming message iteration (`iter.Seq2`), same rows and order as `Messages` | `messages.go:49`, `messages_test.go` `TestIterMessagesMatchesMessages` |
| FULLY_FUNCTIONAL | Tolerant parts decoding: one malformed part degrades to `UnknownPart` keeping its raw payload and siblings; wholly unparseable parts yield nil Parts | `messages.go:54` |
| FULLY_FUNCTIONAL | Sealed `Part` set: Text, Reasoning, ToolCall, ToolResult, Finish, ShellCommand, Unknown (forward-compatible passthrough) | `parts.go:9`–`78` |
| FULLY_FUNCTIONAL | `DecodeParts` — strict public parts decoder | `parts.go` |
| FULLY_FUNCTIONAL | `ReadFiles` — files the agent opened during a session | `messages.go:90` |
| FULLY_FUNCTIONAL | Subagent graphs via a single `WITH RECURSIVE` query: preorder by creation time, depth cap 64 (`ErrGraphDepthExceeded` on cycles) | `agents.go:23`, `agents.go:108` |
| FULLY_FUNCTIONAL | Day activity stats with the crush-daily parity contract (`TestStatsParityWithCrushDailySQL`) | `stats.go:28` |
| FULLY_FUNCTIONAL | Models/Providers in Stats sorted ascending (ORDER BY on DISTINCT query) | `stats.go:121` |
| FULLY_FUNCTIONAL | Per-model breakdown with session-level double-count protection CTE; `Stats` field docs state the 20-row caps (`TestStatsCapsAt20`) | `stats.go:249`, `types.go` |
| FULLY_FUNCTIONAL | CI matrix: ubuntu, windows, macos — green on master and on the v0.2.1 tag (tests run twice per leg via `-count=2`; `go mod verify` guards the module cache). The immutable v0.2.0 tag's Windows leg is permanently red — test code only; erratum in CHANGELOG `[0.2.1]` | `.github/workflows/ci.yml` |
| FULLY_FUNCTIONAL | CI: vet/build/race/shuffle/coverage gate (≥85%), golangci-lint, govulncheck | `.github/workflows/ci.yml`; local measure 2026-08-15: 87.8% statements |
| FULLY_FUNCTIONAL | CI: `nix flake check` job (vendorHash freshness, format) | `.github/workflows/ci.yml` `flake` job |
| FULLY_FUNCTIONAL | go.sum ↔ vendorHash drift guard with tamper-proven failure mode | `scripts/check-vendor-hash.sh` |
| FULLY_FUNCTIONAL | Doc-truth guard: markdown links resolve, reference-style links defined, `file:line` citations in range — wired into gate and CI | `scripts/check-doc-links.sh` |
| FULLY_FUNCTIONAL | Fuzz targets for all four parsers (DecodeParts, DecodeTodos, ParseProjectsOutput, loadRegistry) with seed corpora; nightly matrix runs each | `fuzz_test.go`, `.github/workflows/fuzz.yml` |
| FULLY_FUNCTIONAL | Committed benchmark baseline (Sessions, Messages, AgentGraph) + local benchstat workflow | `docs/benchmarks/baseline-benchmarks.txt` |
| FULLY_FUNCTIONAL | Runnable godoc examples covering discovery, sessions, messages, stats, agent graph, read files | `example_test.go` |
| FULLY_FUNCTIONAL | Tag-driven GitHub Release workflow — observed green on the v0.2.0 and v0.2.1 tags (notes match the CHANGELOG section incl. the erratum); dry-run path exercised via `workflow_dispatch` | `.github/workflows/release.yml`, `RELEASING.md` |
| PARTIALLY_FUNCTIONAL | Nightly fuzz workflow — written and actionlint-clean, first scheduled run pending (03:17 UTC) | `.github/workflows/fuzz.yml` |
| FULLY_FUNCTIONAL | Benchmark-trend CI job comparing pushes against the baseline — observed green on v0.2.0 push | `.github/workflows/bench.yml` |
| FULLY_FUNCTIONAL | Coverage HTML artifact uploaded on CI (ubuntu leg); static ≥85% badge on README | `.github/workflows/ci.yml`, `README.md` |
| PARTIALLY_FUNCTIONAL | Renovate dependency automation — config ready, app not yet installed | `renovate.json`, TODO_LIST |
