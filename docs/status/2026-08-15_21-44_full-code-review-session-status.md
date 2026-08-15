# Status Report — 2026-08-15 21:44

**Session scope:** Full code review of go-crush-data (user request: "Review
this project!"), executed per the `full-code-review` skill. This report
covers only what this session did and noticed. Baseline commit at session
start: `16260fe`. State at report time: `74dd031` (auto-commit daemon landed
all review work at 21:42, pushed to origin/master, tagged `v0.1.1`).

**Verification state at report time:** `go build` OK · `go test -race` OK
(86.2% coverage) · `golangci-lint` 0 issues · `nix flake check` passes ·
real-database smoke test (`CRUSH_DATA_REAL_DATA_DIR=./.crush`) PASS — live
schema reports all capabilities, 3 sessions read.

---

## a) FULLY DONE

1. **Every source and test file read** (22 Go files + README, CHANGELOG,
   flake.nix, .golangci.yml, CONTRIBUTING, LICENSE, .gitignore).
2. **Baseline established** before any change: build, vet, race tests,
   coverage (86.4%), golangci-lint (~90 linters) — all green.
3. **Bug 1 found and reproduced, then fixed:**
   `SessionFilter` bind-args order ≠ SQL placeholder order; `ByID` combined
   with `Day`/`ParentID` silently returned zero rows. Fix (landed via
   concurrent editor, verified by me line-by-line): query builder returns
   `(query, args)` with condition+placeholder appended in the same branch.
4. **Bug 2 found and reproduced, then fixed:** `AgentGraph` siblings ordered
   by reversed `updated_at` instead of documented `created_at` preorder.
   Fix: `sortByCreated` (slices.SortFunc, ID tiebreak).
5. **Regression tests added** (survive as permanent pins):
   `TestSessionsByIDComposesWithOtherFilters`,
   `TestAgentGraphSiblingsOrderedByCreatedNotUpdated`. Both failed on the
   pre-fix code, pass now.
6. **Documentation drift fixed:** `doc.go` day-filter section now states the
   tested semantics (filter value's own location), matching
   `SessionFilter.Day` instead of contradicting it.
7. **Dead code removed:** `fixtureDB`, `insertLegacySession` (gopls
   unusedfunc flags are gone from the build; editor still shows stale ones).
8. **`discover.go` no-op removed:** `project.DataDir = dataDir` redundant
   re-assignment.
9. **Stale `flake.nix` vendorHash fixed** (pre-existing breakage: 16260fe
   refreshed go.sum without updating the hash; verified via
   `nix build .#default.goModules`, then full `nix flake check` after commit).
10. **CHANGELOG:** `### Fixed` entries for both bugs + doc fix, under 0.1.1.
11. **TODO_LIST.md created** with 4 open ticketed findings + done items.
12. **HTML artifacts written:**
    - Plan: `docs/planning/2026-08-15_21_30-full-code-review-execution.html`
    - Report: `docs/reviews/2026-08-15_21-38_full-code-review.html`
13. **Real-database smoke test run** (post-hoc, during this status check) —
    passes; missed during the review proper, owned here.

## b) PARTIALLY DONE

1. **Review report accuracy:** written at 21:38 with claims that went stale
   within minutes (tree uncommitted, flake check blocked). All claims
   re-verified for this status report; the HTML snapshot itself was not
   amended (point-in-time artifact — annotate later if asked).
2. **AGENTS.md session learnings:** the concurrent editor added their notes
   (rows.go, art-dupl acceptances); my learnings (go.sum↔vendorHash coupling,
   nix source-filter ignoring untracked files, concurrent-session
   verification protocol) are NOT yet recorded. Not started item c) covers
   the remainder.
3. **Test-suite hardening:** regression pins landed, but several coverage
   gaps noticed during the read remain (see f)).

## c) NOT STARTED

1. `OpenContext(ctx, dataDir)` (Open is uncancellable).
2. `AgentGraph` single-query CTE variant (currently N+1; acceptable).
3. `Session.Todos` typed as JSON (breaking-ish; next minor).
4. Schema-probe strictness (missing vs. broken indistinguishable).
5. `example_test.go` (the `testableexamples` linter is enabled; zero
   examples exist).
