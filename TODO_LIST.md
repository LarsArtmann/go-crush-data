# TODO List

Short- and mid-term improvement tasks, ranked by impact. Completed items are
deleted from this file — their record lives in [CHANGELOG.md](CHANGELOG.md).
Long-term ideas live in [ROADMAP.md](ROADMAP.md).

## Pending user decisions

- [ ] **Cut v0.2.1** to put the Windows test-suite fixes on a tagged version.
  The v0.2.0 tag is immutable and its Windows CI leg is red (test code only —
  `quoteJSON` escaping and `/bin/sh`-based fake CLIs; library code is
  correct). Master is green since `c7482e2`/`c3a083b`. When cut, fold an
  errata note about v0.2.0's Windows tests into the `[0.2.1]` section.
  Requires tag push approval. 30m — `docs/status/2026-08-16_00-40_*.md` (g/1)

## External (waiting on GitHub UI, schedules, or upstream)

- [ ] Install/enable the Renovate app (config validates; inert until the
  GitHub App is installed). 5m — `renovate.json`
- [ ] Require green CI status checks on master (branch is currently
  unprotected — verified via the branches API). 5m — GitHub settings
- [ ] Observe the first nightly fuzz run (03:17 UTC); on green, flip the
  FEATURES row to FULLY_FUNCTIONAL. 5m — `.github/workflows/fuzz.yml`
- [ ] Observe the first monthly flake-lock PR; check the vendorHash guard
  fires correctly on a stale hash. 5m — `.github/workflows/flake-update.yml`
- [ ] Verify pkg.go.dev crawled v0.2.0 (page still rendered v0.1.1 docs at
  last check); spot-check `OpenContext`/`Todos`. 5m — pkg.go.dev
- [ ] gosec G701 taint false positive: minimal upstream repro + issue
  (verify-before-filing workflow). The `//nolint:gosec` with rationale is
  sufficient meanwhile. 30m — `sessions.go`

## High — Windows & parser correctness

- [ ] Add a `TestParseProjectsOutput` case: `}` in noise AFTER the JSON
  payload (extraction grabs too much → decode error path). The comment in
  `discover.go` now documents this limitation; pin it. 10m —
  `discover_test.go` (table at :380)
- [ ] Add `TestQuoteJSON` — pin backslash escaping as the regression guard
  for the Windows example fix. 10m — `example_test.go:84`
- [ ] Cross-platform fakeCLI: compile a tiny Go helper (or use
  `os.Executable`) instead of `/bin/sh` scripts so CLI fallback tests RUN on
  Windows instead of skipping. 45m — `discover_test.go:202`, `:341`
- [ ] Audit remaining tests for platform assumptions (`/bin/sh`, chmod,
  unescaped Windows paths). 20m — `*_test.go`

## Medium — CI depth

- [ ] Add `-count=2` to the CI test command (documented for local use only
  today). 2m — `.github/workflows/ci.yml:48`
- [ ] Add a `go mod verify` CI step (module cache vs go.sum). 5m —
  `.github/workflows/ci.yml`
- [ ] Trigger the release workflow's `dry_run` input once to exercise the
  notes-extraction path (support shipped, never triggered). 5m —
  `.github/workflows/release.yml:11`

## Low

- [ ] Reuse `fakeCLI` in `TestDiscoverProjectsCLIExitNonzeroWithPartialJSON`
  instead of hand-rolling a second script (DRY; inherits the Windows skip).
  10m — `discover_test.go:341`
- [ ] Mine nightly fuzz artifacts for corpus seeds once runs exist. ongoing —
  `.github/workflows/fuzz.yml`
- [ ] Pin GitHub action versions via Renovate once the app is installed
  (depends on the external item above). 10m — `.github/workflows/*.yml`
