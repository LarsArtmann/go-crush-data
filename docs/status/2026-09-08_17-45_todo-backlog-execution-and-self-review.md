# Status report — TODO-backlog execution + self-review (2026-09-08 17:45)

Session scope: execute the TODO_LIST backlog end-to-end. A **concurrent
session worked the same tree simultaneously** (auto-commit daemon +
interleaved commits) — parts of the backlog were closed by it; this report
attributes honestly and self-reviews only THIS session's work.

## a) FULLY DONE (this session, verified green unless noted)

**Code / tests** (all green under `scripts/verify-all.sh`: 12/12 steps OK;
`go test -race -shuffle=on ./...` OK; CI-identical lint 0 issues):

1. **T11** `TestDiscoverProjectsGlobalDataDirEnvIsADirectory` — pins
   CRUSH_GLOBAL_DATA-is-a-directory through the real resolution path
   (upstream `load.go` joins filenames into it; verified in pinned clone).
2. **T12** `TestDiscoverProjectsCLIFallbackEmptyRegistry` — empty-registry
   CLI fallback → empty result, no error, via the real fakeCLI exec path
   (both `{"projects":[]}` and empty-output shapes).
3. **T19** dedupe comment now cites upstream `projects.Register()`
   LastAccessed-descending sort (verified: internal/projects/projects.go
   v0.92.0).
4. **T25** `TestRegistryLastAccessedUTCRoundTrip` — upstream writes
   `time.Now().UTC()` RFC3339Nano; nano round-trip + offset→UTC pinned.
5. **T27** `TestDiscoverProjectsToleratesUnknownRegistryKeys`.
6. **T28** summary-message decision **pinned and documented**:
   `is_summary_message = 1` rows count everywhere (parity with collector
   SQL, which never filtered); `TestSummaryMessagesAreCounted` + doc
   comments in messages.go/stats.go.
7. **T16** real-data Stats parity:
   `TestStatsParityWithCrushDailySQLOnRealDatabase` (env-gated, day-filtered
   on newest session); **passed against the live 5 GB local DB**; fixture
   parity check extracted into shared `assertCrushDailyCollectorParity`.
8. **T14** parts envelope documented in `doc.go` (all 8 discriminators,
   type mapping, tolerant vs strict, census tripwire pointer).
9. **T38** fixture-drift killed: new `scripts/genschema` (applies pinned
   goose migrations to in-memory SQLite, honors StatementBegin/End, dumps
   deterministic sqlite_master) → checked-in
   `docs/storage-schema-v0.92.0.sql`; `TestFixtureSchemaMatchesUpstreamSnapshot`
   + negative-path test; fixture DDL gained the missing `files` table;
   `check-upstream-drift.sh` now diffs the snapshot (verified green against
   the pinned clone).

**Scripts / CI / workflows** (shellcheck-clean, actionlint-clean):

10. **T37** drift self-test: `--self-test` runs the guard against a
    checked-in doctored fixture (must exit 1 with the rogue column visible)
    + empty-extraction probe (must exit 2 — the vacuous-pass hole). Green.
11. **T42** `scripts/check-root-package-main.sh` + ubuntu CI step.
12. **T17** `release-notice` job in upstream-drift.yml +
    `scripts/check-upstream-release.sh`: opens one tracking issue per new
    stable release (duplicate-safe). **No-op path verified live**
    ("v0.92.0 is latest"); issue-creation path first exercises in CI.
13. **T41** `scripts/check-upstream-status.sh` — ran as today's T35 pass:
    **crush#3576 MERGED (2026-09-08 07:04 UTC — AFTER the v0.92.0 cut on
    08-31; corrected comments ship next release)**; openusage#357 +
    mnemo#22 open → G1/G2 blocked; discussion #3740 has 1 substantive
    third-party comment (ArcticFox2029, pro-version-field).
14. **T46** `scripts/verify-all.sh` — one command: canonical gate + drift
    self-test + live drift + snapshot + real-data sweeps + opt-in registry
    census, per-step summary. **Full run: 12/12 OK.**
15. **T36** flake lint app passes `"$@"` (verified:
    `nix run .#lint -- --timeout=5m` applies the flag).
16. **T18** 2×30s local fuzz (DecodeParts, DecodeTodos): 3.8M execs each,
    zero crashes.
17. **T21** stray `crush.db?_loc=auto*` files diagnosed: a Go tool passed a
    mattn-style DSN as a literal path (no `file:` prefix → whole string
    became the filename). Quirk recorded in AGENTS; files already gone
    (concurrent session trashed them with the same conclusion).
