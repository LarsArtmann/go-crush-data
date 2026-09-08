#!/usr/bin/env bash
#
# Open a tracking issue when a new charmbracelet/crush STABLE release lands.
#
# Why: the AGENTS.md upstream-verification cadence (probe new columns, run
# the drift guard, re-run real-data sweeps) is driven by stable releases,
# but nothing notified us when one shipped — the weekly drift job compares
# migrations only after someone bumps the pin. This script detects "latest
# stable tag ≠ pinned crush_ref" and opens exactly one issue per release,
# so the cadence has a trigger instead of relying on memory.
#
# Usage (from the repo root; needs gh authenticated, or GH_TOKEN set):
#   scripts/check-upstream-release.sh
#
# Behavior:
#   - latest stable = highest vX.Y.Z release tag; nightlies
#     (vX.Y.Z-nightly.N) and draft/patch-channel noise are excluded
#   - pinned ref = the crush_ref line in scripts/check-upstream-drift.sh
#   - up to date  → prints OK, exits 0 (no issue)
#   - new release → opens an issue unless one for that version is already
#     open (idempotent across scheduled runs)

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

upstream_repo=charmbracelet/crush

pinned_ref=$(sed -nE 's/^crush_ref=(.+)$/\1/p' scripts/check-upstream-drift.sh)
if [[ -z $pinned_ref ]]; then
	echo "check-upstream-release: could not read crush_ref from scripts/check-upstream-drift.sh" >&2
	exit 2
fi

latest=$(
	gh release list --repo "$upstream_repo" --limit 100 |
		awk '{print $1}' |
		grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' |
		sort -V |
		tail -1
)

if [[ -z $latest ]]; then
	echo "check-upstream-release: found no stable vX.Y.Z release on $upstream_repo — listing broken?" >&2
	exit 2
fi

if [[ $latest == "$pinned_ref" ]]; then
	echo "check-upstream-release: OK (pinned $pinned_ref is the latest stable)"
	exit 0
fi

echo "check-upstream-release: new stable $latest (pinned: $pinned_ref)"

marker="crush $latest released"
if gh issue list --state open --search "in:title \"$marker\"" | grep -q .; then
	echo "check-upstream-release: an issue for $latest already exists — nothing to do"
	exit 0
fi

gh issue create --title "$marker — run the upstream verification cadence" --body "$(
	cat <<EOF
A new charmbracelet/crush stable release (\`$latest\`) landed; this repo pins
\`$pinned_ref\`. Run the AGENTS.md "Upstream verification cadence":

1. bump \`crush_ref\`/\`crush_sha\` at the top of \`scripts/check-upstream-drift.sh\`
   to the new tag/commit,
2. run the script (clones the pinned release, diffs migrations against the
   guard list) and its \`--self-test\`,
3. on drift: probe or document-exempt every new column/table, extend the
   guard list, re-run \`go test -run TestUpstreamMigrationColumnsAreProbedOrExempt .\`,
4. re-run the real-data sweeps and the parts census, then update the
   last-verified line in AGENTS.md,
5. refresh \`docs/storage-schema-*.md\` via \`scripts/genschema\` when the schema
   moved.

Opened automatically by the upstream-drift workflow.
EOF
)" >/dev/null

echo "check-upstream-release: issue opened for $latest"
