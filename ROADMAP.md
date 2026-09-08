# Roadmap

Long-term direction and raw ideas. Items graduate to [TODO_LIST.md](TODO_LIST.md)
when they become actionable and bounded. This file records _why_, not _when_.

## Direction

This library is the community drift sentinel for Crush's undocumented local
data. The roadmap optimizes for: correctness under schema drift, zero
friction for downstream consumers (crush-daily, mindwalk), and boring
infrastructure that fails loudly before shipping.

## Raw ideas (not yet actionable)

- Two live candidates below (`IterSessions`, `ReadFileVersions`). The last
  three graduates (typed todos decoding, streaming message iteration,
  registry watching) landed on 2026-08-16: the first two as `DecodeTodos`
  and `DB.IterMessages` (CHANGELOG `[Unreleased]`), the third as a
  documented consumer-side pattern
  ([recipes/registry-watching](docs/recipes/registry-watching.md)).

- **`IterSessions`** — iterator parity with `IterMessages` for registries
  whose session lists are too large to materialize. Only worth it once a
  consumer hits thousands of sessions per data dir; `SessionFilter.Limit`
  covers current consumers. Source: docs/status/2026-09-07_21-59…verification.md (f)19.
- **Exposing `files`-table snapshots (`ReadFileVersions`)** — upstream still
  writes file snapshots (v0.92.0 internal/history/file.go), the initial
  migration carries the table, and this library deliberately does not read
  it. Graduate only when a consumer asks for versioned file content.
  Source: same report (f)34.

## Open questions (need a product-scope call)

- **Summary fields and message timestamps**: should
  `Session.SummaryMessageID` and `Message.IsSummaryMessage` become public
  API (both exist since 2025-05/2025-08 migrations, both currently unread,
  both would be probe-gated), or is the library deliberately minimal for
  crush-daily's needs? `Message.UpdatedAt` already landed (initial-schema
  column, no probe needed). Source: docs/status/2026-09-07_21-59…verification.md (g)2/(f)10.

## Recorded non-decisions (anti-drift)

These were consciously evaluated and rejected. Do not re-litigate without
new information.

- **No in-library watching — including a `watch/` sub-module.** A direct
  fsnotify-class dependency would betray the zero-weight contract (see "No
  new dependencies" below). A sub-module would honor that contract but buys
  nothing: the entire glue is ~30 lines of recipe, and watching _policy_
  (debounce, filtering, what to do on event) is not worth pinning as API
  while zero consumers want typed registry events. The verified recipe
  stands in for both:
  [recipes/registry-watching](docs/recipes/registry-watching.md).
  Decided 2026-08-16; the sub-module variant explicitly re-rejected same
  day after evaluating it. Revisit when a real consumer wants a live loop
  (e.g. agent-trace adopting this library for tailing).

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
- **No branch protection on master.** Required CI status checks would also
  block direct pushes, which is how this repo works (local auto-commit
  daemon + manual pushes). The adopted guard instead is RELEASING.md
  precondition 4: a tag may only be cut once all matrix legs are green on
  origin. Decided 2026-08-16; revisit if the workflow moves to PRs.
