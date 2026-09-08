# TODO List

Short- and mid-term improvement tasks, ranked by impact. Completed items are
deleted from this file — their record lives in [CHANGELOG.md](CHANGELOG.md).
Long-term ideas live in [ROADMAP.md](ROADMAP.md).

Items carry stable IDs (`T1`, `T2`, …). Cite them as `TODO_LIST T3` in status
reports and annotations. New items take the next free number; IDs are never
renumbered, and deleting an item retires its ID for good.

## High

- [ ] **T10** Part-discriminator census tripwire: census the `{type,data}`
      envelope's discriminator kinds across real-registry messages when
      `CRUSH_DATA_REAL_DATA_DIR` is set (the todos-shape census pattern) —
      one new upstream part type currently lands in `UnknownPart`
      silently, and `TestAllAPIOnRealDatabase` decodes but does not count
      kinds. 1–2h — `parts_test.go`, `realdata_test.go`; source:
      docs/status/2026-09-07_21-59_upstream-v0.92.0-storage-verification.md (f)8/(f)17

## Medium

- [ ] **T11** Fixture test pinning `CRUSH_GLOBAL_DATA`-is-a-directory
      semantics (upstream joins crush.json then takes dir). 20m —
      `discover_test.go`; source: same report (f)12
- [ ] **T12** Fixture test: `crush projects --json` with an empty registry →
      empty result, no error (runs the real CLI fallback path). 20m —
      `discover_test.go`; source: same report (f)13/(b)4
- [ ] **T13** Storage-schema snapshot doc per verified release
      docs/storage-schema-v0.92.0.md: tables, columns, migrations as
      verified against the pinned source. 30m — `docs/`; source: same
      report (f)14
- [ ] **T14** Document the parts envelope `{type,data}` + all 8 upstream
      discriminators in `doc.go` (currently only in AGENTS.md storage
      facts). 20m — `doc.go`; source: same report (f)15
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

## Low

- [ ] **T6** Mine nightly fuzz artifacts for corpus seeds — nightly runs
      exist and are green since 2026-08-17 (observed 2026-09-08: latest
      five runs success). ongoing — `.github/workflows/fuzz.yml`
- [ ] **T7** Pin GitHub action versions via Renovate once the app is
      installed (depends on T1). 10m — `.github/workflows/*.yml`
- [ ] **T9** Report the `time.UnixMilli` date bug to openusage and
      mnemo (sessions land in 1970; both cite Crush's lying migration
      comment as their schema reference). Needs user go-ahead. 10m each
      — upstream of this repo
- [ ] **T18** Verification-session fuzz cadence: 30s
      `go test -fuzz=DecodeParts` + `go test -fuzz=DecodeTodos` each time
      upstream verification runs (targets + CI workflow exist; never run
      locally). 5m — fuzz targets; source: same report (f)18
- [ ] **T19** Cite upstream `projects.Register()` sort (LastAccessed desc)
      in the dedupe doc comment — justifies "most recent wins". 10m —
      `discover.go`; source: same report (f)22
- [ ] **T20** Outbound upstream issue (run verify-before-filing first):
      sessions/messages migration comments claim milliseconds while
      `update_sessions_updated_at` writes `strftime('%s','now')` (seconds)
      and the CLI renders `time.Unix(CreatedAt, 0)`. 30m — upstream; source:
      same report (f)24
- [ ] **T21** Investigate the `crush.db?_loc=auto` stray file in
      `/home/lars/projects/.crush` (present 2026-09-07; likely a tool
      mishandling a SQLite URI; harmless to this library — it opens exact
      paths). Trash once understood. 20m — local data dir; source: same
      report (f)25
- [ ] **T22** `.golangci.yml` refresh: `exhaustruct` deprecated since
      v2.13 → `exhaustruct_v5` (deprecation warning on every lint run).
      15m — `.golangci.yml`; source: same report (f)26
- [ ] **T23** Consider `nix flake check --all-systems` in CI (gate currently
      omits aarch64/darwin). 1h — `.github/workflows/ci.yml`; source: same
      report (f)28
- [ ] **T24** Benchmarks: add `BenchmarkIterMessages` and regenerate
      `docs/benchmarks/baseline-benchmarks.txt` — regeneration is due
      regardless (the todos probe, `Message.UpdatedAt` scan, and ReadFiles
      ORDER BY all touch read paths), and the iter path has never been in
      the trend. 1h — `go test -bench . -count=6 | tee …` + benchstat;
      source: same report (f)29 + 2026-08-16_08-50 report (f)4/(f)47
- [ ] **T25** Pin registry `last_accessed` UTC round-trip with a test
      (upstream writes `time.Now().UTC()`). 15m — `discover_test.go`;
      source: same report (f)30
- [ ] **T27** Test: registry parse tolerates unknown top-level keys (future
      upstream fields must not break Discovery). 20m — `discover_test.go`;
      source: same report (f)37
- [ ] **T28** Decide whether summary messages (`is_summary_message = 1`)
      should count in day filters/stats, and pin the answer. 30m —
      `sessions.go`, `stats.go`; source: same report (f)38
- [ ] **T29** Add shellcheck to the devShell and run it over `scripts/`
      (three bash guard scripts, today only battle-tested by running
      them). 20m — `flake.nix`, `scripts/`; source:
      2026-08-16_04-20 report (e)4/(f)11

## External (waiting on GitHub UI, schedules, or upstream)

- [ ] **T1** Install/enable the Renovate app (config validates; inert until
      the GitHub App is installed). 5m — `renovate.json`
- [ ] **T3** After the `.github/workflows/flake-update.yml` permissions fix (`contents: write`,
      2026-09-08 — the 2026-09-01 scheduled run failed with
      "Permission to LarsArtmann/go-crush-data.git denied to the
      github-actions bot") reaches origin: observe the first successful
      flake-lock PR and that the vendorHash guard behaves. 5m —
      `.github/workflows/flake-update.yml`

## Parked (plan-level, tracked in the ecosystem plan — not this repo)

- Upstream PR to cosmtrek/mindwalk for the `sdk/go-crush-data` branch
  (Stream X T24; needs user go-ahead to push).
- Post the charmbracelet/crush read-access Discussion from
  docs/upstream-read-access-discussion-draft.md (Stream X T20; needs
  user go-ahead; link it here once posted).
