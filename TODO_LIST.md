# TODO List

Short- and mid-term improvement tasks. Rewritten 2026-08-15 after the
consolidated roadmap execution (C1–C21, all tiers done and gate-verified —
see CHANGELOG `[0.2.0]` and
`docs/status/2026-08-15_22-44_roadmap-t1-t4-execution-status.md`).
v0.2.0 shipped 2026-08-16 — see CHANGELOG and the Pareto plan at
`docs/planning/2026-08-15_23-09_pareto-v0.2.0-ship-and-harden.md`.

## Pending (external dependencies)

- [ ] Install/enable the Renovate app on the repo (config validates; inert
  until enabled via GitHub App UI). 5m
- [ ] Observe first nightly fuzz run (03:17 UTC). 5m
- [ ] Observe first monthly flake-lock PR. 5m
- [ ] Fuzz: mine nightly artifacts for corpus seeds. ongoing
- [ ] gosec G701 taint false positive: upstream minimal repro + issue. 30m
  (nolint is sufficient; upstream repro deferred to a separate session)

## Done

All items from the v0.2.0 Pareto plan (M1–M13, 60 micro tasks) were
executed and gate-verified on 2026-08-16.

### v0.2.0 release (M1)

- [x] Cut v0.2.0: CHANGELOG section, annotated tag, push, release workflow
  green, GitHub Release published, module proxy serving, clean `go get`
  validated. 2026-08-16.

### Truth & safety (M4)

- [x] Add `.crush/` to the repo `.gitignore`. 2m
- [x] RELEASING.md: daemon-commit unreliability note (CHANGELOG = truth). 5m
- [x] AGENTS.md: verify-then-annotate + diff-daemon-commits rules. 5m
- [x] CHANGELOG policy: doc-only changes not changelogged. 5m
- [x] CI: upload `go tool cover -html` artifact. 10m
- [x] Audit daemon commit messages vs diffs (2427223 clean, 9b4d346
  already flagged). 15m

### Automation enablement (M3)

- [x] Pin benchstat via go.mod `tool` directive (drop `@latest`). 10m
- [x] Pin golangci-lint via go.mod `tool` directive (drop `go install
  @v2.12.2`). 12m
- [x] Release workflow `workflow_dispatch` dry-run support. 10m
- [ ] Renovate app install (external — needs GitHub App UI). 5m

### Test pins (M5, M6)

- [x] Pin `Day + Limit` filter composition. 10m
- [x] Pin `dedupeProjects` zero-timestamp "sorts last" claim. 10m
- [x] Pin `DiscoverProjects` result ordering (sorted by DataDir). 5m
- [x] Pin `ReadFiles` empty-path filtering. 10m
- [x] Document + pin `Models`/`Providers` DISTINCT ordering (added ORDER BY). 10m
- [x] Pin `Session(byID)` → `ErrSessionNotFound` (already covered). 5m
- [x] Pin messages legacy-schema NULL substitution (already covered). 10m
- [x] Stats day filter on legacy schema combo test. 10m
- [x] `TestSessionsOnRealDatabase`: assert schema capabilities explicitly. 10m
- [x] CLI-fallback: exit-nonzero + partial JSON → error-path pin. 10m
- [x] `extractJSONObject`: brace-inside-string edge cases documented + tested. 15m

### Concurrency + fuzz (M7)

- [x] WAL concurrency test for `Messages`. 12m
- [x] Fuzz target: `loadRegistry` JSON shape. 12m
- [ ] Corpus mining pass (deferred — no nightly artifacts yet). ongoing

### Examples + README (M8)

- [x] README Design bullets: `OpenContext` + `Todos` raw JSON. 10m
- [x] `ExampleAgentGraph` with `// Output:` block. 20m
- [x] `ExampleReadFiles` with `// Output:` block. 15m

### CI matrix (M9)

- [x] Add windows-latest + macos-latest matrix legs. 25m
- [x] Platform fallout fixes (shell: bash for awk, vendor guard ubuntu-only). 15m
- [x] Bench workflow observed green on push. 5m
- [ ] Observe matrix runs green on origin (in progress at time of writing). 5m

### Benchmarks (M10)

- [x] Add `BenchmarkMessages`. 15m
- [x] Add `BenchmarkAgentGraph`. 15m
- [x] Regenerate baseline with all three benchmarks. 5m

### Hygiene pack (M11)

- [x] CODEOWNERS file. 5m
- [x] SECURITY.md (reporting policy). 12m
- [x] CONTRIBUTING: benchstat fallback note. 5m
- [x] AGENTS.md: `-count=2` on local race command. 2m

### Coverage visibility (M12)

- [x] CI: upload `cover -html` artifact. 10m
- [x] D2 decision: static badge kept (live badge not worth the dependency). 0m

### Deep audits (M13)

- [x] Daemon commit audit (no new misdescriptions). 15m
- [x] gosec G701: nolint still needed; upstream repro deferred. 12m
- [x] `//nolint` sweep: 13 directives, all with rationales, none stale. 12m
- [x] nix vs Renovate overlap: disabled Renovate nix manager. 10m
- [x] Darwin global-dir audit: matches Crush's XDG conventions; no action. 10m

### Prior sessions

- [x] Consolidated roadmap C1–C21 (all four tiers), gates 1–3 + final gate
  green — 2026-08-15.
- [x] TODO_LIST sync, AGENTS.md update, final gate re-run, real-DB smoke
  test, coverage measurement — 2026-08-15 (closure pass).
- [x] Fix `SessionFilter` condition/argument order drift (`sessions.go`) —
  fixed 2026-08-15; regression test `TestSessionsByIDComposesWithOtherFilters`.
- [x] Fix `AgentGraph` sibling ordering to follow `created_at`, not reversed
  `updated_at` (`agents.go`) — fixed 2026-08-15; regression test
  `TestAgentGraphSiblingsOrderedByCreatedNotUpdated`.
- [x] Align `doc.go` day-filter documentation with the tested semantics —
  fixed 2026-08-15.
- [x] Delete dead test helpers `fixtureDB`, `insertLegacySession` — done
  2026-08-15.
- [x] README/AGENTS audit fixes (timezone semantics, `nix run .#lint`,
  Windows path, tooling gotchas) — done 2026-08-15.
- [x] Verify remote tag integrity (origin v0.1.1 = 74dd031, no retag) —
  done 2026-08-15 22:00.

(Raw ideas — `DecodeTodos`, streaming iterator, registry watching — live in
ROADMAP.md, not here.)