6. FEATURES.md / ROADMAP.md (repo has neither; arguably premature at this
   size).
7. README timezone-semantics paragraph for day filters (only in godoc).
8. Actual fuzz execution (`-fuzz` with time budget; only seed corpus runs).
9. Benchmark execution / baseline (BenchmarkSessionsList never run).
10. HTML report render validation (files assembled via head+body concat;
    never opened or structurally validated).

## d) TOTALLY FUCKED UP!

Nothing catastrophic — no build break, no data loss, no reverted foreign
work. Three genuine self-inflicted defects, all caught and two fixed:

1. **Wrong filename in TODO_LIST.md** — referenced
   `2026-08-15_21-30_full-code-review.html`; actual file is `21-38`. Fixed
   just now (21:46). Cause: wrote the reference from the plan file's
   timestamp instead of the report's.
2. **Three lint iterations on my own new tests** — paralleltest, golines ×2,
   godox ("bug:" in a comment tripped the TODO scanner). Sloppy first pass
   against conventions I had already read. All fixed; final state 0 issues.
3. **Scratch-dir collision** — `/tmp/review-plan/body.html` contained
   another project's session output (mtime 04:00). I preserved it under
   `.stale-other-project` instead of clobbering, but I had chosen a generic
   path primed to collide. Future: `mktemp -d` or project-unique names.
   Process defect, zero repo damage.

Honorable mention (luck, not rigor): I appended CHANGELOG entries under
`[0.1.1]` without checking tag state first. It landed correctly only because
the daemon tagged `v0.1.1` at the new commit — the decision was unverified
at the time I made it. See question 2.

## e) WHAT WE SHOULD IMPROVE!

1. **CI has no nix job** — the vendorHash breakage (16260fe) sailed through
   CI because CI installs golangci-lint directly and never runs
   `nix flake check`. The flake can rot invisibly. Highest-value gap.
