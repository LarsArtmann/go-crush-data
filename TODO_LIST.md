# TODO List

Short- and mid-term improvement tasks. Consolidated 2026-08-15 from the full
code review and both status reports; the full Pareto-ranked execution plan
with fine-grained breakdown lives at
`docs/planning/2026-08-15_22-00-consolidated-roadmap-execution.md` (snapshot).
Tier 1 = highest leverage.

## Open — Tier 1 (1% → 51%)

- [ ] **CI nix job + shuffle** — add `nix flake check` and
  `go test -shuffle=on` to CI; the vendorHash rot class is currently
  invisible to CI. ~60 min.
- [ ] **go.sum↔vendorHash drift guard** — CI step that fails when go.sum
  changes without the flake hash. ~45 min.
- [ ] **CLI-fallback stderr robustness** — verify (then fix) that
  `CombinedOutput` parsing survives log noise around the JSON payload;
  suspected live defect. ~60 min.

## Open — Tier 2 (4% → 64%)

- [ ] **Test sweep A (row paths)** — FinishedAt populated, collectRows
  rows.Err, hour-guard, zero-table open. ~90 min.
- [ ] **Test sweep B (filter/graph paths)** — non-UTC stats day, depth cap,
  dedupe tie-break, 100-child fan-out, read-vs-WAL-writer. ~100 min.
- [ ] **Registry pins** — `ParseProjectsOutput("null")`, chmod-000 registry
  fallback. ~40 min.
- [ ] **GitHub Release workflow** — tag-driven, notes from CHANGELOG.
  (~Release-integrity memo: origin v0.1.1 verified = 74dd031, no retag,
  v0.1.2 not needed.) ~60 min.

## Open — Tier 3 (20% → 80%)

- [ ] **API bundle v0.2.0** — `OpenContext(ctx, dir)`; `Session.Todos` →
  `json.RawMessage`; schema-probe strictness. ~4 h total.
- [ ] **Runnable examples** — `example_test.go` (Discover, parts
  type-switch, Stats). ~60 min.
- [ ] **Docs batch** — README Quick start timezone note, CONTRIBUTING
  parity-contract warning, link check. ~45 min.
- [ ] **FEATURES.md + ROADMAP.md** via docs-health BUILD. ~90 min.
- [ ] **Nightly fuzz + benchstat trend** — make fuzz/bench load-bearing.
  ~95 min.

## Open — Tier 4 (rest → 100%)

- [ ] Coverage badge + pkg.go.dev verification. ~30 min.
- [ ] HTML artifact validation (2 files). ~20 min.
- [ ] Dependency automation (Renovate, monthly flake update). ~60 min.
- [ ] Tidiness — `_test.go` unused exclusion re-eval, windowsLocalAppData
  GOOS test, exhaustruct audit. ~45 min.
- [ ] Record non-decisions in ROADMAP (DOMAIN_LANGUAGE skip, AgentGraph CTE
  only-if-hot). ~12 min.

## Done

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
