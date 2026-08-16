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

1. **Review report accuracy:** ~~written at 21:38 with claims that went stale
   within minutes (tree uncommitted, flake check blocked).~~ All claims
   re-verified for this status report; the HTML snapshot itself was not
   amended (point-in-time artifact — annotate later if asked) → annotated +
   archived 2026-08-16.
2. **AGENTS.md session learnings:** ~~the concurrent editor added their notes
   (rows.go, art-dupl acceptances); my learnings (go.sum↔vendorHash coupling,
   nix source-filter ignoring untracked files, concurrent-session
   verification protocol) are NOT yet recorded.~~ done at `88012fe` +
   `232ff1f` — all in AGENTS.md (Tooling gotchas / Process rules).
3. **Test-suite hardening:** ~~regression pins landed, but several coverage
   gaps noticed during the read remain (see f)).~~ done at `9b4d346` — the
   C4/C5 sweeps closed every f/9–f/20 gap.

## c) NOT STARTED

1. ~~`OpenContext(ctx, dataDir)` (Open is uncancellable).~~ done at `9b4d346` (C11), shipped in v0.2.0
2. ~~`AgentGraph` single-query CTE variant (currently N+1; acceptable).~~ **Won't implement — recorded non-decision (ROADMAP.md: only if a consumer profiles it hot)**
3. ~~`Session.Todos` typed as JSON (breaking-ish; next minor).~~ done at `9b4d346` (C12, `json.RawMessage`), shipped in v0.2.0
4. ~~Schema-probe strictness (missing vs. broken indistinguishable).~~ done at `9b4d346` (C13)
5. ~~`example_test.go` (the `testableexamples` linter is enabled; zero
   examples exist).~~ done at `9b4d346` (C14) + `eabdcb1` (6 examples)
6. ~~FEATURES.md / ROADMAP.md (repo has neither; arguably premature at this
   size).~~ done at `9b4d346` (C16)
7. ~~README timezone-semantics paragraph for day filters (only in godoc).~~ done at `74dd031` (Design bullet + Quick-start comment)
8. ~~Actual fuzz execution (`-fuzz` with time budget; only seed corpus runs).~~ done at `9b4d346` (C9: 12.2M + 5.7M execs, 0 crashers; nightly workflow)
9. ~~Benchmark execution / baseline (BenchmarkSessionsList never run).~~ done at `9b4d346` (C10) + `581658b` (3-benchmark baseline)
10. ~~HTML report render validation (files assembled via head+body concat;
    never opened or structurally validated).~~ done at `9b4d346` (C18)

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

1. ~~**CI has no nix job**~~ done at `9b4d346` (C1) — `flake` job in ci.yml
2. ~~**Concurrent-session protocol**~~ done — verify-don't-revert is the norm
   and is encoded in the global agent rules ("respect existing changes")
3. ~~**Changelog discipline**~~ done — policy recorded in CHANGELOG.md +
   RELEASING.md (`232ff1f`): `[Unreleased]` unless tag state is verified
4. ~~**Report-freshness**~~ done — "verify-then-annotate" rule in AGENTS.md
   (`232ff1f`)
5. ~~**Validation depth for generated artifacts**~~ done at `9b4d346` (C18:
   structural HTML validation)
6. ~~**Fuzz/bench budget**~~ done at `9b4d346` (C9 nightly fuzz + C10
   benchstat trend) — no longer decorative
7. ~~**CLI fallback robustness**~~ done at `9b4d346` (C3: defect CONFIRMED
   then fixed via `extractJSONObject`)
8. ~~**Coverage shifted 86.4% → 86.2%** — per-file deltas in CI would make
   such moves legible.~~ done — CI uploads the full coverage HTML report as
   an artifact (`754d32c`), which carries per-file detail

## f) NEXT — up to 50, ordered by impact

