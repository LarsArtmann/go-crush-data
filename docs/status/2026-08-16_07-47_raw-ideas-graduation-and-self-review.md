# Status Report — Raw Ideas Graduation & Self-Review

**Date:** 2026-08-16 07:47 (local) · **Session scope:** executing the three
ROADMAP raw ideas (typed todos, streaming iteration, registry watching)
plus the self-review that followed. Everything below is about THIS session's
run. Format note: Markdown per explicit user request (skill default is HTML).

---

## a) FULLY DONE

Verifiably complete; evidence cited. (Note: nothing this session is
*committed* yet — code work that is done-but-uncommitted sits in (b) by the
"committed, tested, working" rule. These items are complete as one-shot
verifications or checks.)

1. **Todos format census (the gating evidence).** All 287 data dirs in the
   local real registry scanned: 71,747 todo items, every one exactly
   `{content, status, active_form}`, statuses only
   `completed`(62,224)/`pending`(8,225)/`in_progress`(1,298), zero
   malformed arrays, zero extra keys. Evidence: census run output in
   session; recorded in AGENTS.md storage facts + todos.go doc comment.
2. **Registry-watching recipe verified out-of-tree.** Harness at
   /tmp/fwatch-verify (go-filewatcher v2 + this library via `replace`)
   printed `RECIPE VERIFIED: 1 project before, registry event fired,
   2 projects after`. → docs/recipes/registry-watching.md. Linux only.
3. **Scale validation of DecodeTodos through the public API.** 60,404 real
   sessions, 71,805 items decoded, **0 failures**; status distribution
   matches the census (+58 items vs. raw-SQL census — registry changed
   between runs, live system).
4. **Local gate green after all changes.** build, vet, `-race -shuffle=on`,
   golangci-lint **0 issues**, `nix flake check` (with new files visible via
   `git add -N`), actionlint, `check-doc-links.sh` → `GATE_GREEN`;
   `TestSessionsOnRealDatabase` PASS; coverage **87.8%** (≥85% gate, same
   as the documented baseline). Fuzz: `FuzzDecodeTodos` 2,143,838 execs in
   15s, 0 failures.
5. **fuzz.yml matrix gap found and fixed (during this self-review).** The
   nightly matrix hardcoded three targets; `FuzzDecodeTodos` was missing.
   Now added; actionlint clean; FEATURES row updated to four parsers.

## b) PARTIALLY DONE

Done and locally verified, but **uncommitted — CI (ubuntu/windows/macos
legs) has never seen any of it**. Blocking: commit/push needs user
instruction (question g/1).

1. **`DecodeTodos` + `Todo`/`TodoStatus`** (todos.go, todos_test.go,
   example, fuzz target). Works, tested, scale-proven. Gaps: uncommitted;
   Windows/macOS legs unverified. Effort to finish: S (commit + observe CI).
2. **`DB.IterMessages`** (messages.go:49, shared `scanMessage` refactor).
   Parity/early-break/empty/canceled-ctx tests green under race+shuffle.
   Gaps: uncommitted; **no benchmark** — the repo keeps a committed bench
   baseline (Sessions/Messages/AgentGraph) and I added a read path without
   one. Effort: S (BenchmarkIterMessages) + M if regenerating baseline
   (`-count=6` per AGENTS.md).
3. **docs/recipes/registry-watching.md.** Written from the verified
   harness. Gaps: uncommitted; verified on Linux only (fsnotify/inotify
   paths; go-filewatcher self-documents cross-platform support but my
   harness did not prove it).
4. **Docs sweep** (README, FEATURES, CHANGELOG `[Unreleased]`, ROADMAP
   graduation + new non-decision, TODO_LIST T8, AGENTS.md, plan record
   docs/planning/2026-08-16_07-18-raw-ideas-graduation.md with per-task
   verification). Gap: uncommitted; FEATURES/CHANGELOG claims become
   CI-true only after push + green legs.