18. **T24** benchstat old-vs-new independently verified (AgentGraph −78.2%
    time/−75.6% allocs; Messages −6.8%/−32.3%; SessionsList +13.9% —
    regression from the todos-probe/UpdatedAt read columns, expected and
    now baselined; IterMessages new at 4.15ms/op).

**Docs / process**:

19. **T39** AGENTS process rules: concurrent-session convention +
    external-filing checklist (verify-then-file, no forward promises,
    edit-after-posting, re-verify final body, ledger update).
20. **T43** `docs/upstream-filings.md` ledger (8 filings incl. retired
    T9/T20 mapping; #3576 merge date verified against the API).
21. **T13** snapshot doc integrated (concurrent session wrote
    docs/storage-schema-v0.92.0.md; I added the genschema/snapshot/guard
    chain paragraph — no duplication).
22. AGENTS: commands block (verify-all, upstream-status, genschema, guards,
    self-test), cadence step 5 (snapshot refresh), release-notice pointer,
    storage-facts updates, #3576-after-pin note, stale
    `TestSessionsOnRealDatabase` command fixed.
23. TODO_LIST trimmed to genuinely-blocked items; T35 rewritten around the
    one-command pass; CHANGELOG + FEATURES entries for everything shipped.

**Closed by the concurrent session** (verified, integrated, not reverted):
T10 close-out (registry census 243 DBs green + docs), T13 doc body, T22
exhaustruct_v5, T23 --all-systems decision, T29 shellcheck in devShell,
T33 docs-health cadence, T40 censusprobe env var, T44 deletion, deps
refresh (golangci v2.13.2, vendorHash), benchmark baseline regen + CHANGELOG.

## b) PARTIALLY DONE

1. **T24 citation refresh** — FEATURES row updated (by concurrent session);
   CONTRIBUTING.md does NOT mention `verify-all` yet (it should — the
   documented local-verification story changed today).
2. **T13/T38 provenance** — the `.sql` snapshot has no self-describing
   header (pin/date live only in the `.md` and the drift script); genschema
   could emit one.
3. **release-notice job (T17)** — duplicate-check logic verified against
   real `gh` output; `gh issue create` path runs for the first time in CI
   (needs origin green + push first).
4. **Real-cli empty-registry claim (T12)** — the test pins OUR handling of
   the payload shapes; upstream's actual `crush projects --json` behavior
   with an empty registry (prints `{"projects":[]}` vs prints nothing,
   exit code) was never read from upstream source. Low risk (both shapes
   covered), but the TODO's premise is still unverified upstream.

## c) NOT STARTED (deliberately)

1. **T15** `MissingCapabilities()` — needs your call; NOTE: the concurrent
   session's status report (17-43) suggests this is being triaged there.
2. **T45** push master — user action by rule; everything sits local+green.
3. **T1/T7** Renovate app — GitHub UI, yours.
4. **T34** first scheduled drift run — fires 2026-09-14 03:47 UTC.
5. **T35** ongoing daily monitoring — today's pass DONE via the new script;
   the duty itself continues.
6. **T6** fuzz-corpus mining — ongoing.
7. ROADMAP.md untouched (no item required it).

## d) TOTALLY FUCKED UP (self-inflicted, all recovered)

1. **Self-inflicted census timeout failure**: launched the 30m registry
   census while my own verify-all (race suite + nix builds) hammered the
   same machine → run died at exactly 1800s. Correct diagnosis
   (contention), wrong attribution at first ("likely indexer") — it was
   mostly ME. Re-run green in 283s once idle. The 45m budget bump stands
   (prior session hit real contention with 11 DBs), but comments now say
   "self-inflicted" honestly.
2. **discover_test.go comment-merge**: an edit removed the newline between
   a doc comment and `func` — declaration became part of the comment,
   broke the file (caught instantly by diagnostics; fixed in 1 edit).
3. **Snapshot-guard parser took 3 iterations**: v1 had a slice-panic path
   (`[:open]` with open=-1) and checked constraint prefixes AFTER
   firstWord collapsed "PRIMARY KEY"→"PRIMARY" (test caught both). Ironic:
   I built genschema around executing SQL, then hand-rolled a text parser
   for the guard when `pragma_table_info` over an in-memory DB was the
   robust design. Shipped correct, wastefully built.
4. **check-upstream-status.sh v1 shipped with dead debris** (leftover
   `fetch_state` with a broken path) — full rewrite before first run; never
   executed in the bad state.
5. **verify-all.sh unterminated string** (`cd "$repo_root`) — shellcheck
   caught pre-run.
6. **Stale-LSP chasing**: burned cycles arguing with golangci-lint-ls
   diagnostics that lagged my edits — the "independently verify with the
   real binary" rule already existed in global memory; I re-learned it.
7. **Concurrent-session blindness for ~30 min**: the opening `git status`
   showed foreign uncommitted files; I read them as "prior session" and
   only noticed the LIVE session at the `.golangci.yml` mtime clash.
   `git log --since="1 hour ago"` at session start (the exact convention I
   later codified in T39) would have caught it immediately.
8. **T13 double-write**: I authored a full schema doc while the concurrent
   session wrote the same file — the edit-time guard blocked my write
   (no lost update); ~20 min of work fed only one integration paragraph.
9. **CHANGELOG overstatement** (fixed this session, post-hoc): wrote
   "verified on the local registry" for a single-database parity run —
   corrected to "a live local database (the project's own 5 GB crush.db)".

## e) WHAT WE SHOULD IMPROVE

1. **Concurrent-session coordination is still ad-hoc**: two sessions ran
   the same backlog and collided twice (golangci.yml, storage-schema doc).
   A lightweight claim protocol (TODO item → session ID prefix, e.g.
   `T13@B`) or a "claims" section in TODO_LIST would prevent double work.
2. **Guard tests should use the database, not regex**: snapshot-guard
   parsing belongs in SQLite (`pragma_table_info`), same as genschema.
   Text parsers are how the drift class sneaks back in.
3. **Heavy runs need a machine-load gate**: the census self-contention
   would not have happened if verify-all exposed "don't start parallel
   heavy jobs" (or the census waited for the gate to finish — it was
   started BY me in parallel on purpose; sequencing was the fix).