**Release & CI (highest value):**
1. ~~Add `nix flake check` job to CI (closes the flake-rot gap; e-1).~~ done at `9b4d346` (C1)
2. ~~Verify v0.1.1 tag integrity & module proxy state; decide on v0.1.2 (see q2).~~ done — origin v0.1.1 = `74dd031`, fresh tag, no retag; v0.1.2 not needed (decision logged in RELEASING.md, `9b4d346`)
3. ~~Add tag-driven GitHub Release workflow (notes from CHANGELOG).~~ done at `9b4d346` (C8); first real run green on v0.2.0 (`6948933`)
4. ~~Run a real 60s fuzz round of `FuzzDecodeParts`; wire as nightly CI job.~~ done at `9b4d346` (C9: 12.2M execs, 0 crashers)
5. ~~CI: `go test -shuffle=on` in addition to `-race`.~~ done at `9b4d346` (C1)
6. ~~Benchstat trend artifact for `BenchmarkSessionsList` in CI.~~ done at `9b4d346` (C10) + `581658b` (3 benchmarks)
7. ~~Renovate/dependabot for go.mod + flake.lock.~~ done at `9b4d346` (C19 config) — app install still pending (TODO_LIST, external)
8. ~~Scheduled `nix flake update` PR (monthly).~~ done at `9b4d346` (C19 workflow) — first scheduled run not yet observed (TODO_LIST)

**Correctness/robustness of existing code:**
9. ~~Verify CLI-fallback parser against noisy stderr; extract JSON substring
   before parsing (e-7).~~ done at `9b4d346` (C3 — defect confirmed, then fixed)
10. ~~Test `Message.FinishedAt` populated path (fixture never sets it non-null).~~ done at `9b4d346` (`TestMessagesFinishedAtPopulated`)
11. ~~Test `Stats` day filter in non-UTC location (mirror of sessions pin).~~ done at `9b4d346` (C5)
12. ~~Test `AgentGraph` depth-cap path (`ErrGraphDepthExceeded`) — 64+ chain.~~ done at `9b4d346` (65-chain)
13. ~~Test dedupe tie-break (equal `LastAccessed` keeps first).~~ done at `9b4d346` (C5)
14. ~~`collectRows` `rows.Err()` branch test (80% covered).~~ done at `9b4d346` (C4)
15. ~~`fillHourHistogram` out-of-range hour guard test.~~ done at `9b4d346` (C4)
16. ~~Test registry unreadable (chmod 000) triggers CLI fallback, not just
    missing file.~~ done at `9b4d346` (C6, root-skip guard)
17. ~~`ParseProjectsOutput("null")` behavior pin (nil, no error?).~~ done at `9b4d346` (C6)
18. ~~Wide fan-out AgentGraph stress (100 children).~~ done at `9b4d346` (C5)
19. ~~Concurrent-read-while-writer-has-WAL integration test (proves the
    mode=ro + txlock claim under contention).~~ done at `9b4d346` (Sessions) + `0822d70` (Messages)
20. ~~`TestOpen` on a valid SQLite file with no tables at all → currently
    `ErrUnsupportedSchema`? foreign-database case covers it; add explicit
    zero-table case anyway (cheap).~~ done — already covered by `TestOpenUnsupportedSchema` (verified during C4; F14 closed as covered)

**API evolution (next minor, bundled):**
21. ~~`OpenContext(ctx, dataDir)` + `Open` delegating.~~ done at `9b4d346` (C11)
22. ~~`Session.Todos` → `json.RawMessage` or decoded type.~~ done at `9b4d346` (C12, `json.RawMessage`)
23. ~~Schema-probe error surfacing (strictness decision).~~ done at `9b4d346` (C13)
24. ~~Consider `Role.IsValid()` helper? Optional/YAGNI — decide consciously.~~ **Won't implement — YAGNI; unknown roles already pass through as-is (`Role` doc)**
25. ~~AgentGraph CTE single-query variant — only if a consumer reports it hot.~~ **Won't implement — recorded non-decision (ROADMAP.md)**

**Docs & repo hygiene:**
26. ~~Record session learnings in AGENTS.md (vendorHash coupling, nix
    source-filter untracked-file gotcha, concurrent-edit protocol).~~ done at `88012fe` + `232ff1f`
27. ~~README: day-filter timezone semantics paragraph.~~ done at `74dd031`
28. ~~README: add `nix run .#lint` to Development.~~ done at `bcc0a50`
29. ~~`example_test.go` with 2-3 runnable examples.~~ done at `9b4d346` (4) + `eabdcb1` (2 more)
30. ~~CONTRIBUTING: note the stats-parity contract (don't "improve" the SQL
    without updating `TestStatsParityWithCrushDailySQL`).~~ done at `9b4d346` (C15)
