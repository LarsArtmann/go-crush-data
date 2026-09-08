# Upstream filings ledger

Local record of everything this project (or its retired TODO items T9/T20)
has filed on other repositories, so retired TODO entries keep a trace of
what was filed where and why. Filed things also live in the target repos;
this file is our own bookkeeping. Update it on every filing and on every
state change observed via `scripts/check-upstream-status.sh`.

Campaign context: [status/2026-09-08_filing-campaign-plan.md](status/2026-09-08_filing-campaign-plan.md).

| When        | Where                                              | Kind              | Subject / link                                                                                              | State (last observed)                          |
| ----------- | -------------------------------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| 2026-09-08  | charmbracelet/crush#3740                           | Discussion (Ideas) | [Supported read access to historical session data](https://github.com/charmbracelet/crush/discussions/3740) | live; 1 substantive third-party comment 2026-09-08 |
| 2026-09-08  | janekbaraniewski/openusage#357                     | Fix PR            | [read session timestamps as Unix seconds](https://github.com/janekbaraniewski/openusage/pull/357) (retired TODO T9) | open                                            |
| 2026-09-08  | Pilan-AI/mnemo#22                                  | Fix PR            | [read crush message timestamps as Unix seconds](https://github.com/Pilan-AI/mnemo/pull/22) (retired TODO T9) | open                                            |
| 2026-09-08  | charmbracelet/crush#3576                           | Fix PR (ours)     | [fix timestamp unit comments in initial migration](https://github.com/charmbracelet/crush/pull/3576) (retired TODO T20) | **merged** 2026-09-08, before the v0.92.0 pin    |
| 2026-09-08  | charmbracelet/crush#3580                           | Issue (ours)      | [Compress message parts — ecosystem reader inventory](https://github.com/charmbracelet/crush/issues/3580)   | open                                            |
| 2026-09-08  | charmbracelet/crush#3581                           | PR (ours)         | [compress message parts with zstd](https://github.com/charmbracelet/crush/pull/3581)                       | open                                            |
| 2026-09-08  | vshulcz/deja-vu#2949                               | Comment           | verified format facts + pointer to go-crush-data                                                           | posted (thread already closed-completed)        |
| 2026-09-08  | taigrr/crunch#22                                   | Issue             | registry discovery suggestion (Phase G3)                                                                   | posted                                          |

Retired-TODO mapping: **T9** (file ecosystem date-bug fixes) was closed by
the openusage#357 + mnemo#22 PRs; **T20** (fix the upstream migration
comment) was closed by crush#3576. Phase G1/G2 adoption suggestions remain
drafted in the campaign plan, gated on the two fix PRs merging — daily
state via `scripts/check-upstream-status.sh` (TODO_LIST T35).

Process note: all future filings follow the AGENTS.md external-filing
checklist (verify-then-file; no forward promises in bodies; edit-after-
posting; re-verify cross-claims in the final body).
