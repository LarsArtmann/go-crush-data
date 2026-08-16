# TODO List

Short- and mid-term improvement tasks, ranked by impact. Completed items are
deleted from this file — their record lives in [CHANGELOG.md](CHANGELOG.md).
Long-term ideas live in [ROADMAP.md](ROADMAP.md).

## External (waiting on GitHub UI, schedules, or upstream)

- [ ] Install/enable the Renovate app (config validates; inert until the
  GitHub App is installed). 5m — `renovate.json`
- [ ] Observe the first nightly fuzz run (03:17 UTC); on green, flip the
  FEATURES row to FULLY_FUNCTIONAL. 5m — `.github/workflows/fuzz.yml`
- [ ] Observe the first monthly flake-lock PR; check the vendorHash guard
  fires correctly on a stale hash. 5m — `.github/workflows/flake-update.yml`
- [ ] Verify pkg.go.dev renders v0.2.1 (proxy.golang.org already serves it;
  the page still rendered v0.1.1 at last check); spot-check
  `OpenContext`/`Todos`. 5m — pkg.go.dev
- [ ] gosec G701 taint false positive: minimal upstream repro + issue
  (verify-before-filing workflow). The `//nolint:gosec` with rationale is
  sufficient meanwhile. 30m — `sessions.go`

## High — Windows & parser correctness

- [ ] Cross-platform fakeCLI: compile a tiny Go helper (or use
  `os.Executable`) instead of `/bin/sh` scripts so CLI fallback tests RUN on
  Windows instead of skipping. 45m — `discover_test.go:202`, `:341`
- [ ] Audit remaining tests for platform assumptions (`/bin/sh`, chmod,
  unescaped Windows paths). 20m — `*_test.go`

## Medium — CI depth

- [ ] Add `scripts/check-doc-links.sh` (markdown links resolve + `file:line`
  citations point at real files) and wire it into the canonical gate. 30m —
  2026-08-16 audit (the drift class this audit fixed by hand)

## Low

- [ ] Assign stable IDs to TODO_LIST items so annotations can cite
  "TODO_LIST item N" instead of the file generically. 10m
- [ ] Rename `docs/benchmarks/baseline-benchmark-sessions.txt` — it holds
  all three benchmarks now (Sessions, Messages, AgentGraph). 5m —
  `.github/workflows/bench.yml` references it
- [ ] Mine nightly fuzz artifacts for corpus seeds once runs exist. ongoing —
  `.github/workflows/fuzz.yml`
- [ ] Pin GitHub action versions via Renovate once the app is installed
  (depends on the external item above). 10m — `.github/workflows/*.yml`
