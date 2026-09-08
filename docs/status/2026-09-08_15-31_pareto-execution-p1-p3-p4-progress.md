# Status Report — Pareto Execution: P1 Done, P3 Deep In Progress, P4 Started, 2026-09-08 15:31

**Session mandate:** execute `docs/planning/2026-09-08_06-10_pareto-post-audit-drift-hardening.md`
step by step (P1 onward), verify each step, keep going until done. This report
is a point-in-time snapshot requested mid-execution.

**Input state:** master == origin @ `575d1aa`, tree clean, CI green
(run 34185981600, all three legs + nix + bench trend).

**Current state:** local master **4 daemon commits ahead** of origin
(`cfe4bf9..0842dd1`, unpushed — no push authorization this session). A
**concurrent agent session is active in this tree** (see (d)6 and (e)).

---

## a) FULLY DONE

### P1 — post-push verification & T3 closure (all 5 micro steps)

1. **CI green observed** on `575d1aa` (head at session start): CI
   34185981600 — ubuntu/macos/windows + `nix flake check` + Benchmark trend
   all success. Also green one commit earlier on `f6cd473` (34185818702).
2. **`flake-update.yml` dispatched** (workflow_dispatch) → run 34191306038
   **green end-to-end in 1m20s** (Install Nix → Update flake.lock → Verify
   the update still builds → Open pull request, all ✓).
3. **No PR is the CORRECT outcome**: the lock is already fresh — pinned
   nixpkgs `dc5d91f84032` **is** the current nixos-unstable HEAD (verified
   via GitHub API, committed 2026-09-07T03:47Z; lock refreshed locally
   09-07 22:02, commit `ce61ab4`). `nix flake update` produced zero diff →
   create-pull-request exits 0 without a branch. Caveat recorded: the
   bot-push write path itself was not exercised (nothing to push); first
   real exercise = next scheduled run when nixpkgs has moved (2026-10-01
   04:23 UTC).
4. **T3 retired** in TODO_LIST (row deleted, IDs never renumbered).

### P3 — census tripwire, the parts that work

- `realDataDir(t)` helper extracted; the 3rd copy of the env-fallback block
  (realdata/sessions tests) deduplicated onto it.
- `partCensus` + `knownPartKinds` (the 8 v0.92.0 discriminators) +
  `unknownKinds()` (empty-type entries excluded — they are not drift).
- `censusPartDiscriminators`: **Go-side parse** (raw parts streamed
  newest-first via `ORDER BY rowid DESC`, payloads decoded as
  `json.RawMessage`), bounded by **two ceilings** — `partCensusWindow`
  (50,000 newest rows, bounds the rowid-index walk) and
  `partCensusByteBudget` (256 MiB, bounds payload I/O), with
  `CRUSH_DATA_CENSUS_FULL=1` as archaeology mode.
- **`TestPartDiscriminatorsCensusShape` (single-DB tripwire): PASS, 0.91 s
  on the 5 GB `/home/lars/projects/.crush` DB** — 128,208 entries,
  246,000,997 bytes, complete, 0 unparseable, kinds
  `{binary=51 finish=49998 reasoning=11365 text=9844 tool_call=28472
  tool_result=28478}`. Histogram **cross-validated identical** to an earlier
  SQLite-json_each run of the same window.
- **`TestPartDiscriminatorsCensusDetectsUnknown` (fixture negative): PASS**
  — synthetic `hologram` discriminator fails loudly; text + typeless
  entries pass through correctly. Runs in normal CI, all platforms.
- **p3.6 synergy:** `TestAllAPIOnRealDatabase` sweep now counts decoded part
  kinds (`decodedPartKind`, incl. `unknown:<type>` bucket) in its log line.
- **Lint: 0 issues** with the CI-identical
  `go tool golangci-lint run --timeout=5m ./...` (after fixing 4 whitespace
  findings: nlreturn ×2, wsl_v5 ×2).

### P4 — started

- `BenchmarkIterMessages` written (stress_test.go), mirroring
  `BenchmarkMessages`' fixture exactly (2000 messages, same parts), with a
  parts-count sink so the stream cannot be optimized away.

---

## b) PARTIALLY DONE

### P3 — the registry-wide census FAILS (the open front)

`TestPartDiscriminatorsCensusRegistry` (env `CRUSH_DATA_REAL_REGISTRY`) is
implemented but **times out**: two full runs died at the 10-minute test
timeout. Per-DB probing (30 s context timeout per database, 315 data dirs)
isolated the cause:

- **~11 registry databases hang >30 s each** even with the byte budget in
  place: timesheets, wise-go, browser-history, project-discovery-sdk,
  projects-management-automation, emeet-pixyd, art-dupl, segment-buffer,
  clean-wizard, golangci-lint-auto-configure, CreditReformBilanzampel.
- Healthy databases are fine and NOT byte-proportional: e.g. primeXchange
  256 MB in 12.9 s, ast-state-analyzer 131 MB in 19.8 s, accountability-
  system 60 MB in 6.4 s — so the hangs are a per-DB pathology (candidates:
  live-writer lock contention, stale/huge WAL replay at open, pathological
  single rows, planner not using the rowid seek), not volume.
