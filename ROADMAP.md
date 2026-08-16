# Roadmap

Long-term direction and raw ideas. Items graduate to [TODO_LIST.md](TODO_LIST.md)
when they become actionable and bounded. This file records *why*, not *when*.

## Direction

This library is the community drift sentinel for Crush's undocumented local
data. The roadmap optimizes for: correctness under schema drift, zero
friction for downstream consumers (crush-daily, mindwalk), and boring
infrastructure that fails loudly before shipping.

## Raw ideas (not yet actionable)

- **Typed Todos decoding helper** — a best-effort `DecodeTodos` mirroring
  how Crush writes todo lists, once more than one consumer needs it. Until
  then raw JSON is the honest contract.
- **Streaming message iteration** — an iterator (`func(yield ...) bool`)
  over messages for huge sessions, if a consumer materializes one.
- **Registry watching** — fsnotify on projects.json for live dashboards.
  Needs a consumer first; read-only polling is fine today.

## Recorded non-decisions (anti-drift)

These were consciously evaluated and rejected. Do not re-litigate without
new information.

- **No DOMAIN_LANGUAGE.md.** The domain is small and fully captured in
  doc.go plus type doc comments; a separate glossary would drift from them.
  Revisit if the type count doubles.
- **No AgentGraph CTE rewrite.** The recursive per-parent query is O(depth)
  queries; a recursive CTE would collapse it to one. Graphs are shallow
  (≤3 levels in practice) and the read is microseconds — the rewrite is
  complexity without a measured bottleneck. Revisit only when a consumer
  profiles AgentGraph hot on a deep graph.
- **No stats-SQL rewrite, ever unilaterally.** The parity contract with
  crush-daily is law; see CONTRIBUTING.md before touching `stats.go`.
- **No new dependencies.** The whole point is being the cheap, boring,
  zero-CGO reader. modernc.org/sqlite is the only allowed weight.
- **No config surface.** It is a read-only library; options structs cover
  legitimate variation without env vars or files.
- **No live coverage badge.** The static "≥85% enforced" badge states the
  invariant; CI uploads the exact HTML report as an artifact for anyone who
  wants the number. A live badge would add an account/endpoint dependency
  for zero enforcement value.
