#!/usr/bin/env bash
#
# Detect drift between the pinned charmbracelet/crush migration set and the
# capability-coverage list embedded in schema_drift_test.go.
#
# Why: the capability probes are the library's only defense against columns
# added by upstream migrations, and that defense was once held in an agent's
# memory — which is exactly how the sessions.todos gap escaped (a DB frozen
# before 2025-08-12 made Sessions/AgentGraph fail with "no such column").
# This guard makes the migration→probe mapping mechanical: run it on every
# new upstream stable release (see AGENTS.md verification cadence).
#
# What it compares:
#   upstream side : every ALTER TABLE … ADD COLUMN and CREATE TABLE in
#                   internal/db/migrations except the initial migration,
#                   extracted from the pinned clone.
#   our side      : the entries of the migrations table in
#                   schema_drift_test.go (the list the guard test enforces).
#
# Usage:
#   scripts/check-upstream-drift.sh              # live check (clones unless
#                                                # CRUSH_UPSTREAM_DIR is set)
#   scripts/check-upstream-drift.sh --self-test  # prove the negative path
#                                                # fires (doctored fixture)
#
# Environment:
#   CRUSH_UPSTREAM_DIR  reuse an existing clone at the pinned ref instead of
#                       cloning (must be at CRUSH_SHA).
#
# On drift: exit 1 with a unified diff. Update schema_drift_test.go (add the
# entry, probe it, extend the guard), bump CRUSH_REF/CRUSH_SHA below, and
# re-verify against the new release.
#
# The self-test exists because an ad-hoc /tmp doctored fixture once produced
# an empty extraction and a vacuous "verified" pass. It runs the guard
# against a checked-in doctored fixture (an upstream column the guard list
# does not know) and requires exit 1 with the drift visible, plus an empty
# extraction probe that must fail loudly (exit 2) rather than pass.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

# Pinned upstream verification point. Bump together, then re-run this
# script and the guard test.
crush_ref=v0.92.0
crush_sha=559ec80922fecf3baa0b7599230f4c91067440de

initial_migration=20250424200609_initial.sql
guard_test=schema_drift_test.go
selftest_fixture=scripts/fixtures/upstream-drift-selftest/migrations

# extract_additions <migrations-dir>: every column and table addition from
# every non-initial migration. One sed expression per line: the first prints
# table.column for ADD COLUMN lines; when it does not match, the second
# prints table: for CREATE TABLE lines. Indexes and triggers never match.
# find -exec (never xargs) so an empty dir yields empty output instead of a
# cat reading stdin forever.
extract_additions() {
	find "$1" -name '*.sql' ! -name "$initial_migration" -exec cat {} + |
		sed -nE \
			-e 's/.*ALTER TABLE[[:space:]]+([A-Za-z_]+)[[:space:]]+ADD COLUMN[[:space:]]+([A-Za-z_]+).*/\1.\2/p' \
			-e 's/.*CREATE TABLE[[:space:]]+(IF NOT EXISTS[[:space:]]+)?([A-Za-z_]+).*/table:\2/p' |
		sort -u
}

# guard_entries: our side, walked out of the migrations table in
# schema_drift_test.go. Each entry contributes table.column, or a bare
# table when the entry has no column (whole-table migrations like read_files).
guard_entries() {
	awk -v guard="$guard_test" '
		/migrations := \[\]migration\{/ { inblock = 1; next }
		inblock && /^\t\}$/            { inblock = 0; table = "" }
		!inblock                       { next }
		/table:/ {
			i = index($0, "\"")
			j = index(substr($0, i + 1), "\"")
			table = substr($0, i + 1, j - 1)
			hascol = 0
		}
		/column:/ {
			i = index($0, "\"")
			j = index(substr($0, i + 1), "\"")
			print table "." substr($0, i + 1, j - 1)
			hascol = 1
		}
		/\},/ && table != "" && !hascol { print "table:" table; table = "" }
		END { if (inblock) { printf "unterminated migrations block in %s\n", guard > "/dev/stderr"; exit 2 } }
	' "$guard_test" | sort -u
}