- **Fix direction (not yet implemented):** per-database
  `context.WithTimeout` (e.g., 60 s) + skip-and-log in the registry test —
  one slow/locked DB must not sink the sweep. Root-causing the 11 hangers
  is a separate investigation (question g/1).

### P4 — benchmark written, not yet run

p4.2–p4.6 outstanding: lint already green for the file, but the bench
suite has not been executed, `docs/benchmarks/baseline-benchmarks.txt` not
regenerated (`-count=6`), benchstat parse not verified, citations not
refreshed.

### Documentation for P3 (p3.7) — deliberately deferred

FEATURES row + AGENTS storage-facts line wait until the registry census is
green, per verify-then-annotate (the single-DB numbers above ARE verified
this session and can be cited).

---

## c) NOT STARTED

- **P5** lint/tooling hygiene (T22 exhaustruct_v5, T29 shellcheck, T32
  -cover gate + stale numbers)
- **P6** docs truth pack (T13 schema snapshot, T14 doc.go envelope, T19
  Register() citation, T33 docs-health cadence rule)
- **P7** fixture-test pack (T11/T12/T25/T27)
- **P2** annotation fidelity + archived-ref sweep (T30/T31)
- **P8** stray-file investigation (T21) — files confirmed present:
  `/home/lars/projects/.crush/crush.db?_loc=auto` AND
  `crush.db?_loc=auto&_time_format=sqlite` (both 0 bytes, Jul 28 13:41)
- **P9** decision-bundle evidence (p9.1–p9.3) + user call
- **P10** Stats-vs-crush-daily cross-check (T16)
- **P11** release-watch Action (T17)
- **P13** observables (T34 first scheduled drift run Mon 2026-09-14; T18
  per-session fuzz runs — not yet run this session)
- Full canonical gate + real-data rule for the current code state (last
  full-suite run predates the final census rewrite; quick subsets are green)

---

## d) TOTALLY FUCKED UP (honest ledger)

1. **First census design shipped unmeasured** — SQLite `json_each` over the
   whole 5 GB DB: test died at the 601 s timeout. I then wrote a probe and
   learned parse-vs-walk costs in 90 s. Order inverted: probe FIRST, wire
   into a test second.
2. **Registry census shipped unmeasured — twice.** Two 10-minute timeout
   runs before the per-DB probe revealed the hanging-database class. The
   .crush DB (fast, thin rows) was a bad stand-in for 315 heterogeneous
   DBs (fat rows up to 1.7 MB; ~11 pathological).
3. **The byte budget does not bound time** — a hang is not byte-volume; I
   designed for the failure mode I had already seen, not the one I hadn't.
4. **Probe file lived in the repo root across daemon-commit windows** —
   `package main` in the root broke the single-package build for a moment
   and the concurrent session had to relocate it to
   `scripts/censusprobe/main.go` and document the rule (AGENTS now says:
   never put package-main files in the repo root). My "trash it quickly"
   mitigation was not quick enough.
5. **Per-file lint trap** — `golangci-lint run file1.go file2.go` splits
   cross-file test helpers (typecheck: undefined openFixture/SessionFilter).
   Wasted a run discovering that the AGENTS "lint per file while writing
   tests" advice itself is wrong for multi-file test packages.
6. **Missed that a concurrent session went active mid-execution** — it
   edited `.golangci.yml`, AGENTS.md, my `realdata_test.go` (a whitespace
   fix), moved my probe, and filed upstream artifacts (see (e)4). I noticed
   only via daemon-commit archaeology at report time. Guardrail existed in
   the handoff summary; I didn't re-check `git log` between write batches.

---

## e) WHAT WE SHOULD IMPROVE

1. **Probe-before-wire**: any new query shape touching real data gets a
   timed probe on ≥2 representative databases (incl. one pathological)
   before it becomes a test.
2. **Every long test gets a per-unit timeout** — registry sweeps iterate
   external state; one hostile database must degrade to a skip+log, never a
   suite timeout.
3. **Scratch programs belong under `scripts/` with a package clause that
   cannot collide** (rule now in AGENTS via the concurrent session).
4. **Coordination with the concurrent session**: re-read `git log`/status
   before AND after each write batch (daemon races were known; concurrent
   *content* changes were the surprise).
5. **Fix the AGENTS lint gotcha** (per-file linting of test files breaks)
   to "lint the whole package, or per-file for non-test files only".
6. Canonical gate should run after each MEDIUM task, not be deferred to a
   final pass (I deferred; a mid-session regression would surface late).
7. Census learnings worth pinning in AGENTS storage facts once final:
   recent-rows are fat (≤1.7 MB single parts values), registry DBs are
   heterogeneous, .crush is NOT representative.

---

## f) NEXT (roughly priority-ordered, session-borne + plan remainder)

