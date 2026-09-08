# Filing campaign plan — 2026-09-08

Comprehensive plan for the upstream/ecosystem filing campaign. Every task
≤12 min. Execution order = dependency order; table in the session report
is sorted by importance. All external filings require explicit user
go-ahead before execution (per repo process rules).

Status markers: [ ] todo · [x] done.

## Phase A — Verify (gates, run before any filing)

- [x] **A1** Verify openusage tests don't pin millis behavior: read
      `internal/providers/crush/provider_test.go` + `paths_test.go`,
      apply fix locally, run `go build ./... && go test ./...` on their
      module. 10m
- [x] **A2** Verify mnemo tests don't pin millis behavior: same on
      `cmd/index_crush.go` (their clone already at
      /tmp/crush-ecosystem). 10m
- [x] **A3** Check CONTRIBUTING/PR conventions of both repos (DCO, CLA,
      commit style) + confirm gh auth can fork/create PRs. 8m

## Phase B — openusage fix PR (receipt #1)

- [x] **B1** Fork openusage, branch `fix/timestamps-unix-seconds`. 5m
- [x] **B2** Apply one-line fix (`time.UnixMilli` → `time.Unix` in
      `millisToTime`) + add a date-pinning test. 10m
- [x] **B3** Run their full suite locally on the branch. 10m (3
      pre-existing failures verified on pristine main: daemon
      change-detection, hermes/zed missing-DB — unrelated.)