4. **Unverified-in-CI paths**: release-notice creation, drift self-test in
   CI, root-package-main CI step — all first-run-after-push. Expect
   Monday's scheduled run to be the real test; check it (T34 covers the
   scheduled run; add a manual workflow_dispatch check right after push).
5. **CI runs `-count=2`; my local gate ran count=1** — an order-dependent
   flake could pass locally. Run
   `go test -race -shuffle=on -count=2 ./...` once before the v0.4.0 tag.
6. **Windows leg unverified for new tests** (env-dir, UTC round-trip,
   summary-message, snapshot guard): designed cross-platform (t.Setenv,
   jsonString, no POSIX premises) but only CI-windows can prove it.
7. **Docs formatter mangling**: something stripped spaces after inline
   code spans in AGENTS (`(rename` after backtick) — actor unidentified
   (dprint? the other session?). Find and fix the formatter config before
   it corrupts more prose.

## f) NEXT — up to 50, ranked

**Unblock / ship:**

1. **Push master** (T45) — everything green locally; origin still red;
   no new CI job runs until this happens.
2. After push: watch the first CI run of the new steps (root-package-main,
   drift self-test) + bench trend vs new baseline.
3. Manually `workflow_dispatch` upstream-drift.yml post-push; verify both
   jobs incl. release-notice no-op path.
4. Decide T15 (`MissingCapabilities()` vs `MissingColumns()` — being
   triaged in the concurrent session's report; do not double-decide).
5. Cut **v0.4.0** once [Unreleased] settles + green origin (needs approval).
6. Draft v0.4.0 release notes from [Unreleased] sections.

**Upstream / campaign:**

7. Daily T35 pass via `scripts/check-upstream-status.sh`.
8. Address reviewers on openusage#357 / mnemo#22 if comments appear.
9. Decide (with your go-ahead) whether to reply to ArcticFox2029 on #3740 —
   the version-field idea aligns with our census evidence.
10. G1 openusage adoption issue once #357 merges (draft ready in campaign
    plan; re-verify claims at post time).
