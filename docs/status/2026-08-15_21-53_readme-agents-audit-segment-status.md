# Status Report — 2026-08-15 21:53

**Session scope:** Continuation of the 2026-08-15 full-code-review session.
Since the last status report (21:44): the README/AGENTS "superb?" audit and
its on-the-spot fixes, plus the TODO_LIST filename correction from the
previous report. This report covers only this session's run and what it
noticed. Baseline: `16260fe` → daemon landed `74dd031` (pushed, tagged
v0.1.1). Working tree right now: README.md, AGENTS.md, TODO_LIST.md modified
+ this docs/status/ directory — all mine, uncommitted (daemon owns commits).

**Verification state:** `go build ./...` green after doc edits (run even for
docs-only changes). Full gate (race/lint/flake) was green at 21:44 on
`74dd031`; nothing code-side changed since — docs and markdown only.

---

## a) FULLY DONE

1. **Full code review (22 Go files + all meta files)** — every file read;
   verdict: production-ready for scope. (Prior segment.)
2. **Bug 1 fixed + regression-pinned:** SessionFilter args/placeholder order
   drift (`TestSessionsByIDComposesWithOtherFilters`). (Prior.)
3. **Bug 2 fixed + regression-pinned:** AgentGraph sibling order by
   created_at, not reversed updated_at
   (`TestAgentGraphSiblingsOrderedByCreatedNotUpdated`). (Prior.)
4. **doc.go day-filter drift fixed; dead test helpers deleted; discover.go
   no-op removed; vendorHash corrected; CHANGELOG ### Fixed entries;
   TODO_LIST.md created; plan + review HTML artifacts written.** (Prior.)
5. **Real-database smoke test run** against `./.crush` — pass, live schema
   all-capabilities, 3 sessions. (Prior segment, during 21:44 report.)
6. **TODO_LIST.md report-reference corrected** (21:30 → 21-38 filename) —
   self-caught error from the previous report's (d) list. (This segment.)
7. **README/AGENTS audit (this segment's main ask):** honest "not yet
   superb" verdict delivered, then three README gaps fixed on the spot —
   (a) day-filter timezone semantics added to the Timestamps design bullet
   (the subtlest public behavior was absent from the whole file);
   (b) `nix run .#lint` added to Development (the ~90-linter gate was
   undiscoverable); (c) Windows `%LOCALAPPDATA%` registry path added next to
   the Unix one (GlobalDataDir supports it; README claimed Unix-only
   implicitly). AGENTS.md gained a "Tooling gotchas" section capturing this
   session's two tooling landmines: go.sum↔vendorHash coupling (the 16260fe
   breakage class) and nix source-filter blindness to untracked .go files
   (the misleading "undefined: collectRows" class).
8. **Deliberate non-change documented:** README `strings.Repeat` snippet
   intentionally fragmentary — dismissed as a nit, not fixed. Judgment
   logged rather than silently skipped.

## b) PARTIALLY DONE

1. **README at ~8.5–9/10, AGENTS.md at ~9/10.** The audit fixed everything
   found; "superb" as a *verified* claim would need the remaining nits
   below plus (f) items 33–34. Remaining known nits: README has no
   day-filter timezone note in the Quick start comments (only in Design);
   no coverage badge; CONTRIBUTING not re-checked this segment.
2. **AGENTS.md learnings capture:** tooling gotchas now recorded (item a-7),
   but the concurrent-session verification protocol and the
   changelog-tag-state rule are still not in any memory file (they live in
   this report and the 21:44 report only).
3. **Stale-editor noise:** the LSP/gopls session still surfaces
   `repro_test.go` paralleltest warnings and unusedfunc infos for files
   deleted/edited minutes ago — the project compiles and lints clean;
   the diagnostics pane is stale. Restarting LSP mid-session not yet done.

## c) NOT STARTED

1. `OpenContext(ctx, dataDir)` — uncancellable Open (ticketed).
2. AgentGraph single-query CTE (N+1 today; fine for realistic graphs).
3. `Session.Todos` typed as JSON (`json.RawMessage` or decoded; next minor).
4. Schema-probe strictness: missing vs. broken indistinguishable.
5. `example_test.go` — zero runnable examples despite `testableexamples`.
6. FEATURES.md / ROADMAP.md — repo has neither.
7. Real fuzz execution (only seed corpus runs in CI) and bench baseline.
8. HTML artifact render validation (assembled head+body, never opened).
9. CI nix job / go.sum↔vendorHash pre-commit guard (e-1 from 21:44 — still
   the highest-value infra gap).
10. CLI-fallback noisy-stderr robustness check (`CombinedOutput` parses the
    whole blob as JSON; any stray log line breaks the fallback parser;
    unverified suspicion).

## d) TOTALLY FUCKED UP!

Nothing new this segment. Cumulative honest ledger from this session:

1. **Wrong report filename in TODO_LIST.md** (21-30 vs 21-38) — caught in
   the 21:44 report, fixed at 21:46. Cause: wrote from the plan file's
   timestamp instead of the report's.
2. **Three lint iterations on my own new tests** (paralleltest, golines ×2,
   godox on "bug:" in a comment) — sloppy against conventions I'd already
   read. All fixed; 0 issues final.
3. **Generic `/tmp/review-plan` scratch dir collided** with another
   session's output — preserved under `.stale-other-project`, zero repo
   damage, but the collision was foreseeable. Future: `mktemp -d`.
