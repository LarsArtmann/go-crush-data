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
- [ ] **T4** Verify pkg.go.dev renders v0.2.1 (proxy.golang.org already
  serves it; the page still rendered v0.1.1 at last check); spot-check
  `OpenContext`/`Todos`. 5m — pkg.go.dev
- [ ] **T5** gosec G701 taint false positive: minimal upstream repro + issue
  (verify-before-filing workflow). The config-level exclusion in
  `.golangci.yml` (with rationale) is sufficient meanwhile. 30m —
  `sessions.go:56`

## Low

- [ ] **T6** Mine nightly fuzz artifacts for corpus seeds once runs exist.
  ongoing — `.github/workflows/fuzz.yml`
- [ ] **T7** Pin GitHub action versions via Renovate once the app is
  installed (depends on T1). 10m — `.github/workflows/*.yml`
