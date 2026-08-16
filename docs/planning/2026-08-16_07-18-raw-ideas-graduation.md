# Raw Ideas Graduation — 2026-08-16 07:18

Execute the three ROADMAP raw ideas: implement what is now actionable, give
the third an honest home, keep the recorded non-decisions intact.

## Evidence gathered before planning

- **Todos format pinned by census** (this session): 71,747 items across all
  287 data dirs in the local registry — every item carries exactly
  `content`, `status`, `active_form`; statuses observed: `completed`
  (62,224), `pending` (8,225), `in_progress` (1,298); zero malformed arrays.
  The "more than one consumer" gate guarded against guessing the shape
  wrong; that risk is retired by evidence.
- **go-filewatcher v2** (`/home/lars/projects/go-filewatcher`, module
  `github.com/larsartmann/go-filewatcher/v2`) watches directories, not
  single files; `New(paths, WithExtensions, WithDebounce) ... Watch(ctx)`
  returns `<-chan Event` with `Event.Path` absolute. It depends on fsnotify
  + 3 more modules — adopting it would violate this repo's recorded
  "No new dependencies" non-decision (ROADMAP.md). Watching therefore stays
  consumer-side; this repo contributes a verified composition recipe.

## Plan (pareto order)

| # | Task | Outcome | Status |
|---|---|---|---|
| 1 | `DecodeTodos` + `Todo`/`TodoStatus` in `todos.go`; census-derived tests; example | Roadmap idea 1 graduates to shipped API | DONE — verified: unit+census+E2E tests green; 60,404 real sessions scanned through the public API, 71,805 items decoded, 0 failures; fuzz target 2.1M execs PASS |
| 2 | `DB.IterMessages` (`iter.Seq2[Message, error]`) sharing one `scanMessage` with `Messages`; tests + example | Roadmap idea 2 graduates to shipped API | DONE — verified: parity/early-break/empty/canceled-ctx tests green under `-race -shuffle=on` |
| 3 | Out-of-tree verification of the go-filewatcher recipe against a temp registry, then `docs/recipes/registry-watching.md` | Roadmap idea 3 becomes a documented pattern with a verified recipe, zero new deps | DONE — verified: harness printed `RECIPE VERIFIED: 1 project before, registry event fired, 2 projects after` (2026-08-16) |
| 4 | Docs sweep: README, FEATURES, CHANGELOG `[Unreleased]`, ROADMAP graduation, TODO_LIST (adoption task T8), AGENTS.md storage facts | Truth discipline upheld | DONE |
| 5 | Full gate + `CRUSH_DATA_REAL_DATA_DIR` re-run | Verified, not annotated-before-verified | DONE — `GATE_GREEN` (build, vet, race+shuffle, lint 0 issues, `nix flake check`, actionlint, doc-links) + `TestSessionsOnRealDatabase` PASS |

## Non-goals

- No fsnotify/go-filewatcher dependency in this module (recorded
  non-decision stands).
- No tolerant per-item todos decoding (census shows zero malformed items;
  add it when real drift appears).
- No changes to `Session.Todos` (`json.RawMessage` contract is law).