5. **parts.go `jsonNull` constant** (drive-by goconst fix). Tested
   (`TestDecodeParts` green). Gap: uncommitted; deserves a line in the
   eventual commit message.

## c) NOT STARTED

1. **Commit + push + CI observation** of this session's work. Not started
   by design: user rule (never commit unless told). Priority: Critical —
   everything in (b) is one `git push` away from real verification.
2. **T8 — adopt DecodeTodos/IterMessages in crush-daily**
   (TODO_LIST, 30m). The "second consumer" the todos helper was gated on.
   Not started; needs user go-ahead (question g/3).
3. **BenchmarkIterMessages + baseline decision.** Not started; see (b)/2.
4. **v0.3.0 release.** CHANGELOG `[Unreleased]` now carries two `Added`
   entries (DecodeTodos, IterMessages) on top of Fixed/Changed → minor
   bump by the repo's SemVer policy. Not started; owner decision
   (question g/2). RELEASING preconditions (all legs green on origin)
   apply.
5. **Harvesting this report's (f) list into TODO_LIST/ROADMAP**
   (docs-health HARVEST). Deferred: user said report, then wait.
6. **doc.go mention of the new APIs.** doc.go is conceptual, not an API
   index, so nothing is false today; a sentence on streaming/todos would
   still fit its "Schema drift"/"Read-only" narrative. Low priority.

## d) TOTALLY FUCKED UP

Nothing shipped broken — the final state passes every local gate. The
honest failures are process failures:

1. **messages.go edit splice briefly broke the build.** My 3-edit batch
   spliced a new function head onto the old closure body (one edit was a
   no-op placeholder — sloppy construction). Severity: low (caught by the
   immediate `go build`, fixed in one step). Root cause: composing a
   large structural edit as find/replace instead of one precise edit.
2. **goconst detour: ~6 wasted round trips.** The linter's first message
   literally said "make it a constant"; I instead ran comment-rewording
   archaeology, including a botched sed experiment that *disproved* my
   comment-counting theory while lint still reported 3 occurrences — and
   I never established the real counting mechanism. The final fix was the
   mechanical constant I could have applied first. Time burned, zero
   knowledge gained. Lesson: **read the linter's remedy before
   theorizing about its detector.**
3. **I claimed fuzz coverage I had not wired.** Adding FuzzDecodeTodos +
   a FEATURES row implying nightly coverage — without checking that
   fuzz.yml's matrix hardcodes target names. Found only in this
   self-review; fixed minutes ago. This is exactly the doc-drift class
   this repo's tooling exists to catch, and I introduced it by hand.
4. **Untargeted `golangci-lint fmt` mid-session.** I ran the repo-wide
   rewriter when only my files needed it. Harmless only because master
   was clean; it could have rewritten unrelated files in a dirty tree.
5. **Stale LSP diagnostics ignored all session.** godox/goconst warnings
   on todos.go appeared in every tool result after the CLI already passed
   (the "A Todo is one entry…" reword happened at minute one; the
   diagnostic never updated). I silently trusted the CLI — correct call,
   but the LSP/CLI divergence was never investigated or reported until
   now. Unknown cause; deserves one look (see e/4).

## e) WHAT WE SHOULD IMPROVE

1. **Linter remedy first, archaeology second.** When a linter states the
   fix ("make it a constant"), apply it; investigate the detector only if
   the fix is wrong. Impact: this session, ~6 round trips ≈ 10% of the
   session's tool budget.
2. **CI-adjacent claims require touching the CI file in the same
   change.** Any new test/fuzz target ⇒ grep `.github/workflows/` for
   hardcoded lists before writing "covered by nightly" anywhere.
3. **New read path ⇒ benchmark in the same change** (repo convention:
   committed baseline + benchstat trend). IterMessages shipped without
   one; the trend CI is blind to it until added.
