# Filing campaign plan — 2026-09-08

Comprehensive plan for the upstream/ecosystem filing campaign. Every task
≤12 min. Execution order = dependency order; table in the session report
is sorted by importance. All external filings require explicit user
go-ahead before execution (per repo process rules).

Status markers: [ ] todo · [x] done.

## Phase A — Verify (gates, run before any filing)

- [ ] **A1** Verify openusage tests don't pin millis behavior: read
      `internal/providers/crush/provider_test.go` + `paths_test.go`,
      apply fix locally, run `go build ./... && go test ./...` on their
      module. 10m
- [ ] **A2** Verify mnemo tests don't pin millis behavior: same on
      `cmd/index_crush.go` (their clone already at
      /tmp/crush-ecosystem). 10m
- [ ] **A3** Check CONTRIBUTING/PR conventions of both repos (DCO, CLA,
      commit style) + confirm gh auth can fork/create PRs. 8m

## Phase B — openusage fix PR (receipt #1)

- [ ] **B1** Fork openusage, branch `fix/timestamps-unix-seconds`. 5m
- [ ] **B2** Apply one-line fix (`time.UnixMilli` → `time.Unix` in
      `millisToTime`) + add a date-pinning test. 10m
- [ ] **B3** Run their full suite locally on the branch. 10m
- [ ] **B4** Push + open PR. Body: real-DB evidence, root cause =
      charmbracelet/crush migration comment (PR #3576 pending), plus a
      short "Beyond this fix" paragraph: go-crush-data exists for
      exactly this format (schema-drift probing, tolerant parts), link
      the Discussion once posted. Suggest, never swap. 10m

## Phase C — mnemo fix PR (receipt #2)

- [ ] **C1** Fork mnemo, branch `fix/timestamps-unix-seconds`. 5m
- [ ] **C2** Apply fix in `cmd/index_crush.go` (line ~106). 5m
- [ ] **C3** Run their build + tests locally. 10m
- [ ] **C4** Push + open PR. Same description pattern as B4. 10m

## Phase D — Upstream Discussion (the ask + the endgame)

- [ ] **D1** Re-read `docs/upstream-read-access-discussion-draft.md`;
      insert the two fix-PR links + "fixed this morning" receipts, and
      add the adoption offer: if Crush ever wants an official read
      surface, go-crush-data (MIT, tested against v0.92.0, census-
      backed) is offered as a candidate basis — the "merge into crush
      for everybody's stability" pathway. 10m
- [ ] **D2** Fetch charmbracelet/crush Ideas category ID via GraphQL. 5m
- [ ] **D3** Post Discussion (createDiscussion mutation), verify
      rendering + cross-links. 10m

## Phase E — Loop-closing upstream comments

- [ ] **E1** Comment on crush PR #3576: evidence chain — 2 of 6 Go
      readers shipped UnixMilli date bugs citing that comment, both now
      fixed (PR links), Discussion posted. 10m
- [ ] **E2** Comment on crush #3580 (ours): ecosystem reader inventory
      — who parses messages.parts today (blast radius for compression;
      supports PR #3581). 8m

## Phase F — Adoption + bookkeeping

- [ ] **F1** Comment on vshulcz/deja-vu#2949: verified format facts
      (unix seconds, 8-discriminator envelope, registry path) + pointer
      to go-crush-data as a ready-made Go reader. 8m
- [ ] **F2** Update TODO_LIST: T9 superseded by fix-PRs (B4/C4); link
      posted Discussion in Parked section. 8m
- [ ] **F3** Batched response-monitoring pass over both PRs + the
      Discussion (repeat daily). 10m/pass

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
- [ ] **G3** crunch + crush-tmux: lightweight suggestion Issues
      (crunch gains registry discovery instead of filesystem walks;
      crush-tmux marginal — send only if a natural thread exists). 12m

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
