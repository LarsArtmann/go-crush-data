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

There is a small but real ecosystem of tools that read Crush's local data
for analytics/search, all currently built on the undocumented on-disk
format:

- [go-crush-data](https://github.com/LarsArtmann/go-crush-data) — typed
  read-only Go library over `projects.json` + `crush.db` (mine; verified
  against v0.92.0 / 559ec80)
- crush-daily — daily per-project stats (uses go-crush-data)
- [deja-vu](https://github.com/vshulcz/deja-vu) session search
  (vshulcz/deja-vu#2949, see discussion #3697)
- crush-tmux — read-only SQLite watermark probing (see #3531)

What we all currently reverse-engineer (no upstream docs):

- `<global>/projects.json` registry shape and location resolution
- `crush.db`: `sessions` / `messages` / `read_files` tables (goose
  migrations evolve these — fine, but changes are invisible to readers)
- the message parts envelope (`[{"type":…, "data":{…}}]`, 8 discriminators)
- agent child session IDs (`messageID$$toolCallID`)
- unix-second timestamps (the migration comments say milliseconds — they
  are not 🙂)

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
