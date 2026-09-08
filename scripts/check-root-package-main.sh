#!/usr/bin/env bash
#
# Fail when a Go file declares `package main` outside scripts/ or cmd/.
#
# Why: this repo is a single-package Go library (`package crushdata` at the
# root). A stray root-level package-main file clashes with the library
# package and breaks `go build ./...` for every consumer. It happened once:
# an auto-committed probe file sat unnoticed while CI and the benchmark
# trend ran red for hours (2026-09-08, 33d454d). This guard names the
# failure at the source instead of letting it surface as a confusing build
# error three CI legs later.
#
# Usage:
#   scripts/check-root-package-main.sh
#
# Allowed locations for package main: scripts/** (tooling such as
# censusprobe and genschema) and cmd/** (none today, kept for the future).
# vendor/ and generated files are ignored.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

violations=$(find . -name '*.go' \
	-not -path './scripts/*' \
	-not -path './cmd/*' \
	-not -path './vendor/*' \
	-not -name '*.gen.go' \
	-not -name '*_templ.go' \
	-exec grep -l '^package main' {} + 2>/dev/null || true)

if [[ -n $violations ]]; then
	echo "check-root-package-main: package main outside scripts/ and cmd/:" >&2
	while IFS= read -r file; do
		echo "  $file" >&2
	done <<<"$violations"
	echo >&2
	echo "The repo root is the single-package library 'crushdata'; a package-main" >&2
	echo "file there breaks go build ./... for every consumer. Move the file under" >&2
	echo "scripts/ (see scripts/censusprobe) or delete it." >&2
	exit 1
fi

echo "check-root-package-main: OK"
