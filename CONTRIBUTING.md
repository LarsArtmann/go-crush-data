# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## The parity contract (read before touching SQL)

`TestStatsParityWithCrushDailySQL` re-runs the historical crush-daily
collector SQL against the same fixtures and requires **identical numbers**.
The stats SQL in `stats.go` is ported verbatim on purpose: downstream
consumers compare their numbers across tools. Do not "improve" that SQL —
not the join, not the CTE, not the `strftime` — without a coordinated
migration plan for every consumer. If a change is truly required, the parity
test must change **in the same commit**, with the reason in the CHANGELOG.

## Development

```bash
nix develop       # dev shell: Go, golangci-lint, govulncheck, actionlint
nix run .#lint    # golangci-lint (~90 linters, see .golangci.yml)
nix run .#test    # race test via nix
go test ./...     # full suite (fixture databases are generated per-test)
nix flake check   # build + format checks
```

No Nix? `go build ./... && go test -race ./... && golangci-lint run ./...`
is the whole gate. CI additionally runs `go test -shuffle=on` and
`nix flake check` (see `.github/workflows/`).

### go.sum ↔ vendorHash coupling

The Nix package vendors dependencies and pins their hash in `flake.nix`
(`vendorHash`). Whenever `go.mod`/`go.sum` change, that hash must be
re-derived: run `nix build .#default`, copy the `got:` sha256 from the error
into `flake.nix`, build again. `scripts/check-vendor-hash.sh` fails fast in
CI when the two drift apart. Also note the Nix source filter ignores
untracked files — a new `.go` file that was never `git add`ed produces
misleading "undefined:" build errors inside `nix flake check`.

### Benchmarks

`docs/benchmarks/baseline-benchmarks.txt` is the committed baseline;
the `Benchmark trend` workflow compares every push against it and posts the
diff to the job summary. To regenerate the baseline:

```bash
go test -run '^$' -bench BenchmarkSessionsList -count 6 . \
  > docs/benchmarks/baseline-benchmarks.txt
```

To compare locally: `go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt`.
(benchstat is not in nixpkgs; the `go run` fallback is the documented path.)

## Conventions

- Tests build fixture databases per-test (`testutil_test.go`); never commit
  binary testdata.
- Every behavior the library promises has a pinning test; if you add a
  promise, add the test in the same commit.
- Error paths distinguish "old database" (`ErrUnsupportedSchema`) from
  "broken database" (probe errors) — keep them separate.
- Release procedure lives in [RELEASING.md](RELEASING.md).

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
