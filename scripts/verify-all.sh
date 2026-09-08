#!/usr/bin/env bash
#
# Single entry point for full verification: the canonical gate plus the
# upstream drift guard (self-test and live clone) plus the env-gated
# real-data sweeps. Every step runs, the summary table reports each one,
# and the exit code is nonzero when any step failed — so a session cannot
# stage verification piecemeal and call the tree verified.
#
# Usage:
#   scripts/verify-all.sh
#
# Environment (all optional; unset steps report SKIPPED where noted):
#   CRUSH_UPSTREAM_DIR        reuse a pinned clone for the drift check
#                             (must be at the pinned CRUSH_SHA)
#   CRUSH_DATA_REAL_DATA_DIR  real-database sweeps (tests auto-detect ../.crush)
#   CRUSH_DATA_REAL_REGISTRY  whole-registry census (heavy, ~30m; opt-in)

set -uo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root" || exit 1

declare -a step_names=()
declare -a step_status=()

record() {
	step_names+=("$1")
	step_status+=("$2")
}

# run_step <name> <command...>: run the command, never abort the sweep.
run_step() {
	local name=$1
	shift

	echo
	echo "==> $name"
	if "$@"; then
		record "$name" OK
	else
		record "$name" FAIL
	fi
}

echo "verify-all: environment"
echo "  CRUSH_UPSTREAM_DIR=${CRUSH_UPSTREAM_DIR:-<unset, drift check clones fresh>}"
echo "  CRUSH_DATA_REAL_DATA_DIR=${CRUSH_DATA_REAL_DATA_DIR:-<unset, ../.crush auto-detected if present>}"
echo "  CRUSH_DATA_REAL_REGISTRY=${CRUSH_DATA_REAL_REGISTRY:-<unset, registry census skipped>}"

run_step "go build" go build ./...
run_step "go vet" go vet ./...
run_step "go test -race -shuffle" go test -race -shuffle=on ./...
run_step "golangci-lint (CI binary)" go tool golangci-lint run --timeout=5m ./...
run_step "nix flake check" nix flake check
run_step "actionlint" nix develop --command actionlint
run_step "doc links" scripts/check-doc-links.sh
run_step "root package-main guard" scripts/check-root-package-main.sh
run_step "vendor-hash guard" scripts/check-vendor-hash.sh HEAD~1
run_step "upstream drift self-test" scripts/check-upstream-drift.sh --self-test
run_step "upstream drift (pinned clone)" scripts/check-upstream-drift.sh
run_step "real-data sweeps" go test -run 'TestAllAPIOnRealDatabase|TestPartDiscriminatorsCensusShape' .

if [[ -n ${CRUSH_DATA_REAL_REGISTRY:-} ]]; then
	run_step "registry census (heavy)" \
		go test -run TestPartDiscriminatorsCensusRegistry -timeout 30m .
else
	record "registry census (heavy)" SKIPPED
	echo "==> registry census (heavy): SKIPPED (set CRUSH_DATA_REAL_REGISTRY to run it)"
fi

echo
echo "verify-all: summary"
failed=0
for i in "${!step_names[@]}"; do
	printf '  %-36s %s\n' "${step_names[$i]}" "${step_status[$i]}"
	[[ ${step_status[$i]} == FAIL ]] && failed=1
done

if ((failed)); then
	echo "verify-all: FAILED — see the steps above"
	exit 1
fi

echo "verify-all: all steps OK (skips noted above are opt-in)"
