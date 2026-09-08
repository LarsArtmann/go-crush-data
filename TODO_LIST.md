# TODO List

Short- and mid-term improvement tasks, ranked by impact. Completed items are
deleted from this file — their record lives in [CHANGELOG.md](CHANGELOG.md).
Long-term ideas live in [ROADMAP.md](ROADMAP.md).

Items carry stable IDs (`T1`, `T2`, …). Cite them as `TODO_LIST T3` in status
reports and annotations. New items take the next free number; IDs are never
renumbered, and deleting an item retires its ID for good.

## Medium

- [ ] **T15** `Schema.MissingColumns()` omits the read_files _table_;
      consider `MissingCapabilities()` covering tables (small API break —
      needs a user call). 30m — `schema.go`; source: same report (f)16
- [ ] **T16** Real-data cross-check: run this library's `Stats` and
      crush-daily's collector over the same DB and require number parity
      (the consumer contract behind the verbatim-SQL rule). 1–2h —
      `stats_test.go`; source: same report (f)20
- [ ] **T17** GitHub Action: open an issue when a new crush stable release
      lands (drives the AGENTS.md verification cadence; the weekly
      `.github/workflows/upstream-drift.yml` job already covers scheduled detection).
      1h — `.github/workflows/`; source: same report (f)33
- [ ] **T38** Generate the test-fixture DDL from the pinned upstream
      migrations instead of the hand-maintained approximation (still
      missing the `files` table, indexes, triggers, CHECK constraints —
      drifted once already). Single source of truth kills the fixture-drift
      class the way the probe guard killed probe drift. 1–2h —
      `testutil_test.go` + `scripts/check-upstream-drift.sh`; source:
      docs/status/archived/2026-09-08_05-08_upstream-task-execution-and-harvest.md (f)6/(e)3
- [ ] **T41** One-command upstream-status helper under `scripts/` (e.g.
      `check-upstream-status`): prints the merge state of openusage#357,
      mnemo#22, crush #3740/#3576/#3580/#3581 and a "G1/G2 UNBLOCKED"
      verdict — turns T35's daily pass from a hand-rolled ritual into a
      single run. 30m — `scripts/`; source:
      docs/status/2026-09-08_15-33_filing-campaign-status-and-self-review.md (c)4/(f)17
- [ ] **T42** CI guard against root `package main` build-breakers: fail when
      a `package main` file exists outside `scripts/`/`cmd/` (the daemon
      committed one at 06:24 UTC and origin's CI + bench legs sat red for
      hours until the fix was pushed). 30m — `.github/workflows/ci.yml`
      or a check script; source: same report (e)5/(f)18

## Low

- [ ] **T6** Mine nightly fuzz artifacts for corpus seeds — nightly runs
      exist and are green since 2026-08-17 (observed 2026-09-08: latest
      five runs success). ongoing — `.github/workflows/fuzz.yml`
- [ ] **T7** Pin GitHub action versions via Renovate once the app is
      installed (depends on T1). 10m — `.github/workflows/*.yml`
- [ ] **T18** Verification-session fuzz cadence: 30s
      `go test -fuzz=DecodeParts` + `go test -fuzz=DecodeTodos` each time
      upstream verification runs (targets + CI workflow exist; never run
      locally). 5m — fuzz targets; source: same report (f)18
- [ ] **T28** Decide whether summary messages (`is_summary_message = 1`)
      should count in day filters/stats, and pin the answer. 30m —
      `sessions.go`, `stats.go`; source: same report (f)38
- [ ] **T36** flake.nix lint app drops extra arguments (`exec golangci-lint
      run ./...` is hardcoded): pass `"$@"` through so `nix run .#lint --
      <args>` stops silently ignoring them — the false-belief trap hit one
      session twice. Mitigation already landed (AGENTS gates on the
      go-tool lint); this closes the wrapper itself. 15m — `flake.nix`;
      source: docs/status/archived/2026-09-08_05-08_upstream-task-execution-and-harvest.md (f)4/(e)6
