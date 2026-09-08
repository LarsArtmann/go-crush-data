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

## Low

- [ ] **T6** Mine nightly fuzz artifacts for corpus seeds — nightly runs
      exist and are green since 2026-08-17 (observed 2026-09-08: latest
      five runs success; first local fuzz sessions run 2026-09-08,
      2×3.8M execs clean). ongoing — `.github/workflows/fuzz.yml`
- [ ] **T7** Pin GitHub action versions via Renovate once the app is
      installed (depends on T1). 10m — `.github/workflows/*.yml`

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
      campaign — daily pass is now one command:
      `scripts/check-upstream-status.sh` (thread states + G1/G2 verdict;
      2026-09-08: crush#3576 MERGED, #3740 has 1 third-party comment, fix
      PRs still open). Run Phase G (adoption suggestions, drafts in the
      campaign plan) once both fix PRs merge — `docs/upstream-filings.md`
      is the ledger.

## Parked (plan-level, tracked in the ecosystem plan — not this repo)

- Cut **v0.4.0** once `[Unreleased]` settles (census tripwire close-out
  is the last code item); tags need explicit approval and a green origin
  first (T45).
- Upstream PR to cosmtrek/mindwalk for the `sdk/go-crush-data` branch
  (Stream X T24; needs user go-ahead to push).
- charmbracelet/crush read-access Discussion: POSTED 2026-09-08 as
  [discussion #3740](https://github.com/charmbracelet/crush/discussions/3740)
  (Ideas; body verified). Follow-ups tracked as TODO_LIST T35.
