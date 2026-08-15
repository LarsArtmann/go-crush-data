# TODO List

Short- and mid-term improvement tasks, harvested from the 2026-08-15 full
code review (`docs/reviews/2026-08-15_21-30_full-code-review.html`). Status
changes as items are done; new review findings land here.

## Open

- [ ] **Add a context-aware Open variant** — `Open(dataDir string)` cannot be
  cancelled; `probeSchema` runs on `context.Background()`. Add
  `OpenContext(ctx, dataDir)` (and keep `Open` delegating to it) so callers
  can bound startup on slow filesystems. ~30 min.
- [ ] **Batch AgentGraph traversal** — `AgentGraph` issues one query per node
  (N+1). Fine for realistic graphs (depth cap 64, small fan-out), but a
  recursive CTE over `parent_session_id` would make it a single query if any
  consumer ever reports it hot. Measure first. ~2 h.
- [ ] **Type `Session.Todos` as JSON** — it is an opaque JSON blob stored as
  a plain `string`. Exposing `json.RawMessage` (or a decoded `[]Todo`) would
  make "this is structured data" explicit. Breaking-ish API change; bundle
  with the next minor. ~1 h.
- [ ] **Distinguish "column missing" from "probe failed"** —
  `tableExists`/`columnExists` return false on query errors, so a broken
  database can masquerade as a legacy schema. Consider surfacing probe errors
  from `Open`. Needs a design decision on strictness. ~1 h.

## Done

- [x] Fix `SessionFilter` condition/argument order drift (`sessions.go`) —
  fixed 2026-08-15 during the review; regression test
  `TestSessionsByIDComposesWithOtherFilters`.
- [x] Fix `AgentGraph` sibling ordering to follow `created_at`, not reversed
  `updated_at` (`agents.go`) — fixed 2026-08-15 during the review; regression
  test `TestAgentGraphSiblingsOrderedByCreatedNotUpdated`.
- [x] Align `doc.go` day-filter documentation with the tested semantics —
  fixed 2026-08-15 during the review.
- [x] Delete dead test helpers `fixtureDB`, `insertLegacySession` — done
  2026-08-15 during the review.