- [x] **B4** Push + open PR →
      [openusage#357](https://github.com/janekbaraniewski/openusage/pull/357).
      Body: real-DB evidence, root cause =
      charmbracelet/crush migration comment (PR #3576 pending), plus a
      short "Beyond this fix" paragraph: go-crush-data exists for
      exactly this format (schema-drift probing, tolerant parts), link
      the Discussion once posted. Suggest, never swap. 10m

## Phase C — mnemo fix PR (receipt #2)

- [x] **C1** Fork mnemo, branch `fix/timestamps-unix-seconds`. 5m
- [x] **C2** Apply fix in `cmd/index_crush.go` (line ~106). 5m
- [x] **C3** Run their build + tests locally. 10m (full suite green.)
- [x] **C4** Push + open PR →
      [mnemo#22](https://github.com/Pilan-AI/mnemo/pull/22). Same
      description pattern as B4. 10m

## Phase D — Upstream Discussion (the ask + the endgame)

- [x] **D1** Re-read `docs/upstream-read-access-discussion-draft.md`;
      insert the two fix-PR links + "fixed this morning" receipts, and
      add the adoption offer: if Crush ever wants an official read
      surface, go-crush-data (MIT, tested against v0.92.0, census-
      backed) is offered as a candidate basis — the "merge into crush
      for everybody's stability" pathway. 10m
- [x] **D2** Fetch charmbracelet/crush Ideas category ID via GraphQL. 5m
      (DIC_kwDOOt6mSM4Cs27u)
- [x] **D3** Post Discussion →
      [#3740](https://github.com/charmbracelet/crush/discussions/3740);
      body verified byte-identical after posting (later edited via
      GraphQL to correct a deja-vu "not merged" claim after their
      parser landed, PR #3158). 10m

## Phase E — Loop-closing upstream comments

- [x] **E1** Comment on crush PR #3576: evidence chain — 2 of 6 Go
      readers shipped UnixMilli date bugs citing that comment, both now
      fixed (openusage#357, mnemo#22), Discussion posted. 10m
- [x] **E2** Comment on crush #3580 (ours): ecosystem reader inventory
      — who parses messages.parts today (blast radius for compression;
      supports PR #3581). 8m

## Phase F — Adoption + bookkeeping

- [x] **F1** Comment on vshulcz/deja-vu#2949: verified format facts
      (unix seconds, 8-discriminator envelope, registry path) + pointer
      to go-crush-data as a ready-made Go reader. 8m (thread was already
      closed-completed with solid recon — comment adds only the delta:
      CRUSH_GLOBAL_DATA order, bug receipts validating their unixGuess,
      census data point.)
- [x] **F2** Update TODO_LIST: T9 superseded by fix-PRs (B4/C4); T20
      superseded by crush PR #3576; posted Discussion linked in Parked;
      T35 added for monitoring + Phase G gating. 8m
- [x] **F3** Batched response-monitoring pass over both PRs + the
      Discussion (repeat daily). 10m/pass — first pass 2026-09-08:
      no responses yet (posted minutes earlier); PRs open, discussion
      live at 0 reactions. Recurring duty now tracked as TODO_LIST T35.

## Phase G — Ecosystem adoption suggestions (after fix PRs merge)

Strategy: fix first (credibility), suggest second. Every suggestion is
a separate, respectful touchpoint — never a code swap inside a fix PR.
Go repos only; the Rust/TS repos (tokscale, CASS, history-viewer,
cli-continues) get nothing to consume — at most a courtesy link to the
ecosystem review where on-topic.

- [ ] **G1** openusage Issue/Discussion: consider go-crush-data for the
      crush provider. Selling points: schema-drift probing (their #1
      silent-breakage risk), correct registry resolution incl.
      CRUSH_GLOBAL_DATA, and a pure-Go driver story (modernc — dropping
      their cgo/mattn build dependency entirely). 12m
- [ ] **G2** mnemo Issue: same suggestion; they already use modernc, so
      adoption costs zero driver change; replaces their hardcoded
      columns + ~/.crush/crush.db path with probed, registry-driven
      reads. 12m
- [x] **G3** crunch + crush-tmux: lightweight suggestion Issues
      (crunch gains registry discovery instead of filesystem walks;
      crush-tmux marginal — send only if a natural thread exists). 12m
      — crunch: [taigrr/crunch#22](https://github.com/taigrr/crunch/issues/22)
      filed 2026-09-08 (timestamp handling verified correct first —
      unix-seconds in SQL, no third bug). crush-tmux: SKIPPED — zero
      open/closed issues exist, no natural thread (per this plan's own
      condition); revisit if one appears.

### G1/G2 status 2026-09-08

Gated on openusage#357 + mnemo#22 merging (both OPEN at session end;
external maintainer action). Posting before merge would break the
fix-first-then-suggest strategy. Drafts below are ready to post;
**re-verify claims at post time** (note: openusage's cgo claim was
corrected — their Cursor/telemetry stores also use mattn, so adopting
go-crush-data for the crush provider alone does not drop the cgo
dependency; the honest angle is "no new cgo surface + drift probing").

#### G1 draft (openusage Issue, post after openusage#357 merges)

Title: Idea: crush provider — schema-drift probing via go-crush-data?

Body:

> Following up on #357 (now merged): the deeper risk it exposed isn't
> the one-line bug — it's that the crush schema is an undocumented,
> moving target. Two concrete upcoming examples from upstream:
> message-parts compression (charmbracelet/crush#3580/#3581) would
> break every JSON-parsing reader silently, and goose migrations evolve
> the schema invisibly to readers.
>
> If you ever want the crush provider to degrade gracefully instead of
> breaking, [go-crush-data](https://github.com/LarsArtmann/go-crush-data)
> (MIT, mine) exists for exactly this: capability probing via
> pragma_table_info (strict error split: "recognized schema, missing
> columns" vs corrupt DB), correct registry resolution
> (CRUSH_GLOBAL_DATA → XDG → LOCALAPPDATA, plus `crush projects --json`
> fallback), tolerant parts decoding, tested against crush v0.92.0 and
> 287 real databases. It uses modernc.org/sqlite (pure Go) — no new
> cgo surface, though I realize Cursor/telemetry keep mattn regardless,
> so this isn't a cgo-elimination pitch.
>
> Purely a suggestion — the provider works as-is. Context: upstream
> Discussion about supported read access for ecosystem tools:
> https://github.com/charmbracelet/crush/discussions/3740

#### G2 draft (mnemo Issue, post after mnemo#22 merges)

Title: Idea: registry-driven crush discovery + schema probing via go-crush-data?

Body:

> Following up on #22 (now merged): the fix was one line, but the
> surrounding surface is still hand-rolled — hardcoded column lists,
> the ~/.crush/crush.db path, and no drift detection.
>
> [go-crush-data](https://github.com/LarsArtmann/go-crush-data) (MIT,
> mine) packages the same knowledge: registry-driven discovery
> (projects.json: every project ever opened, incl. custom data_dir
> locations and CRUSH_GLOBAL_DATA/XDG/LOCALAPPDATA resolution — more
> complete than enumerating candidate paths), schema capability probing
> so migrations degrade gracefully instead of panicking the indexer,
> tolerant parts decoding, tested against crush v0.92.0 and 287 real
> databases. It uses modernc.org/sqlite — same driver mnemo already
> ships, so adoption costs zero driver change.
>
> Purely a suggestion — happy to help if it's interesting. Context:
> upstream Discussion about supported read access for ecosystem tools:
> https://github.com/charmbracelet/crush/discussions/3740

## Dependency chain

A1→B1→B2→B3→B4 ┐
A2→C1→C2→C3→C4 ┴→ D1→D2→D3 → E1
E2, F1 independent of D
F2 after D3; F3 after B4/C4/D3
G1/G2 after B4/C4 merge; G3 anytime after D3

## Out of scope (tracked elsewhere)

- TODO_LIST T1–T7 (Renovate, fuzz, flake-lock, pkg.go.dev, corpus
  mining) and T10+ from the concurrent session — untouched by this plan.
- No new charmbracelet/crush Issues (feature asks belong in
  Discussions; bug PR #3576 already open).
- No code swaps inside fix PRs (adoption happens via Phase G
  suggestions after credibility is established).
