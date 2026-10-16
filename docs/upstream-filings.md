# Upstream filings ledger

Local record of everything this project (or its retired TODO items T9/T20)
has filed on other repositories, so retired TODO entries keep a trace of
what was filed where and why. Filed things also live in the target repos;
this file is our own bookkeeping. Update it on every filing and on every
state change observed via `scripts/check-upstream-status.sh`.

Campaign context: [status/2026-09-08_filing-campaign-plan.md](status/2026-09-08_filing-campaign-plan.md).

| When       | Where                          | Kind               | Subject / link                                                                                                          | State (last observed)                                                                                                                                                                        |
| ---------- | ------------------------------ | ------------------ | ----------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-08-16 | stump-wtf/agent-trace#22       | Issue (proposal)   | [adopt go-crush-data as the Crush adapter's storage layer](https://github.com/stump-wtf/agent-trace/issues/22)          | open; maintainer verified 2 of 3 findings and fixed them there (`9726ecb` registry dedupe, `5d5023c` rowid cursor), declined adoption, and left the door open — see the follow-up note below |
| 2026-09-07 | stump-wtf/agent-trace#22       | Comment            | rowid ordering landed on this side too (`13bbad8`) + independent 52,569-message inversion replication                   | posted; thread's last human word is `@joestump`'s "will take a look next week" 2026-09-11                                                                                                    |
| 2026-09-08 | charmbracelet/crush#3740       | Discussion (Ideas) | [Supported read access to historical session data](https://github.com/charmbracelet/crush/discussions/3740)             | live; 1 substantive third-party comment 2026-09-08                                                                                                                                           |
| 2026-09-08 | janekbaraniewski/openusage#357 | Fix PR             | [read session timestamps as Unix seconds](https://github.com/janekbaraniewski/openusage/pull/357) (retired TODO T9)     | open                                                                                                                                                                                         |
| 2026-09-08 | Pilan-AI/mnemo#22              | Fix PR             | [read crush message timestamps as Unix seconds](https://github.com/Pilan-AI/mnemo/pull/22) (retired TODO T9)            | open                                                                                                                                                                                         |
| 2026-09-08 | charmbracelet/crush#3576       | Fix PR (ours)      | [fix timestamp unit comments in initial migration](https://github.com/charmbracelet/crush/pull/3576) (retired TODO T20) | **merged** 2026-09-08 (v0.92.0 was cut 2026-08-31, so the corrected comments ship in the next release)                                                                                       |
| 2026-09-08 | charmbracelet/crush#3580       | Issue (ours)       | [Compress message parts — ecosystem reader inventory](https://github.com/charmbracelet/crush/issues/3580)               | open                                                                                                                                                                                         |
| 2026-09-08 | charmbracelet/crush#3581       | PR (ours)          | [compress message parts with zstd](https://github.com/charmbracelet/crush/pull/3581)                                    | open                                                                                                                                                                                         |
| 2026-09-08 | vshulcz/deja-vu#2949           | Comment            | verified format facts + pointer to go-crush-data                                                                        | posted (thread already closed-completed)                                                                                                                                                     |
| 2026-09-08 | taigrr/crunch#22               | Issue              | registry discovery suggestion (Phase G3)                                                                                | posted                                                                                                                                                                                       |

Retired-TODO mapping: **T9** (file ecosystem date-bug fixes) was closed by
the openusage#357 + mnemo#22 PRs; **T20** (fix the upstream migration
comment) was closed by crush#3576. Phase G1/G2 adoption suggestions remain
drafted in the campaign plan, gated on the two fix PRs merging — daily
state via `scripts/check-upstream-status.sh` (TODO_LIST T35).

agent-trace#22 follow-up (state 2026-10-15): the maintainer's stated
precondition for reconsidering adoption — "genuinely open to revisiting if
`MessagesSince` lands" — is now met on this side; `MessagesSince` /
`IterMessagesSince` / `LastMessageRowID` / `MessageRowID` shipped in
CHANGELOG `[Unreleased]` (TODO_LIST T36 tracks re-opening the conversation,
and the thread's mirror is bot-managed since 2026-10-13, so any comment goes
to the canonical tracker). Two items from that thread remain open and are
NOT filed:

- the invitation to patch the adapter's silent `continue` — surface
  unreadable databases through its existing `Diagnostics()` instead of
  failing the whole listing ("Happy to take a patch for it if you're
  interested");
- the adoption revisit itself, gated on that repo's priorities (their
  `@joestump` parked it 2026-09-11: "getting back to Harness and skill
  distillation next week").

Process note: all future filings follow the AGENTS.md external-filing
checklist (verify-then-file; no forward promises in bodies; edit-after-
posting; re-verify cross-claims in the final body).