- [ ] **T37** Drift-script self-test: checked-in doctored fixture proving the
      negative path fires (the ad-hoc /tmp doctoring once produced an empty
      file and a vacuous "verified" pass). 30m —
      `scripts/check-upstream-drift.sh`; source: same report (f)5/(e)4
- [ ] **T39** AGENTS.md process-rule batch: (a) concurrent-session convention
      (pre-edit `git log --since`/status check, surgical edits, never
      revert foreign work); (b) external-filing checklist (no forward
      promises in PR/Issue bodies, edit-after-posting as a required step,
      re-verify every cross-claim in the FINAL body immediately before
      posting). 30m — `AGENTS.md`; source: same report (f)13 +
      2026-09-08_15-33 report (e)1/(e)2/(f)16/(f)23
- [ ] **T40** censusprobe polish: generalize the hardcoded `/home/lars/...`
      registry path to an env var and add a header comment marking
      `scripts/censusprobe/` as the canonical location (public-repo
      credibility; the campaign depends on it). 20m —
      `scripts/censusprobe/main.go`; source: 2026-09-08_15-33 report (f)20/(f)21
- [ ] **T43** Record the external-filing trace: a short docs/upstream-filings
      ledger (or a CHANGELOG policy note) so retired TODO items (T9/T20)
      keep a local record of what was filed where. 15m — `docs/` or
      `CHANGELOG.md`; source: 2026-09-08_15-33 report (b)4/(f)19
- [ ] **T46** Single `verify-all` entry point: one command running the
      canonical gate + `scripts/check-upstream-drift.sh` + the env-gated
      real-data sweeps, so a session cannot stage verification piecemeal.
      30m — `scripts/`; source:
      docs/status/archived/2026-09-08_05-08_upstream-task-execution-and-harvest.md (f)14/(e) N5

## External (waiting on GitHub UI, schedules, or upstream)

- [ ] **T45** **Push master to origin — origin is RED**: CI and Benchmark
      trend both fail on `33d454d` (a root-level `package main` probe file
      clashed with the library package, pushed 06:24 UTC 2026-09-08). The
      fix (`0842dd1`) and every commit after it sit locally, all green
      (full gate re-verified by the 2026-09-08 docs-health pass). Never
      pushed without instruction — user action. 5m — `git push`
- [ ] **T1** Install/enable the Renovate app (config validates; inert until
      the GitHub App is installed). 5m — `renovate.json`
- [ ] **T34** Observe the first SCHEDULED upstream-drift run
      (Mondays 03:47 UTC; first = 2026-09-14) — the workflow_dispatch
      run on 2026-09-08 was green (5s). 5m —
      `.github/workflows/upstream-drift.yml`
- [ ] **T35** Monitor upstream responses to the 2026-09-08 filing
      campaign — [crush discussion #3740](https://github.com/charmbracelet/crush/discussions/3740),
      fix PRs [openusage#357](https://github.com/janekbaraniewski/openusage/pull/357) +
      [mnemo#22](https://github.com/Pilan-AI/mnemo/pull/22), comment on
      [crush#3576](https://github.com/charmbracelet/crush/pull/3576) — and
      run Phase G (adoption suggestions) once both fix PRs merge. Daily
      pass — `docs/status/2026-09-08_filing-campaign-plan.md`

## Parked (plan-level, tracked in the ecosystem plan — not this repo)

- Cut **v0.4.0** once `[Unreleased]` settles (census tripwire close-out
  is the last code item); tags need explicit approval and a green origin
  first (T45).
- Upstream PR to cosmtrek/mindwalk for the `sdk/go-crush-data` branch
  (Stream X T24; needs user go-ahead to push).
- charmbracelet/crush read-access Discussion: POSTED 2026-09-08 as
  [discussion #3740](https://github.com/charmbracelet/crush/discussions/3740)
  (Ideas; body verified). Follow-ups tracked as TODO_LIST T35.