11. G2 mnemo adoption issue once #22 merges.
12. Watch crush#3581 (zstd): if it lands without an in-band encoding
    marker, file the reader-breakage follow-up (matrix from #3580).
13. Observe first SCHEDULED drift run (T34, Mon 2026-09-14 03:47 UTC).
14. When the next crush stable ships: run the full AGENTS cadence incl. new
    step 5 (genschema snapshot rename + fixture guard + doc refresh).

**Hardening / correctness:**

15. Rewrite `tableColumns` in schema_snapshot_test.go via in-memory SQLite
    `pragma_table_info` (kill the text parser).
16. Run `go test -race -shuffle=on -count=2 ./...` before any tag (CI parity).
17. Verify upstream's real empty-registry `crush projects --json` output in
    the pinned clone (T12 premise).
18. Give `docs/storage-schema-*.sql` a generated header (pin ref + date +
    regen command) via genschema.
19. Consider an adaptive census: detect contention and skip faster instead
    of burning the full 60s per DB.
20. Add `-count=2` (opt-in flag) to verify-all.
21. Identify and fix the docs formatter eating spaces after code spans.
22. Unit-test check-upstream-release.sh's matching logic (extract to a
    function; bash test or shfmt+review pass).

**Docs / bookkeeping:**

23. Add `scripts/verify-all.sh` to CONTRIBUTING.md's verification story.
24. Verify the concurrent session recorded the T23 (--all-systems) decision
    in CHANGELOG; add a one-liner if absent.
25. Run the docs-health pass per the new T33 cadence (this session touched
    23+ items): TODO ↔ FEATURES ↔ CHANGELOG ↔ reality cross-check.
26. Re-measure coverage (`go test -cover ./...`) — new tests moved it.
27. Annotate the 15-31/15-33 status reports whose numbered items are now
    done (docs-health ANNOTATE mode).
28. Update `docs/benchmarks/` citation if bench.yml assumptions changed
    with the new baseline.
29. Investigate root `reports/` dir and the stray `result` entry (noticed
    at session start; never looked inside).
30. ROADMAP.md: check whether the ecosystem plan references need a
    post-campaign refresh (G3 done, #3576 merged).

**Small polish:**

31. check-upstream-status: also print fix-PR review/comment activity (not
    just merge state).
32. genschema: fail loudly when a migration file has neither Up nor Down
    markers (currently silently skipped).
33. verify-all: print per-step durations in the summary.
34. Consider retiring or documenting censusprobe's relationship to the
    registry census test (overlap exists now).
35. Add a fixture with a pre-existing `files` table row to prove fixture
    fidelity end-to-end (currently the table exists but is never written).
36. README: nothing new needed (scripts are dev-facing) — verify that
    claim once after push.
37. Consider `MissingTables()` info on `Schema` as the non-breaking half of
    T15.
38. Add example_test.go coverage only if v0.4.0 adds public API (T15
    outcome decides).
39. Sweep `.golangci.yml` exclusions for ones made obsolete by today's
    changes (e.g. if genschema paths need censusprobe-style carve-outs —
    currently not, but re-check after any new script).
40. Bench: add a BenchmarkStats to the trend (read path changed twice
    since baseline; not in baseline at all).

**External / waiting:**

41. T1/T7: Renovate app install (yours).
42. T6: mine nightly fuzz artifacts for corpus seeds (after a week of
    runs).
43. mindwalk `sdk/go-crush-data` branch PR (parked; needs go-ahead).
44. Re-check deja-vu#2949 / crunch#22 for responses during T35 passes.
45. Post-v0.4.0: pkg.go.dev freshness check for the new doc.go section.

**Meta:**

46. Consider a session-claims convention (see e)1) — add to AGENTS if you
    want it.
47. Set a recurring reminder/CI note to run docs-health after v0.4.0.
48. Review whether AGENTS.md is getting too long (it grew ~40 lines today;
    candidates to move: storage-facts quirks → docs/storage-schema.md).
49. Teach verify-all to refuse running alongside another verify-all
    (lockfile) — cheap contention insurance.
50. Archive this report's "fucked up" list into the retrospective habit:
    every status report keeps a section d) (this format works).

## g) QUESTIONS (cannot figure out myself)

1. **Push master now?** (T45) — full gate green locally on every commit
   since the fix; origin red; none of today's CI improvements run until
   pushed. You said never without instruction: instruct?
2. **T15 — decide the API**: grow `Schema.MissingColumns()` into
   `MissingCapabilities()` covering tables (small breaking change) before
   v0.4.0, keep `MissingColumns()` + add non-breaking `MissingTables()`,
   or defer past v0.4.0? (The concurrent session flagged the same item.)
3. **May I reply to ArcticFox2029's comment on crush discussion #3740**
   (endorse the schema-version-field idea, attach census evidence), or are
   upstream voices yours only for now?

— Session closed at 17:45 CEST; waiting for instructions.
