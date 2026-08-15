# TODO List

Short- and mid-term improvement tasks. Rewritten 2026-08-15 after the
consolidated roadmap execution (C1–C21, all tiers done and gate-verified —
see CHANGELOG `[Unreleased]` and
`docs/status/2026-08-15_22-44_roadmap-t1-t4-execution-status.md`).
Harvested from that report's "next things" list; Pareto-ordered.

## Pending user decisions

- [ ] **Cut v0.2.0?** `[Unreleased]` holds the breaking `Todos` change plus
  `OpenContext` and probe strictness; tag per RELEASING.md also gives the
  release workflow its first real run.
- [ ] **Coverage badge: live or static?** Current static "≥85% enforced" badge
  states the invariant but shows no number (codecov/artifact-backed
  alternative ~30 min).

## High — release & automation observability (contingent on v0.2.0)

- [ ] Observe release workflow on the v0.2.0 tag; tick RELEASING checklist. 10m
- [ ] Observe first nightly fuzz run (03:17 UTC). 5m
- [ ] Observe first bench-trend run; sanity-check the summary render. 5m
- [ ] Install/enable the Renovate app on the repo (config validates; inert
  until enabled). 5m
- [ ] Observe first monthly flake-lock PR. 5m

## Medium — hardening the new surfaces

- [ ] Pin bench.yml's benchstat version (currently `@latest`). 10m
- [ ] Release workflow `workflow_dispatch` dry-run support. 10m
- [ ] CI: pin golangci-lint via go.mod `tool` directive instead of
  `go install @v2.12.2`. 15m
- [ ] CI: add windows + macos matrix legs (also gives `windowsLocalAppData`
  a real-Windows run). 30m
- [ ] Benchmarks: add `BenchmarkMessages`, `BenchmarkAgentGraph` to the
  trend. 20m
- [ ] Live coverage badge via codecov/artifact (replaces static). 30m
- [ ] pkg.go.dev re-verification after the v0.2.0 crawl. 5m

## Medium — test pins for documented-but-unproven behavior

- [ ] README: mention `OpenContext` + `Todos` raw JSON in Design bullets. 10m
- [ ] Examples: `AgentGraph` + `ReadFiles` examples. 20m
- [ ] Pin `dedupeProjects` zero-timestamp "sorts last" doc claim with a test. 10m
- [ ] Pin `Session(byID)` → `ErrSessionNotFound` if not already covered. 5m
- [ ] Pin `ReadFiles` empty-path filtering. 10m
- [ ] Pin combined `Day + Limit` filter composition. 10m
- [ ] Pin messages legacy-schema NULL substitution (model/provider/finished). 10m
- [ ] WAL concurrency: exercise `Messages` concurrently, not just `Sessions`. 15m
- [ ] Document/pin `Models`/`Providers` DISTINCT ordering (undefined today). 10m
- [ ] Stats day filter on legacy schema combo test. 10m
- [ ] `TestSessionsOnRealDatabase`: assert schema capabilities explicitly. 10m
- [ ] CLI-fallback: fake CLI emitting exit-nonzero + partial JSON → error path
  pin. 10m
- [ ] `DiscoverProjects` result ordering pin (sorted by DataDir). 5m
- [ ] `extractJSONObject`: handle brace-inside-string noise edge explicitly
  (documented limit). 15m
- [ ] Fuzz `loadRegistry` JSON shape (registryFile). 15m
- [ ] Fuzz: mine nightly artifacts for corpus seeds. ongoing

## Low — polish

- [ ] CODEOWNERS file. 5m
- [ ] SECURITY.md (reporting policy; read-only lib, tiny surface). 15m
- [ ] CONTRIBUTING: document `go run benchstat@latest` fallback next to nix
  absence. 5m
- [ ] Add `-count=2` to documented local race command. 2m
- [ ] Consider darwin global-dir handling audit (upstream parity). 20m
- [ ] Sweep `//nolint` directives for staleness quarterly. 15m
- [ ] gosec G701 taint false positive: upstream minimal repro + issue. 30m
- [ ] Evaluate `nix flake update` cadence vs Renovate nix manager overlap. 10m

(Raw ideas — `DecodeTodos`, streaming iterator, registry watching — live in
ROADMAP.md, not here.)

## Done

- [x] Consolidated roadmap C1–C21 (all four tiers), gates 1–3 + final gate
  green — 2026-08-15; details in CHANGELOG `[Unreleased]` and the status
  report above.
- [x] TODO_LIST sync, AGENTS.md update, final gate re-run, real-DB smoke
  test, coverage measurement — 2026-08-15 (this session's closure pass).
- [x] Fix `SessionFilter` condition/argument order drift (`sessions.go`) —
  fixed 2026-08-15 during the review; regression test
  `TestSessionsByIDComposesWithOtherFilters`.
- [x] Fix `AgentGraph` sibling ordering to follow `created_at`, not reversed
  `updated_at` (`agents.go`) — fixed 2026-08-15 during the review; regression
  test `TestAgentGraphSiblingsOrderedByCreatedNotUpdated`.
- [x] Align `doc.go` day-filter documentation with the tested semantics —
  fixed 2026-08-15 during the review.
- [x] Delete dead test helpers `fixtureDB`, `insertLegacySession` — done
  2026-08-15 during the review.
- [x] README/AGENTS audit fixes (timezone semantics, `nix run .#lint`,
  Windows path, tooling gotchas) — done 2026-08-15.
- [x] Verify remote tag integrity (origin v0.1.1 = 74dd031, no retag) —
  done 2026-08-15 22:00.