31. ~~FEATURES.md + ROADMAP.md via docs-health BUILD (small, honest).~~ done at `9b4d346` (C16)
32. ~~Coverage badge if CI exposes it.~~ done at `9b4d346` (C17, static ≥85% badge); live badge is a recorded non-decision
33. ~~Validate the two HTML artifacts render (open browser / structural check).~~ done at `9b4d346` (C18)
34. ~~pkg.go.dev verification once proxy crawls v0.1.1.~~ done at `9b4d346` (C17) — v0.2.0 re-verification still pending (TODO_LIST)

**Tidiness:**
35. ~~Re-evaluate the blanket `_test.go` `unused` lint exclusion (dead helpers
    are gone; exclusion may be stale now — verify noise before removing).~~ done at `9b4d346` (C20 — removed, 0 dead helpers behind it)
36. ~~Remove stale LSP diagnostics annoyance: restart LSP after file deletion
    mid-session (process habit, not repo change).~~ **NOT-DO — process habit, no repo artifact**
37. ~~`go.sum` ↔ `vendorHash` pre-commit check script (fails if go.sum changed
    without flake.nix hash) — kills the e-1 class at the source.~~ done at `9b4d346` (C2: `scripts/check-vendor-hash.sh`, wired into CI)
38. ~~Add `.gitattributes`/`.editorconfig` note? Already present — verify
    consistency only. (Skip if no findings.)~~ done — both present, no findings
39. ~~Consider `windowsLocalAppData` GOOS-gated unit test (low).~~ done at `9b4d346` (C20 env-only test); real-Windows coverage via the CI matrix legs (`79c9720`, green at `c7482e2`)
40. ~~Blanket `exhaustruct` exclusions audit (os/exec.Cmd only today — fine;
    re-check after new deps).~~ done at `9b4d346` (C20 — all load-bearing)

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Concurrent session policy:** another editor session worked this repo
   simultaneously tonight (sessions/agents/messages/stats/rows.go). I
   verified rather than reverted. Going forward: are parallel sessions on
   this repo intended, and should reviews serialize (or is last-writer-
   verifies fine)?
   → **Resolved de-facto: parallel sessions happen here; verify-don't-revert
   is the norm** (also encoded in the global agent rules: never revert
   changes you didn't author).
2. **Tag integrity:** `v0.1.1` currently resolves to tonight's `74dd031`
   and origin/master contains it. Did `v0.1.1` previously point at an older
   commit that anyone could already have fetched (→ retag = module-proxy
   poisoning, v0.1.2 advisable), or was it freshly created tonight (→
   nothing to do)? I cannot see deleted tag targets from here.
   → **Resolved: fresh tag, no retag, nothing to do** — origin `v0.1.1` =
   `74dd031` verified against local the next session; decision logged in
   RELEASING.md.
3. **Commit policy for me:** hard rule says never commit unless you say
   "commit", but this repo's daemon auto-commits (it committed tonight's
   work before I finished reporting). Should session work be left to the
   daemon always, or do you want explicit commits at milestones?
   → **Resolved de-facto: the daemon owns commits; sessions leave work
   uncommitted.** Because daemon messages can misdescribe diffs, CHANGELOG.md
   (not `git log`) is the record of what shipped — see RELEASING.md and
   AGENTS.md "Process rules".

---

## Resolution (2026-08-16)

Every numbered item in b), c), e), and f) is closed: 35 shipped (mostly the
C1–C21 batch at `9b4d346`, plus `88012fe`, `232ff1f`, `eabdcb1`, `581658b`,
`754d32c`, `79c9720`, `c7482e2`), 5 consciously not done with reasons
inline. All 3 questions resolved. Still-open follow-ups live in TODO_LIST.md
(Renovate app install, scheduled-workflow first observations, pkg.go.dev
v0.2.0 check). Archived by the 2026-08-16 docs-health audit.

*Point-in-time snapshot. Living work items live in TODO_LIST.md. Generated
by the 2026-08-15 full-code-review session.*