4. **CHANGELOG appended under `[0.1.1]` without verifying tag state** —
   landed correctly only because the daemon tagged v0.1.1 at the new commit
   afterward. Right outcome, unverified decision.
5. **Report-freshness race** — closing message of the review said "nothing
   committed" ~60s before the daemon committed. Fixed behaviorally in this
   segment: state re-verified at report time (see header).

## e) WHAT WE SHOULD IMPROVE!

1. **CI still has no nix job** — the vendorHash rot class stays invisible
   to CI. Cheapest high-value infra fix remaining.
2. **Memory-file discipline for process learnings** — technical gotchas now
   in AGENTS.md, but process rules (concurrent-session protocol,
   changelog-tag-state, report-freshness verification) have no home. They
   evaporate with the session; consider a short AGENTS.md "session
   protocol" section or global memory.
3. **Docs verification depth** — "is it superb?" deserved a lint-style
   mechanical pass too (link check, dead-code-reference scan) instead of
   pure reading. Cheap to add.
4. **Fuzz/bench are decorative** — seeds and benchmarks exist; a nightly
   60s fuzz + benchstat trend would make them load-bearing.
5. **CLI fallback suspicion unresolved** — noticed twice now, still
   untested. Small, bounded, should just be verified or debunked.
6. **Diagnostics hygiene** — restart LSP after file deletions so the pane
   stops lying; stale warnings waste reviewer attention.

## f) NEXT — up to 50, ordered by impact

**Release & CI:**
1. Add `nix flake check` job to CI.
2. Decide v0.1.1 tag integrity question (q2 from 21:44 — still open).
3. Tag-driven GitHub Release workflow (notes from CHANGELOG).
4. go.sum↔vendorHash pre-commit guard script (kills the rot class at source).
5. Real 60s `FuzzDecodeParts` round; wire as nightly job.
6. `go test -shuffle=on` alongside `-race` in CI.
7. Benchstat trend artifact for `BenchmarkSessionsList`.
8. Renovate/dependabot for go.mod + flake.lock.
9. Monthly scheduled `nix flake update` PR.

**Correctness/robustness (existing code):**
10. Verify CLI-fallback parser against noisy stderr; extract JSON substring
    before parsing if confirmed.
11. Test `Message.FinishedAt` populated path (fixture never sets it).
12. Test `Stats` day filter in non-UTC location.
13. Test `AgentGraph` depth-cap path (`ErrGraphDepthExceeded`).
14. Test dedupe tie-break (equal LastAccessed keeps first).
15. `collectRows` `rows.Err()` branch test.
16. `fillHourHistogram` out-of-range hour guard test.
17. Registry unreadable (chmod 000) triggers CLI fallback, not just missing.
18. `ParseProjectsOutput("null")` behavior pin.
19. Wide fan-out AgentGraph stress (100 children).
20. Concurrent read alongside a WAL writer integration test.
21. Zero-table SQLite file explicit case (cheap, removes doubt).

**API evolution (bundle into a next minor):**
22. `OpenContext(ctx, dataDir)` + `Open` delegating.
23. `Session.Todos` → `json.RawMessage` or decoded type.
24. Schema-probe error surfacing (strictness decision).
25. AgentGraph CTE variant — only if a consumer reports it hot.

**Docs & repo hygiene:**
26. README: timezone note inside the Quick start snippet comments (not just
    Design).
27. CONTRIBUTING: stats-parity contract warning (don't "improve" SQL
    without updating `TestStatsParityWithCrushDailySQL`).
28. `example_test.go` with 2–3 runnable examples.
29. FEATURES.md + ROADMAP.md via docs-health BUILD.
30. Coverage badge if CI exposes it.
31. Validate both HTML artifacts render.
32. pkg.go.dev verification once the proxy crawls v0.1.1.
33. README link/anchor check (mechanical pass).
34. CONTRIBUTING re-read vs current state (was not re-checked this segment).

**Tidiness:**
35. Re-evaluate blanket `_test.go` `unused` lint exclusion (dead helpers
    gone; exclusion may be stale).
36. Restart LSP / clear stale diagnostics habit after mid-session deletes.
37. `windowsLocalAppData` GOOS-gated unit test (low).
38. `exhaustruct` exclusion audit after any new dependency.
39. Consider `docs/DOMAIN_LANGUAGE.md` — likely premature at this size;
    decide consciously, default skip.
40. Re-verify reports' claims before any future "waiting for instructions"
    handoff (behavioral rule from d-5 — keep enforcing).

(35–40 are the honest bottom of the barrel; stop at real value.)

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Unanswered from 21:44 — concurrent session policy:** parallel editor
   sessions worked this repo tonight (verified, not reverted). Intended
   workflow? Should reviews serialize, or is verify-don't-revert the norm?
2. **Unanswered from 21:44 — v0.1.1 tag history:** did v0.1.1 previously
   point at an older commit anyone could have fetched (→ retag = proxy
   poisoning → cut v0.1.2), or was it freshly created tonight? I cannot see
   deleted tag targets from here.
3. **Superb-bar:** for README/AGENTS you asked "superb???" — I applied my
   own bar and stopped at "close, now closer". If you have a specific bar
   (e.g., every snippet runnable, godoc-permalink coverage, a specific doc
   set like FEATURES/ROADMAP mandatory), name it and I'll drive to it
   mechanically.

---

*Point-in-time snapshot. Living work items live in TODO_LIST.md. Generated
by the 2026-08-15 review session (segment 2: README/AGENTS audit).*