2. **Concurrent-session protocol** — two editors worked the same tree
   tonight. It worked (complementary changes, each verified the other's),
   but nothing enforced that. A read-before-edit re-check on external-mod
   errors should be standard, and heavy concurrent sessions deserve
   serialization by intent.
3. **Changelog discipline** — append entries under `[Unreleased]` unless the
   tag state is verified first. Cheap rule, prevents mis-attribution.
4. **Report-freshness** — verify claims (`git status`, `nix flake check`)
   at the moment of writing final summaries, not minutes before. My closing
   message said "nothing committed" while the daemon committed 60s later.
5. **Validation depth for generated artifacts** — HTML reports and assembled
   files deserve at least a structural sanity check before being referenced.
6. **Fuzz/bench budget** — the suite has seeds and benchmarks; a nightly
   60s fuzz + benchstat trend would make them real rather than decorative.
7. **CLI fallback robustness (noticed, untested):** `queryProjectsCLI` uses
   `CombinedOutput()` and parses the whole blob as JSON — any log line the
   real CLI prints alongside the payload breaks the fallback parser. The
   fake-CLI test only feeds pure JSON. Suspect, needs verification + likely
   substring extraction.
8. **Coverage shifted 86.4% → 86.2%** after the refactor + new tests; not a
   problem, but per-file deltas in CI would make such moves legible.

## f) NEXT — up to 50, ordered by impact

**Release & CI (highest value):**
1. Add `nix flake check` job to CI (closes the flake-rot gap; e-1).
2. Verify v0.1.1 tag integrity & module proxy state; decide on v0.1.2 (see q2).
3. Add tag-driven GitHub Release workflow (notes from CHANGELOG).
4. Run a real 60s fuzz round of `FuzzDecodeParts`; wire as nightly CI job.
5. CI: `go test -shuffle=on` in addition to `-race`.
6. Benchstat trend artifact for `BenchmarkSessionsList` in CI.
7. Renovate/dependabot for go.mod + flake.lock.
8. Scheduled `nix flake update` PR (monthly).

**Correctness/robustness of existing code:**
9. Verify CLI-fallback parser against noisy stderr; extract JSON substring
   before parsing (e-7).
10. Test `Message.FinishedAt` populated path (fixture never sets it non-null).
11. Test `Stats` day filter in non-UTC location (mirror of sessions pin).
12. Test `AgentGraph` depth-cap path (`ErrGraphDepthExceeded`) — 64+ chain.
13. Test dedupe tie-break (equal `LastAccessed` keeps first).
14. `collectRows` `rows.Err()` branch test (80% covered).
15. `fillHourHistogram` out-of-range hour guard test.
16. Test registry unreadable (chmod 000) triggers CLI fallback, not just
    missing file.
17. `ParseProjectsOutput("null")` behavior pin (nil, no error?).
18. Wide fan-out AgentGraph stress (100 children).
19. Concurrent-read-while-writer-has-WAL integration test (proves the
    mode=ro + txlock claim under contention).
20. `TestOpen` on a valid SQLite file with no tables at all → currently
    `ErrUnsupportedSchema`? foreign-database case covers it; add explicit
    zero-table case anyway (cheap).

**API evolution (next minor, bundled):**
21. `OpenContext(ctx, dataDir)` + `Open` delegating.
22. `Session.Todos` → `json.RawMessage` or decoded type.
23. Schema-probe error surfacing (strictness decision).
24. Consider `Role.IsValid()` helper? Optional/YAGNI — decide consciously.
25. AgentGraph CTE single-query variant — only if a consumer reports it hot.

**Docs & repo hygiene:**
26. Record session learnings in AGENTS.md (vendorHash coupling, nix
    source-filter untracked-file gotcha, concurrent-edit protocol).
27. README: day-filter timezone semantics paragraph.
28. README: add `nix run .#lint` to Development.
29. `example_test.go` with 2-3 runnable examples.
30. CONTRIBUTING: note the stats-parity contract (don't "improve" the SQL
    without updating `TestStatsParityWithCrushDailySQL`).
31. FEATURES.md + ROADMAP.md via docs-health BUILD (small, honest).
32. Coverage badge if CI exposes it.
33. Validate the two HTML artifacts render (open browser / structural check).
34. pkg.go.dev verification once proxy crawls v0.1.1.

**Tidiness:**
35. Re-evaluate the blanket `_test.go` `unused` lint exclusion (dead helpers
    are gone; exclusion may be stale now — verify noise before removing).
36. Remove stale LSP diagnostics annoyance: restart LSP after file deletion
    mid-session (process habit, not repo change).
37. `go.sum` ↔ `vendorHash` pre-commit check script (fails if go.sum changed
    without flake.nix hash) — kills the e-1 class at the source.
38. Add `.gitattributes`/`.editorconfig` note? Already present — verify
    consistency only. (Skip if no findings.)
39. Consider `windowsLocalAppData` GOOS-gated unit test (low).
40. Blanket `exhaustruct` exclusions audit (os/exec.Cmd only today — fine;
    re-check after new deps).

(36–40 are honest bottom-of-barrel items; stop at real value, don't pad.)

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Concurrent session policy:** another editor session worked this repo
   simultaneously tonight (sessions/agents/messages/stats/rows.go). I
   verified rather than reverted. Going forward: are parallel sessions on
   this repo intended, and should reviews serialize (or is last-writer-
   verifies fine)?
2. **Tag integrity:** `v0.1.1` currently resolves to tonight's `74dd031`
   and origin/master contains it. Did `v0.1.1` previously point at an older
   commit that anyone could already have fetched (→ retag = module-proxy
   poisoning, v0.1.2 advisable), or was it freshly created tonight (→
   nothing to do)? I cannot see deleted tag targets from here.
3. **Commit policy for me:** hard rule says never commit unless you say
   "commit", but this repo's daemon auto-commits (it committed tonight's
   work before I finished reporting). Should session work be left to the
   daemon always, or do you want explicit commits at milestones?

---

*Point-in-time snapshot. Living work items live in TODO_LIST.md. Generated
by the 2026-08-15 full-code-review session.*
