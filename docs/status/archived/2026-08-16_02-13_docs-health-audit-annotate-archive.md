# Status Report — Docs-Health Audit: Verify, Harvest, Annotate, Archive, 2026-08-16 02:13

**Session scope:** Execute the `docs-health` skill as a full AUDIT over all
10 files matching `**/2026-08-1*` (6 status reports, 2 planning docs, 2 HTML
artifacts): view all, VERIFY living docs against code/git/CI, HARVEST open
items, repair TODO_LIST/CHANGELOG/AGENTS/ROADMAP/FEATURES, ANNOTATE every
numbered item inline, ARCHIVE fully-resolved files. Baseline at session
start: `7733d36` (clean tree; master ahead 1 of origin). State at report
time: all audit edits uncommitted (daemon owns commits).

**Verification state at report time:** `go build` ✓ · `go vet` ✓ ·
`go test -race -shuffle=on` ✓ · golangci-lint 0 issues ✓ ·
`nix flake check` ✓ · actionlint ✓ · real-DB smoke test
(`CRUSH_DATA_REAL_DATA_DIR=./.crush`) PASS ✓ · both annotated HTML
artifacts structurally valid (tag-balance parser) ✓.

---

## a) FULLY DONE

1. **All 10 `2026-08-1*` files viewed in full** — 5 status .md + 00-40
   status .md + 2 planning .md + planning .html + review .html.
2. **VERIFY pass over all 6 living docs**, claims checked against code,
   `git log`, `gh run list`, `gh api`, and pkg.go.dev: found and fixed —
   stale `discover.go` citation cluster in FEATURES (file changed after the
   doc was written: :109→:237 etc.), `types.go:38`→:39, CI-matrix row
   missing the v0.2.0-tag caveat.
3. **CHANGELOG repaired**: `[0.2.0]` was used as a link reference but never
   defined, and `[Unreleased]` compared against v0.1.1 instead of v0.2.0 —
   both fixed (link refs now complete and correct).
4. **ROADMAP de-staled**: removed the shipped "v0.2.0 (next minor)" section
   (CHANGELOG owns shipped history), removed the resolved coverage-badge
   idea, added the recorded non-decision "No live coverage badge".
5. **AGENTS.md hardened**: 2 commit-hash references removed (endurance
   test), real-DB smoke rule widened to ANY source change, 2 new tooling
   gotchas (Windows POSIX assumptions in tests; `defaults.run.shell: bash`
   vs PowerShell). 7.2 KB — inside the 5–15 KB budget, 0 hashes, gotchas ≤
   10 rows.
6. **TODO_LIST rebuilt**: the ~95-line "Done" trophy section (≈85 %
   non-job content — the dominant structural-decay failure mode) deleted;
   17 verified-open items kept, freshly routed (pending-decisions /
   external / high / medium / low); zero done items remain.
7. **ANNOTATE — ~200 numbered items resolved inline across all 10 files**,
   every one checked (zero skipped): verdicts are `done at <hash>` /
   `Won't implement — reason` / `NOT-DO` / `still open`, citing real
   commits, code paths, or CI observations. Tables got Status columns
   (Pattern B); prose lists got full-line strikethroughs.
8. **ARCHIVE**: 8 fully-resolved files `git mv`'d to
   `docs/{status,planning,reviews}/archived/`. The 2 freshest (00-40
   report, 23-09 plan) annotated in place — they still carry open items.
9. **Source truth-fix**: the misleading `extractJSONObject` doc comment
   (00-40 report e/6 — the "braces in strings are safe" claim is wrong for
   noise-after-JSON) corrected in `discover.go`; limitation now stated
   truthfully. Tests untouched (behavior unchanged, gate green).
10. **RELEASING.md precondition 4 added**: CI green on ALL matrix legs on
    origin before tagging — the v0.2.0 poisoned-tag lesson, encoded where
    the next release will actually read it.
11. **HTML artifacts**: placeholder `<title>Report Title</title>` replaced
    with real titles in both; all four "Ticketed" verdicts resolved inline
    (+ resolved table badges); resolution appendix comments added.
12. **Dangling reference repaired**: the 23-09 plan pointed at the 23-04
    report's pre-archive path — updated to `archived/`.
13. **Canonical gate green with `set -o pipefail`** (build, vet,
    race+shuffle, lint, flake check, actionlint) — run BEFORE the health
    report was claimed, per the verify-then-annotate rule.
14. **Health report printed inline** (not written to a file, per skill):
    Accuracy 7.25/10 and Fitness 7.30/10, all findings fixed on sight.

## b) PARTIALLY DONE