# run_guard <migrations-dir> <label>: compare both sides. Exit 0 on match,
# 1 on drift (diff on stderr), 2 when either extraction comes back empty —
# an empty extraction must never produce a vacuous pass.
run_guard() {
	local dir=$1 label=$2
	local upstream ours diff

	upstream=$(extract_additions "$dir")
	if [[ -z $upstream ]]; then
		echo "check-upstream-drift: extracted no migrations from $dir — extraction broken?" >&2
		return 2
	fi

	ours=$(guard_entries)
	if [[ -z $ours ]]; then
		echo "check-upstream-drift: extracted no entries from $guard_test — parsing broken?" >&2
		return 2
	fi

	if ! diff=$(diff -u <(printf '%s\n' "$ours") <(printf '%s\n' "$upstream")); then
		cat >&2 <<EOF
check-upstream-drift: DRIFT DETECTED ($label) against the guard list in $guard_test

$diff

The upstream migration set no longer matches the capability guard. Fix:
  1. add every new table.column above to the migrations table in
     $guard_test — probed (plus a Schema capability and NULL
     substitution) when any query reads it, listed as unread otherwise
  2. run the guard: go test -run TestUpstreamMigrationColumnsAreProbedOrExempt .
  3. bump crush_ref/crush_sha at the top of this script to the release
     you verified, and record the re-verification in AGENTS.md
EOF
		return 1
	fi

	echo "check-upstream-drift: OK ($label, $(printf '%s\n' "$upstream" | wc -l) non-initial migrations)"
}

if [[ ${1:-} == "--self-test" ]]; then
	if [[ ! -d $selftest_fixture ]]; then
		echo "check-upstream-drift self-test: fixture dir missing: $selftest_fixture" >&2
		exit 2
	fi

	set +e
	output=$(run_guard "$selftest_fixture" "self-test fixture" 2>&1)
	status=$?
	set -e

	if [[ $status -ne 1 ]]; then
		echo "check-upstream-drift self-test: FAILED — doctored fixture must exit 1 (drift), got $status" >&2
		printf '%s\n' "$output" >&2
		exit 1
	fi

	if ! grep -q "DRIFT DETECTED" <<<"$output"; then
		echo "check-upstream-drift self-test: FAILED — drift verdict line missing from output" >&2
		printf '%s\n' "$output" >&2
		exit 1
	fi

	if ! grep -q "selftest_rogue" <<<"$output"; then
		echo "check-upstream-drift self-test: FAILED — rogue column not visible in the drift diff" >&2
		printf '%s\n' "$output" >&2
		exit 1
	fi

	empty_dir=$(mktemp -d)
	set +e
	output=$(run_guard "$empty_dir" "empty extraction probe" 2>&1)
	status=$?
	set -e
	rm -rf "$empty_dir"

	if [[ $status -ne 2 ]]; then
		echo "check-upstream-drift self-test: FAILED — empty extraction must exit 2, got $status" >&2
		printf '%s\n' "$output" >&2
		exit 1
	fi

	echo "check-upstream-drift self-test: OK (doctored fixture fires drift; empty extraction fails loudly)"
	exit 0
fi

upstream_dir=${CRUSH_UPSTREAM_DIR:-}

if [[ -z $upstream_dir ]]; then
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	echo "check-upstream-drift: cloning charmbracelet/crush at $crush_ref …"
	git clone --quiet --depth 1 --branch "$crush_ref" \
		https://github.com/charmbracelet/crush "$tmp/crush"

	upstream_dir="$tmp/crush"
fi

head_sha=$(git -C "$upstream_dir" rev-parse HEAD)
if [[ $head_sha != "$crush_sha" ]]; then
	echo "check-upstream-drift: $upstream_dir is at $head_sha, want pinned $crush_sha" >&2
	echo "  hint: point CRUSH_UPSTREAM_DIR at a clone of $crush_ref, or unset it to clone fresh" >&2
	exit 2
fi

migrations_dir=$upstream_dir/internal/db/migrations
if [[ ! -d $migrations_dir ]]; then
	echo "check-upstream-drift: migrations dir not found: $migrations_dir" >&2
	exit 2
fi

run_guard "$migrations_dir" "crush $crush_ref @ ${crush_sha:0:7}"
