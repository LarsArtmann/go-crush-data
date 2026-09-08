# TODO List

Short- and mid-term improvement tasks, ranked by impact. Completed items are
deleted from this file — their record lives in [CHANGELOG.md](CHANGELOG.md).
Long-term ideas live in [ROADMAP.md](ROADMAP.md).

Items carry stable IDs (`T1`, `T2`, …). Cite them as `TODO_LIST T3` in status
reports and annotations. New items take the next free number; IDs are never
renumbered, and deleting an item retires its ID for good.

## External (waiting on GitHub UI, schedules, or upstream)

- [ ] **T1** Install/enable the Renovate app (config validates; inert until
      the GitHub App is installed). 5m — `renovate.json`
- [ ] **T2** Observe the first nightly fuzz run (03:17 UTC); on green, flip
      the FEATURES row to FULLY_FUNCTIONAL. 5m — `.github/workflows/fuzz.yml`
- [ ] **T3** Observe the first monthly flake-lock PR; check the vendorHash
      guard fires correctly on a stale hash. 5m —
      `.github/workflows/flake-update.yml`
- [ ] **T4** Verify pkg.go.dev renders v0.3.0 (`DecodeTodos`,
      `DB.IterMessages`); confirm the recipe page lists. 5m — pkg.go.dev

## Low

- [ ] **T6** Mine nightly fuzz artifacts for corpus seeds once runs exist.
      ongoing — `.github/workflows/fuzz.yml`
- [ ] **T7** Pin GitHub action versions via Renovate once the app is
      installed (depends on T1). 10m — `.github/workflows/*.yml`
- [ ] **T8** Harden the read-only DSN with `busy_timeout` and
      `query_only` (2 of 6 ecosystem readers set busy_timeout; ours can
      fail fast with SQLITE_BUSY under a live writer). See
      docs/ecosystem-implementation-review.md. 30m — `db.go`
- [ ] **T9** Report the `time.UnixMilli` date bug to openusage and
      mnemo (sessions land in 1970; both cite Crush's lying migration
      comment as their schema reference). Needs user go-ahead. 10m each
      — upstream of this repo

## Parked (plan-level, tracked in the ecosystem plan — not this repo)

- Upstream PR to cosmtrek/mindwalk for the `sdk/go-crush-data` branch
  (Stream X T24; needs user go-ahead to push).
- Post the charmbracelet/crush read-access Discussion from
  docs/upstream-read-access-discussion-draft.md (Stream X T20; needs
  user go-ahead; link it here once posted).