_Resolved 2026-09-08: #1/#2 closed post-hoc same day; #3 shipped at
`473f321`; #4 remains a deliberate provenance choice._

1. ~~**HTML structural re-validation**~~ done — the tag-balance
   validation ran post-hoc while writing the report: both VALID
2. ~~**Real-DB smoke test after the `discover.go` edit**~~ done — run
   post-hoc, PASS
3. ~~**TODO_LIST routing** — 17 items in place, but they carry no stable
   IDs~~ done at `473f321` — stable `T1…` IDs + citation convention
4. **22-44 archived file's original banner** still reads
   "[ARCHIVED 2026-08-15]" (historically true for the 23:04 closure note)
   while the file moved to `archived/` on 08-16 — the move is recorded only
   in the resolution appendix. Split provenance, deliberately left (history
   preserved), but it should have been one conscious choice documented in
   one place. _(verdict: deliberate leave — history preserved; documented
   here as that one place)_

## c) NOT STARTED

_Resolved 2026-09-08: #1 and #2 shipped same day; #3 superseded; #4
routed to the live TODO_LIST._

1. ~~Docs link/citation checking automation (`scripts/check-doc-links.sh`)~~
   done at `04901d6` — written, self-tested, wired into gate + CI
2. ~~FEATURES.md header still carries the temporal line "Generated
   2026-08-15 by a docs BUILD pass"~~ done at `0927fec` — dropped
3. ~~Push of the ahead-1 commit (`7733d36`) and of this session's diff~~
   superseded — pushes landed via the user's flow; master == origin
   held thereafter
