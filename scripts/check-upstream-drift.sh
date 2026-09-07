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
#   scripts/check-upstream-drift.sh
#
# Environment:
#   CRUSH_UPSTREAM_DIR  reuse an existing clone at the pinned ref instead of
#                       cloning (must be at CRUSH_SHA).
#
# On drift: exit 1 with a unified diff. Update schema_drift_test.go (add the
# entry, probe it, extend the guard), bump CRUSH_REF/CRUSH_SHA below, and
# re-verify against the new release.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

# Pinned upstream verification point. Bump together, then re-run this
# script and the guard test.
crush_ref=v0.92.0
crush_sha=559ec80922fecf3baa0b7599230f4c91067440de

initial_migration=20250424200609_initial.sql
guard_test=schema_drift_test.go

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

# Upstream side: column and table additions from every non-initial
# migration. Two sed expressions per line: the first prints table.column for
# ADD COLUMN lines; when it does not match, the second prints table: for
# CREATE TABLE lines. Indexes and triggers never match.
upstream=$(find "$migrations_dir" -name '*.sql' ! -name "$initial_migration" -print0 |
	xargs -0 cat |
	sed -nE \
		-e 's/.*ALTER TABLE[[:space:]]+([A-Za-z_]+)[[:space:]]+ADD COLUMN[[:space:]]+([A-Za-z_]+).*/\1.\2/p' \
		-e 's/.*CREATE TABLE[[:space:]]+(IF NOT EXISTS[[:space:]]+)?([A-Za-z_]+).*/table:\2/p' |
	sort -u)

if [[ -z $upstream ]]; then
	echo "check-upstream-drift: extracted no migrations from $migrations_dir — extraction broken?" >&2
	exit 2
fi

# Our side: walk the migrations table in schema_drift_test.go. Each entry
# contributes table.column, or a bare table when the entry has no column
# (whole-table migrations like read_files).
ours=$(awk -v guard="$guard_test" '
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
' "$guard_test" | sort -u)

if [[ -z $ours ]]; then
	echo "check-upstream-drift: extracted no entries from $guard_test — parsing broken?" >&2
	exit 2
fi

if ! diff=$(diff -u <(printf '%s\n' "$ours") <(printf '%s\n' "$upstream")); then
	cat >&2 <<EOF
check-upstream-drift: DRIFT DETECTED against charmbracelet/crush $crush_ref

$diff

The upstream migration set no longer matches the capability guard. Fix:
  1. add every new table.column above to the migrations table in
     $guard_test — probed (plus a Schema capability and NULL
     substitution) when any query reads it, listed as unread otherwise
  2. run the guard: go test -run TestUpstreamMigrationColumnsAreProbedOrExempt .
  3. bump crush_ref/crush_sha at the top of this script to the release
     you verified, and record the re-verification in AGENTS.md
EOF
	exit 1
fi

echo "check-upstream-drift: OK (crush $crush_ref @ ${crush_sha:0:7}, $(printf '%s\n' "$upstream" | wc -l) non-initial migrations)"