4. **Resolve or pin the LSP-vs-CLI lint divergence.** One investigation;
   if unresolvable, an AGENTS.md line "gopls-class diagnostics may serve
   stale golangci findings; the `nix run .#lint` result is authoritative"
   — mirroring the existing cross-tree cache gotcha.
5. **Targeted formatting.** `golangci-lint fmt <files>` where possible;
   repo-wide fmt only at session end.
6. **Edit hygiene on structural changes:** one precise edit over a
   context-rich anchor, never a batch containing no-op filler edits.

## f) Next tasks (ranked)

| # | Task | Impact | Effort | Category |
|---|---|---|---|---|
| 1 | Commit this session's work (see question g/1 for granularity) | Critical | S | Cleanup |
| 2 | Push + observe all three CI legs green | Critical | S | CI |
| 3 | T8: adopt DecodeTodos/IterMessages in crush-daily | High | M | Feature |
| 4 | Add BenchmarkIterMessages; regenerate baseline if trend shifts | High | M | Quality |
| 5 | Cut v0.3.0 after legs green (2× Added ⇒ minor) | High | S | Release |
| 6 | Verify pkg.go.dev renders v0.3.0 + spot-check new symbols (extends T4) | Medium | S | Docs |
| 7 | Observe first nightly fuzz incl. FuzzDecodeTodos (extends T2) | Medium | S | CI |
| 8 | HARVEST this report's (f) into TODO_LIST/ROADMAP | Medium | S | Docs |
| 9 | Investigate LSP-vs-CLI lint divergence; pin note in AGENTS.md | Medium | S | Tooling |
| 10 | Reproduce goconst's occurrence counting (comments? tests?) and document it in AGENTS.md tooling gotchas | Medium | S | Tooling |
| 11 | Cross-platform note for the recipe (or a macOS runner check for the harness pattern) | Low | S | Docs |
| 12 | doc.go: one sentence each for streaming + todos under existing sections | Low | S | Docs |
| 13 | T1 (Renovate app install) — still external, config validates | Medium | S | CI |
| 14 | T3 (first flake-lock PR observation) | Low | S | CI |
| 15 | T6 (mine fuzz corpus seeds once nightly runs exist) | Low | M | Quality |
| 16 | T7 (pin action SHAs via Renovate after T1) | Low | S | CI |
| 17 | Upstream mindwalk PR (Parked; needs go-ahead) | Medium | M | Ecosystem |
| 18 | File/refresh the charmbracelet/crush schema-docs issue; link it in Parked | Medium | S | Ecosystem |
| 19 | Consider a `DecodeTodos` strictness knob ONLY if a real consumer hits drift (none has) | Low | S | Feature |
| 20 | Re-run census after next Crush release; update pinned shape if drifted | Low | S | Maintenance |
| 21 | Add `IterMessages` + `DecodeTodos` to example_test.go "all APIs" coverage check (both have examples; keep the invariant: every public API has a runnable example) | Low | S | Docs |

(21 real items; padded filler withheld — the skill allows up to 50, not
a quota.)

## g) Questions I cannot answer myself

1. **Commit granularity for this session's work.** Options: (a) one commit
   "feat: DecodeTodos + IterMessages + registry-watching recipe", (b)
   split — feature code / recipe+docs / fuzz+workflow fix, (c) you review
   the diff first. I cannot know your preference, and the auto-commit
   daemon may race me — say the word and I commit per your choice.
2. **Cut v0.3.0 once CI legs are green, or hold for T8 (crush-daily
   adoption) first?** Releasing before adoption makes the API public
   before its second consumer validates the ergonomics; holding keeps
   `[Unreleased]` growing. Owner call.
3. **Is T8 mine to execute (next session in ~/projects/crush-daily), or
   yours?** The task lives in this repo's TODO_LIST with an estimate, but
   it edits a different codebase — I need your go-ahead before touching it
   (same class as the Parked mindwalk PR).

---

*Report ends. Waiting for instructions. Section (f) is pending HARVEST into
TODO_LIST/ROADMAP until instructed.*