4. External/schedule-gated items (unchanged, live in TODO_LIST): Renovate
   app install, first nightly fuzz observation, first monthly flake-update
   PR, pkg.go.dev v0.2.0 crawl, gosec G701 upstream repro. ← all since
   resolved or re-routed: fuzz observed green, pkg.go.dev crawled, gosec
   data point posted (#1712); Renovate still open (T1); flake-update's
   first run failed and was fixed 2026-09-08 (T3)

## d) TOTALLY FUCKED UP

1. **Seven wasted edit round-trips from exact-match sloppiness**: an edit
   attempted before Viewing the file (tool refused — system working, my
   fault), a FEATURES multiedit built from _rendered_ `||` pipes instead of
   raw `|`, a guessed table cell ("Todos untyped string" vs the actual
   "untyped JSON string"), a RELEASING edit that swallowed a blank line
   between numbered items (caught on read-back), and a wrong grep pattern
   when counting my own resolution markers (`| done \` vs`done at \`).
   All self-caught, zero repo damage — all preventable by View-before-edit,
   every single time.
2. **Wrote a rule, then immediately skirted it**: I widened the AGENTS
   real-DB smoke rule to "after ANY source change" and then did not run the
   smoke test after editing `discover.go`, rationalizing "comment-only".
   The rationalization cost more deliberation than the 0.084s test. Caught
   while writing this report; closed post-hoc (PASS).
3. **Todo-list tool left stale**: the session tracker still showed
   annotate/archive/gate as pending after they were finished — the same
   lie-class as docs drift (claiming a state that isn't), just in tool
   state instead of a file.
4. **Annotated HTML without validating**: I treated C18's structural HTML
   validation as a closed precedent, then edited two HTML files and never
   re-ran that validation on my own edits until this report forced the
   question. Precedent is not a substitute for re-verification after _my_
   changes.

## e) WHAT WE SHOULD IMPROVE

1. **Mechanize the docs checks**: a `scripts/check-doc-links.sh` (markdown
   links resolve; `file:line` citations point at real files) wired into the
   gate would have caught the broken `[0.2.0]` ref and the stale FEATURES
   citations for free — no session grep-heroics required.
2. **View-before-edit is non-negotiable**: every wasted round-trip in d/1
   was an "I knew what it looked like" edit. The discipline exists; apply
   it when tired too.
3. **Verify-then-annotate applies to MY OWN session claims**, not just
   historical docs: d/2 and d/4 are both instances of claiming
   completeness one step before the last verification existed.
4. **Stable IDs for TODO_LIST items** (the old M/m scheme had this right):
   annotations should cite "TODO_LIST item N", not "TODO_LIST".
5. **One canonical place for archival provenance** — banner or appendix,
   not both, when a historical file is annotated AND moved.
6. **Cadence for docs-health**: this much drift accumulated in ONE day of
   intense sessions. A docs-health audit gated on every release (and after
   any 50+ item session) keeps it bounded.

## f) Up to 50 things we should get done next

TODO_LIST.md is the canonical list (17 open items, verified this session).
Below are THIS session's genuinely new or re-ranked items.

_Resolved 2026-09-08 (docs-health annotate pass): every item verdict'd
inline; all shipped, closed, or re-routed to the live TODO_LIST._

| #  | Task                                                                                      | Size    | Resolution (2026-09-08)                                        |
| -- | ----------------------------------------------------------------------------------------- | ------- | --------------------------------------------------------------- |
| 1  | ~~Clean the temporal "Generated 2026-08-15" header line from FEATURES.md~~                | 2m      | Done at `0927fec`                                               |
| 2  | ~~`scripts/check-doc-links.sh` (links + `file:line` citations resolve) + wire into the gate~~ | 30m  | Done at `04901d6`                                               |
| 3  | ~~Give TODO_LIST items stable IDs; annotations cite them~~                                | 15m     | Done at `473f321`                                               |
| 4  | ~~Rename `docs/benchmarks/baseline-benchmark-sessions.txt` (holds 3 benchmarks now)~~     | 5m      | Done at `473f321`                                               |
| 5  | ~~**Cut v0.2.1** (standing user decision — see g/2) incl. v0.2.0-Windows errata note~~    | 30m     | Done at `7ff9e72` (tag), release + erratum verified             |
| 6  | ~~Observe first nightly fuzz run (03:17 UTC) — TODO_LIST external~~                       | 5m      | Done — green continuously since 2026-08-17 (observed 2026-09-08) |
| 7  | Observe first monthly flake-update PR — TODO_LIST external                                | 5m      | Changed — first run FAILED 2026-09-01 (bot push 403); permissions fixed 2026-09-08, re-observation = TODO_LIST T3 |
| 8  | Install Renovate app — TODO_LIST external (GitHub App UI)                                 | 5m      | Still open — TODO_LIST T1                                       |
| 9  | ~~Verify pkg.go.dev crawled v0.2.0 (still v0.1.1 at audit time) — TODO_LIST external~~    | 5m      | Done — crawl landed; page now renders through v0.3.0            |
| 10 | ~~`TestParseProjectsOutput`: `}` in noise AFTER JSON — TODO_LIST High~~                   | 10m     | Done at `0927fec`                                               |
| 11 | ~~`TestQuoteJSON` backslash-escape pin — TODO_LIST High~~                                 | 10m     | Done at `0927fec` (renamed `TestJSONString` at `63ad9a7`)        |
| 12 | ~~Cross-platform fakeCLI (Go-compiled helper, no `/bin/sh`) — TODO_LIST High~~            | 45m     | Done at `697b337` (ran on the real windows runner)              |
| 13 | ~~Platform-assumption audit of remaining tests — TODO_LIST High~~                         | 20m     | Done at `63ad9a7` (GOOS guards with reasons)                    |
| 14 | ~~CI: `-count=2` + `go mod verify` steps — TODO_LIST Medium~~                             | 7m      | Done at `d2d4634`                                               |
| 15 | ~~Release `workflow_dispatch` dry-run trigger — TODO_LIST Medium~~                        | 5m      | Done — trigger added; dry-run exercised (run 31919208017)        |
| 16 | ~~gosec G701 upstream repro — TODO_LIST external~~                                        | 30m     | Done — data point posted on securego/gosec#1712 (no duplicate filed) |
| 17 | Fuzz corpus mining once nightly artifacts exist — TODO_LIST ongoing                       | ongoing | Still open — TODO_LIST T6 (artifacts now exist)                  |
| 18 | CI status-check requirement on master (branch verified unprotected) — TODO_LIST external  | 5m      | **Won't implement — recorded ROADMAP non-decision "No branch protection on master"** |

(18 real items — stop at real value; the remainder of the backlog is
unchanged and lives in TODO_LIST.md.)

## g) Questions I cannot figure out myself (max 3)

_Resolved 2026-09-08: all three settled by the 04:20 release session._

1. ~~**Push policy right now**~~ superseded — pushes landed via the
   user's flow; master == origin held thereafter.
2. ~~**v0.2.1 — cut it?**~~ done at `7ff9e72` — cut with all four CI
   jobs green on origin before tagging (RELEASING precondition 4).
3. ~~**v0.2.1 CHANGELOG scope**~~ resolved — fixes-only + errata note,
   as recommended; verified by diffing `v0.2.0..HEAD` (only non-test Go
   change was a comment correction).

---

_Point-in-time snapshot. Living work items live in TODO_LIST.md. Generated
by the 2026-08-16 docs-health audit session. Left uncommitted per daemon
policy (no user commit instruction)._
