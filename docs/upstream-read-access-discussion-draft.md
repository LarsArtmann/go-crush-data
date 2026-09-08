# Draft: upstream discussion (charmbracelet/crush, category: Ideas)

Status: DRAFT — verified against crush v0.92.0 (559ec80) on 2026-09-07.
Do not post before final review.

- Title: Supported read access to historical session data for third-party tools
- Category: Ideas (upstream convention: feature requests go to Discussions,
  not Issues — see maintainer redirect in #1765)
- Cross-references: #3531 (live observability), #1765 (server mode), #2707
  (hooks, shipped), #1965 (export), discussion #3697 (deja-vu)

---

## Body

### What I'm asking about

A supported, documented way for third-party tools to read **historical**
session data (workspaces, sessions, messages, todos, cost stats). Mainly
three questions at the bottom — the server API looks like it already covers
most of this, so this is partly "please consider analytics consumers in the
upcoming docs/release" and partly "what should on-disk readers do meanwhile".

### Context: the read-only consumer ecosystem

There is a real ecosystem of open-source tools that read Crush's local
data — Go, Rust, TypeScript; all currently built on the
undocumented on-disk format (the Go six reviewed source-level on
2026-09-07, see
[the review](https://github.com/LarsArtmann/go-crush-data/blob/master/docs/ecosystem-implementation-review.md)):

| Tool                                                                                                                     | Lang       | What it does with the data                                                                                                                                                        |
| ------------------------------------------------------------------------------------------------------------------------ | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [junhoyeo/tokscale](https://github.com/junhoyeo/tokscale)                                                                | Rust       | token-usage tracking from per-project `crush.db`                                                                                                                                  |
| [jhlee0409/claude-code-history-viewer](https://github.com/jhlee0409/claude-code-history-viewer)                          | Rust/Tauri | GUI history browser across `<project>/.crush/crush.db`                                                                                                                            |
| [yigitkonur/cli-continues](https://github.com/yigitkonur/cli-continues)                                                  | TS         | session handoff parser; maintains its own reverse-engineered docs of the format                                                                                                   |
| [Dicklesworthstone/coding_agent_session_search](https://github.com/Dicklesworthstone/coding_agent_session_search) (CASS) | Rust       | cross-agent session search over `crush.db`                                                                                                                                        |
| [janekbaraniewski/openusage](https://github.com/janekbaraniewski/openusage)                                              | Go         | usage tracking from `.crush/crush.db` per project                                                                                                                                 |
| [Pilan-AI/mnemo](https://github.com/Pilan-AI/mnemo)                                                                      | Go         | indexes Crush sessions into its SQLite                                                                                                                                            |
| [superbasedapp/observer](https://github.com/superbasedapp/observer)                                                      | Go         | agent observer via a `crush` adapter (deepest parser in the ecosystem)                                                                                                            |
| [taigrr/crunch](https://github.com/taigrr/crunch)                                                                        | Go         | LLM-generated daily summaries from `crush.db` scans                                                                                                                               |
| [soyomarvaldezg/crush-tmux](https://github.com/soyomarvaldezg/crush-tmux)                                                | Go         | tmux status via read-only `crush.db` probing (#3531)                                                                                                                              |
| [LarsArtmann/go-crush-data](https://github.com/LarsArtmann/go-crush-data)                                                | Go         | typed read-only library over `projects.json` + `crush.db` (mine; verified against v0.92.0 / 559ec80), plus crush-daily (private, mine) building daily per-project summaries on it |

Adjacent, not on-disk readers: [perplexityai/numbat](https://github.com/perplexityai/numbat)
integrates via the hooks surface; vshulcz/deja-vu has a Crush parser
proposed (vshulcz/deja-vu#2949) but not merged on main.

When even motivated third parties get misled — deja-vu's Crush parser
notes: "It keeps sessions in SQLite, not in the JSON state file the
README talks about" — that's a docs gap, not a tooling gap.

What these tools all currently reverse-engineer (no upstream docs):

- `<global>/projects.json` registry shape and location resolution
- `crush.db`: `sessions` / `messages` / `read_files` tables (goose
  migrations evolve these — fine, but changes are invisible to readers)
- the message parts envelope (`[{"type":…, "data":{…}}]`, 8 discriminators)
- agent child session IDs (`messageID$$toolCallID`)
- unix-second timestamps (the migration comments say milliseconds — they
  are not; comment fix pending in #3576). The comment has real victims:
  2 of the 6 Go readers ship date bugs from converting with
  `time.UnixMilli` — their sessions land in January 1970

This works, but breaks silently on migrations. For example, #3580
(compressing message parts) would break every JSON-parsing reader. As #2707's
author put it: "Schema isn't a public API, breaks on migrations." We know —
that's exactly why we're asking.

### What I found in the repo (v0.92.0)

The server API already looks like the answer for most of this.
`internal/swagger/swagger.json` includes:

- `GET /workspaces` — would replace `projects.json` discovery
- `GET /workspaces/{id}/sessions` + `/sessions/{sid}` — `proto.Session`
  even carries `parent_session_id` (agent subtrees), `todos`, `cost`,
  token counts, timestamps
- `GET /workspaces/{id}/sessions/{sid}/messages` (+ `/history`)

and per @meowgorithm in #3531: "Server-client is still not officially
released, but usable and hopefully not too far off. Part of that work will
definitely be docs."

### Questions

1. **Enumerating historical projects.** `Backend.ListWorkspaces` returns
   the running server's in-memory workspaces (populated via
   `CreateWorkspace`; I couldn't find hydration from the `projects.json`
   registry at startup). For analytics tools the make-or-break operation is
   "enumerate every project I have ever used, offline ones included, and
   read its full history." Could `GET /workspaces` (or a sibling endpoint)
   cover the registry, or is that intentionally out of scope?
2. **Stability expectations for the HTTP API.** Once documented, do you
   intend to treat it as a compatibility surface (semver-ish pinning), or
   best-effort/may-change for a while? Even a rough answer lets downstream
   tools decide how much to lean on it versus the DB.
3. **On-disk readers meanwhile.** Would you be open to blessing the goose
   migrations under `internal/db/migrations` as the de-facto changelog for
   read-only consumers ("watch this dir"), or a short docs note stating the
   on-disk format is unsupported and will keep changing? Either is cheap
   and removes the guesswork.

Happy to share what we learned from real-world data (e.g. a census of the
todos JSON shape across 71,747 items in 287 databases — zero malformed,
three statuses, no extra keys) if that's useful for the docs work.

### Related

- #3531 — live observability surface (complementary; this post is about
  historical reads)
- #1765 — server mode (completed)
- #2707 — lifecycle hooks (shipped)
- #1965 — built-in export (adjacent, user-facing)