1. Registry census: per-DB `context.WithTimeout` (60 s) + skip-and-log; re-run to green
2. Root-cause ONE hanging DB (WAL size/lock/row size on timesheets et al.)
3. Re-run full canonical gate + real-data rule on final census code
4. p3.7: FEATURES census row + AGENTS storage-facts histogram line (verified numbers above)
5. TODO_LIST: T10 annotated (single-DB tripwire green; registry pending), not retired yet
6. CHANGELOG entry for census tripwire (code change, unlike doc-only)
7. Validate concurrent session's `scripts/censusprobe/main.go` passes `go build ./...`/vet (it's now in the module tree)
8. P4 p4.3: `go test -bench . -count=6 | tee docs/benchmarks/baseline-benchmarks.txt`
9. P4 p4.4: benchstat parses; sanity-check deltas vs old baseline
10. P4 p4.5: refresh FEATURES bench citation; check bench.yml target list
11. P5 p5.1–p5.2: `exhaustruct` → `exhaustruct_v5`; verify zero deprecation warnings
12. P5 p5.3: shellcheck into flake.nix devShell
13. P5 p5.4: shellcheck over scripts/*.sh; fix or annotate findings
14. P5 p5.5: `-cover` gate variant in AGENTS; refresh FEATURES coverage number
15. P5 p5.6: refresh bench "observed green" date
16. P5-born: correct the AGENTS per-file-lint gotcha (see (e)5)
17. P6 p6.1: `docs/storage-schema-v0.92.0.md` snapshot
18. P6 p6.2: cross-check snapshot vs `schema_drift_test.go` guard list
19. P6 p6.3: doc.go parts-envelope + 8-discriminator paragraph
20. P6 p6.4: `discover.go` upstream `projects.Register()` LastAccessed-desc citation
21. P6 p6.5: AGENTS docs-health cadence rule (T33)
22. P6 p6.6: doc-links gate
23. P7 p7.1: T11 CRUSH_GLOBAL_DATA-is-a-directory fixture
24. P7 p7.2: T12 empty-registry CLI fallback test
25. P7 p7.3: T25 last_accessed UTC round-trip pin
26. P7 p7.4: T27 unknown top-level-keys tolerance test
27. P7 p7.6: gate subset + real-data rule
28. P2 p2.1–p2.4: restore ~15 compressed annotation bodies (02:13 b/c, 08:50 d/e5–e10, 04:20 b-tails)
29. P2 p2.5: T31 archived→archived ref sweep + doc-links
30. P8: identify creator of the two `crush.db?_loc=auto*` files (mtime Jul 28 13:41, 0 bytes)
31. P8: trash them; document finding
32. P9 p9.1: real-data prevalence of `is_summary_message=1` / `summary_message_id`
33. P9 p9.2: day-filter behavior with summary rows (query, don't guess)
34. P9 p9.3: draft T15/T28/summary-fields options with tradeoffs
35. P9 p9.4: USER CALL (see g/3)
36. P10: Stats vs crush-daily collector on the same real DB (needs crush-daily checkout)
37. P10: repeat on a second DB (size-class diversity)
38. P10: pin parity as env-gated test or recipe; retire T16
39. P11: author release-watch workflow; pin action SHAs; actionlint
40. P11: dispatch-verify (needs push authorization)
41. T34: observe first SCHEDULED upstream-drift run (Mon 2026-09-14 03:47 UTC)
42. T18: 30 s fuzz runs (DecodeParts/DecodeTodos) this session
43. Push decision for the accumulating daemon commits (see g/2)
44. Re-read the concurrent session's `docs/status/2026-09-08_filing-campaign-plan.md` before touching any P12/T20 item (overlap risk)
45. Consider one overnight `CRUSH_DATA_CENSUS_FULL=1` registry census to pin the all-time histogram (todos-census precedent)
46. TODO_LIST refresh at session end (T3 retired already; T24 note; new items from this report)
47. Consider documenting the hanging-DB findings upstream if root cause is a crush bug (candidate for the filing campaign)
48. Fix T23 orphan: TODO_LIST T23 (`nix flake check --all-systems`) is missing from the plan's mapping table — plan claims zero orphans (see g/3)
49. Annotate THIS report's items as they complete (docs-health discipline)
50. Update the plan file's P3 section with the two-ceiling + timeout design once green (plan-vs-reality drift)

---

## g) QUESTIONS (cannot self-answer)

1. **Registry-census policy:** is per-DB timeout (60 s) + skip-and-log an
   acceptable standing tripwire, or do you want the ~11 hanging databases
   root-caused first (potentially a live-writer lock or WAL pathology — and
   possibly an upstream reportable bug) before the test ships?
2. **Push:** local master is 4 daemon commits ahead of origin and more will
   accumulate (census + benchmark + docs). Push when green (previous session
   had explicit authorization; this one does not), or hold everything for a
   single authorized push at session end?
3. **Plan gap T23:** TODO_LIST T23 (add `nix flake check --all-systems` to
   CI) was never mapped into the Pareto plan (mapping table claims zero
   orphans). Do you want it executed this session (aarch64/darwin legs add
   CI minutes), parked, or dropped?

---

*Verify-then-annotate discipline: every "green/PASS" above corresponds to a
command that exited 0 in this session. The registry census failures are
reported as failures.*
