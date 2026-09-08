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
	"janekbaraniewski/openusage|pr|357|openusage fix PR — G1 gate"
	"Pilan-AI/mnemo|pr|22|mnemo fix PR — G2 gate"
	"charmbracelet/crush|pr|3576|our migration-comment fix PR"
	"charmbracelet/crush|issue|3580|our parts-reader inventory issue"
	"charmbracelet/crush|pr|3581|our zstd parts-compression PR"
)

declare -i resolved=0
declare -A verdicts=()

echo "check-upstream-status: filing-campaign threads (2026-09-08 campaign, TODO_LIST T35)"

for thread in "${threads[@]}"; do
	IFS='|' read -r repo kind number label <<<"$thread"

	endpoint=issues
	[[ $kind == pr ]] && endpoint=pulls

	json=$(gh api "repos/$repo/$endpoint/$number" 2>/dev/null) || json=""

	if [[ -z $json ]]; then
		printf '  %-30s %-16s %s\n' "$repo#$number" "ERROR" "gh call failed — auth/network? ($label)"
		continue
	fi

	resolved+=1

	state=$(jq -r '.state' <<<"$json")
	merged=$(jq -r '.merged_at // empty' <<<"$json")
	title=$(jq -r '.title' <<<"$json" | cut -c1-46)

	verdict="OPEN"
	[[ -n $merged ]] && verdict="MERGED"
	[[ -z $merged && $state == closed ]] && verdict="CLOSED-unmerged"

	verdicts["$repo#$number"]=$verdict

	printf '  %-30s %-16s %s\n' "$repo#$number" "$verdict" "$title ($label)"
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
	title=$(jq -r '.data.repository.discussion.title' <<<"$discussion_json" | cut -c1-46)
	printf '  %-30s %-16s %s\n' "crush discussion #3740" "$comments comments" "$title (the read-access ask; $reactions reactions)"
else
	echo "  crush discussion #3740     ERROR            GraphQL call failed (the read-access ask)"
fi

echo
echo "Phase G verdict (fix first, suggest second):"
if [[ ${verdicts[janekbaraniewski/openusage#357]:-unknown} == MERGED ]]; then
	echo "  G1 openusage: fix MERGED — adoption suggestion UNBLOCKED (post the plan's draft; re-verify claims first)"
else
	echo "  G1 openusage: fix PR not merged yet — suggestion stays blocked"
fi

if [[ ${verdicts[Pilan-AI/mnemo#22]:-unknown} == MERGED ]]; then
	echo "  G2 mnemo: fix MERGED — adoption suggestion UNBLOCKED (post the plan's draft; re-verify claims first)"
else
	echo "  G2 mnemo: fix PR not merged yet — suggestion stays blocked"
fi

if ((resolved == 0)); then
	echo "check-upstream-status: no thread resolved — gh is failing (auth? network?)" >&2
	exit 1
fi
