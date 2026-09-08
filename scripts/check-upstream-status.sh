#!/usr/bin/env bash
#
# One-command status of the 2026-09-08 filing campaign: prints the state of
# every thread we filed on, plus the Phase G verdict — the openusage and
# mnemo adoption suggestions (G1/G2 in docs/status/2026-09-08_filing-
# campaign-plan.md) unblock exactly when their fix PRs merge.
#
# Replaces the hand-rolled daily monitoring pass behind TODO_LIST T35.
#
# Usage:
#   scripts/check-upstream-status.sh
#
# Requires: gh, authenticated. Exits 0 as long as at least one thread
# resolved; exits 1 when gh failed everywhere (auth/network problem, not an
# upstream state).

set -euo pipefail

threads=(
	"janekbaraniewski/openusage|pr|357|openusage fix PR (G1 gate)"
	"Pilan-AI/mnemo|pr|22|mnemo fix PR (G2 gate)"
	"charmbracelet/crush|pr|3576|our migration-comment fix PR"
	"charmbracelet/crush|issue|3580|our parts-reader inventory issue"
	"charmbracelet/crush|pr|3581|our zstd parts-compression PR"
)

declare -i resolved=0
declare -a fix_states=()

fetch_state() {
	local repo=$1 kind=$2 number=$3
	local json state merged title
	if json=$(gh api "repos/$repo/$kind s/$number" 2>/dev/null); then :; fi
	# gh api path above intentionally has no space; reassemble correctly:
	:
}

# render_thread <repo> <pr|issue> <number> <label>: one status line.
render_thread() {
	local repo=$1 kind=$2 number=$3 label=$4
	local json state merged title verdict

	if ! json=$(gh api "repos/$repo/pulls/$number" 2>/dev/null) &&
		! json=$(gh api "repos/$repo/issues/$number" 2>/dev/null); then
		printf '  %-28s %-9s ERROR (gh call failed — auth/network?)\n' "$repo#$number" "$kind"
		return 1
	fi

	state=$(jq -r '.state' <<<"$json")
	merged=$(jq -r '.merged_at // empty' <<<"$json" 2>/dev/null)
	title=$(jq -r '.title' <<<"$json" | cut -c1-48)

	if [[ -n $merged ]]; then
		verdict="MERGED $merged"
	elif [[ $state == closed ]]; then
		verdict="CLOSED-unmerged"
	else
		verdict="OPEN"
	fi

	printf '  %-28s %-15s %s\n' "$repo#$number" "$verdict" "$title ($label)"
	# gh api "repos/$repo/pulls/$n" also answers for issues; "repos/$repo/issues/$n" also answers for PRs,
	# so the first successful call wins either way.

	return 0
}

echo "check-upstream-status: filing-campaign threads (2026-09-08 campaign, TODO_LIST T35)"

for thread in "${threads[@]}"; do
	IFS='|' read -r repo kind number label <<<"$thread"
	if render_thread "$repo" "$kind" "$number" "$label"; then
		resolved+=1
		if [[ $repo == janekbaraniewski/openusage || $repo == Pilan-AI/mnemo ]]; then
			fix_states+=("$repo:$(
				json=$(gh api "repos/$repo/pulls/$number" 2>/dev/null)
				if [[ $(jq -r '.merged_at // empty' <<<"$json") != "" ]]; then
					echo merged
				else
					echo open
				fi
			)")
		fi
	fi
done

# Discussion #3740 (GraphQL-only API): total comments is the activity signal.
discussion_json=$(gh api graphql -f query='
	query {
		repository(owner: "charmbracelet", name: "crush") {
			discussion(number: 3740) {
				title
				comments { totalCount }
				reactions { totalCount }
			}
		}
	}' 2>/dev/null || true)

if [[ -n $discussion_json ]]; then
	resolved+=1
	comments=$(jq -r '.data.repository.discussion.comments.totalCount' <<<"$discussion_json")
	reactions=$(jq -r '.data.repository.discussion.reactions.totalCount' <<<"$discussion_json")
	title=$(jq -r '.data.repository.discussion.title' <<<"$discussion_json" | cut -c1-48)
	printf '  %-28s %-15s %s\n' "crush discussion #3740" "$comments comments" "$title (the read-access ask)"
else
	echo "  crush discussion #3740     ERROR (GraphQL call failed)"
fi

echo
echo "Phase G verdict:"
for state in "${fix_states[@]}"; do
	repo=${state%%:*}
	verdict=${state##*:}
	if [[ $verdict == merged ]]; then
		echo "  ${repo}: fix MERGED — adoption suggestion unblocked (post the plan's draft, re-verify claims first)"
	else
		echo "  ${repo}: fix PR still open — suggestion stays BLOCKED (fix first, suggest second)"
	fi
done

if ((resolved == 0)); then
	echo "check-upstream-status: no thread resolved — gh is failing (auth? network?)" >&2
	exit 1
fi
