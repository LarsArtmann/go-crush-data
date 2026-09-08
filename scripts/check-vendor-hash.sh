#!/usr/bin/env bash
#
# Detect go.sum/go.mod ↔ flake.nix vendorHash drift before nix does.
#
# Why: buildGoModule's vendorHash must be re-derived whenever the Go module
# set changes. When it rots, "nix build" fails with a cryptic hash-mismatch
# error (this bit us at commit 16260fe). This guard gives a fast, actionable
# message instead — and catches the drift even when no nix job runs.
#
# Usage:
#   scripts/check-vendor-hash.sh [BASE_REV]
#
#   No argument  → compare the working tree against HEAD (local pre-commit
#                  use). Drift FAILS: you are mid-edit and can fix it now.
#   BASE_REV     → compare BASE_REV..working tree, or BASE_REV..HEAD when the
#                  working tree is clean (CI use, e.g. "HEAD~1"). Drift is
#                  ADVISORY only (a ::warning annotation): the range
#                  heuristic cannot distinguish real drift from legal
#                  go.mod edits that leave the vendor set unchanged (go
#                  directive bumps, tidy pruning) or from related changes
#                  split across commits by the auto-commit daemon. Only
#                  `nix flake check` / `nix build` verifies the hash; the
#                  nix CI job enforces it on every push.
#
# Rules:
#   local FAIL : go.mod or go.sum changed but the vendorHash in flake.nix
#                did not.
#   CI   WARN  : same condition in BASE_REV mode (advisory, see above).
#   both WARN  : vendorHash changed with no go.mod/go.sum change (possible
#                nixpkgs rehash via flake.lock bump — legal, so only a
#                warning).
#   PASS       : everything else, including no changes at all.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if [[ $# -gt 0 ]]; then
	base_rev=$1
	ci_mode=1
else
	base_rev=HEAD
	ci_mode=0
fi

if ! git rev-parse --verify --quiet "$base_rev" >/dev/null; then
	echo "check-vendor-hash: base revision '$base_rev' not found" >&2
	echo "  hint: on CI, checkout with fetch-depth: 2 so HEAD~1 exists" >&2
	exit 2
fi

go_files_changed=0
if ! git diff --quiet "$base_rev" -- go.mod go.sum; then
	go_files_changed=1
fi

base_vendor_hash=$(git show "$base_rev:flake.nix" 2>/dev/null | grep -o 'vendorHash = "[^"]*"' || true)
head_vendor_hash=$(grep -o 'vendorHash = "[^"]*"' flake.nix || true)

if [[ $go_files_changed -eq 1 && "$base_vendor_hash" == "$head_vendor_hash" ]]; then
	if [[ $ci_mode -eq 1 ]]; then
		echo "::warning::check-vendor-hash: go.mod/go.sum changed since $base_rev without a vendorHash change — advisory only (the range heuristic false-positives on daemon-split commits and vendor-set-neutral go.sum edits); the nix CI job verifies the real hash. If that job fails: nix build .#default and copy the 'got:' sha256 into flake.nix's vendorHash."
		exit 0
	fi
	cat >&2 <<EOF
check-vendor-hash: DRIFT DETECTED

go.mod/go.sum changed since $base_rev but the vendorHash in flake.nix
did not ($head_vendor_hash).

Fix:
  nix build .#default       # fails with the correct hash in the error
  # copy the "got:" sha256 into flake.nix's vendorHash
  nix build .#default       # now passes
EOF
	exit 1
fi

if [[ $go_files_changed -eq 0 && "$base_vendor_hash" != "$head_vendor_hash" ]]; then
	echo "check-vendor-hash: note — vendorHash changed without go.mod/go.sum changes"
	echo "  (expected after a nixpkgs/flake.lock bump rehashes the vendor derivation)"
fi

echo "check-vendor-hash: OK (base=$base_rev)"
