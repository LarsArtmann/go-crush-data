# Ecosystem implementation review — Go tools that read Crush data

Source-level review of every Go project found reading Crush local data,
compared against go-crush-data. All findings verified against each repo's
source (shallow clones, 2026-09-07). Crush version referenced by all: the
on-disk format as of v0.92.0 (559ec80).

## Master comparison

| Dimension | go-crush-data (ours) | openusage | observer (superbased) | mnemo | crunch | crush-tmux | numbat |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Reads crush.db? | yes | yes | yes | yes | yes | yes | **no — hooks only** |
| Discovery | registry + `crush projects --json` CLI fallback + CRUSH_GLOBAL_DATA→XDG→LOCALAPPDATA; dedupe by DataDir | registry (XDG/LOCALAPPDATA, no CRUSH_GLOBAL_DATA, no CLI fallback); abs-path dedupe | registry + cross-mount (WSL `/mnt/*`) translation + macOS App Support extra candidate | hardcoded `~/.crush/crush.db` + project-dir scan | `filepath.WalkDir` filesystem scan | `<cwd>/.crush/crush.db` | n/a |
| DSN | `mode=ro&_txlock=immediate`, MaxOpenConns(1), modernc | `mode=ro&immutable=1`, MaxOpenConns(1), mattn | `mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(2000)`, modernc | `mode=ro&_busy_timeout=3000` | `file:` URI `mode=ro` | `mode=ro&_query_only=true` | n/a |
| Schema-drift handling | systematic pragma_table_info probing, ErrUnsupportedSchema vs probe-error split | probes 1 column (messages.provider) | none — hardcodes incl. newer `finished_at` | none — hardcodes parts/model/provider, per-row skip | none | none | n/a |
| Parts parsing | sealed Part types, 8 discriminators, UnknownPart tolerance, strict+tolerant modes | none (session aggregates + latest assistant model only) | deepest: text/reasoning/tool_call/tool_result pairing, reasoning state machine, tool-name taxonomy | text-only concat, skip-on-error | user-message text only | none | n/a |
| Timestamps | **seconds** ✓ (verified vs upstream source + real DBs) | **`time.UnixMilli` — BUG** (sessions land in 1970) | **seconds** ✓ (independently documented, live-verified 2026-07-09) | **`time.UnixMilli` — BUG** | **seconds** ✓ (`time.Unix`) | raw ints ✓ | n/a |
| Sessions model | filterable, agent subtrees (recursive CTE, depth 64), todos (census-pinned), day stats with parity contract, read_files | root-only, skips empty, double-count aware | root watermarks + session-cumulative tokens/cost | session stats + titles | day-range user messages | MAX(created_at) watermark only | n/a |
| Read pattern | batch queries, context-aware | batch | incremental `updated_at > ?` watermarks (daemon) | full re-index, mtime short-circuit | day-range batch | poll watermark | hooks events |
| Windows story | first-class (CI leg, path escaping tests) | LOCALAPPDATA handled | cross-mount aware | none | none | none | hooks config |

## Notable per-repo findings

### openusage (janekbaraniewski/openusage) — 2 bugs / good instincts
- **Timestamp bug**: `millisToTime` uses `time.UnixMilli`. Their comment
  cites `internal/db/migrations/20250424200609_initial.sql` as the schema
  reference — the exact migration whose comment lies about milliseconds.
  Real data proves seconds: `1784269138` → 2026-07-17 as seconds,
  1970-01-21 as millis (verified on local DBs). Every date bucket they
  derive is wrong.
- `immutable=1` DSN: avoids lock contention with a live writer at the
  cost of reading potentially stale/torn snapshots — deliberate, and
  commented. Different tradeoff than our `_txlock=immediate`.
- Only ecosystem project besides us that probes for a capability column
  (messages.provider, added by migration 20250627000000).
- Registry dedupe and relative `data_dir` handling are correct; misses
  CRUSH_GLOBAL_DATA and the CLI fallback.

### observer (superbasedapp/observer) — best-in-class
- Independently discovered and documented the seconds-vs-comment lie
  ("Timestamps are Unix SECONDS despite the schema comment claiming
  milliseconds — the update trigger writes strftime('%s','now')",
  live-verified 2026-07-09). Matches our AGENTS.md finding exactly.
- Only project handling WSL cross-mount path translation and a macOS
  Application Support discovery candidate.
- Deepest semantic layer: tool_call/tool_result pairing across messages,
  reasoning consumed-once/last-wins state machine, normalized tool
  taxonomy. Tied to their event model, not reusable as a library.
- Hardcodes `finished_at` (newer column) — would fail on older DBs; no
  probing. Watcher snapshot of projects.json needs restart/backfill for
  new projects.

### mnemo (Pilan-AI/mnemo) — 2 bugs / minimal
- **Same `time.UnixMilli` bug** as openusage.
- Primary path hardcoded to `~/.crush/crush.db` — the modern per-data-dir
  layout (and projects.json) is ignored for the main index.
- No schema probing; any upstream column rename breaks the whole indexer
  silently (per-row `continue` on scan/parse errors hides it).

### crunch (taigrr/crunch) — correct but shallow
- `time.Unix(createdAt, 0)` — correct seconds. Same use-case family as
  crush-daily (daily activity summaries).
- Discovers DBs by walking the filesystem (no registry knowledge needed,
  but O(tree) and scan-root dependent).
- User-message text only; no session metadata.

### crush-tmux (soyomarvaldezg/crush-tmux) — minimal by design
- Watermark reads only (`MAX(messages.created_at)`); raw integer math,
  so timestamp units never mattered. `mode=ro&_query_only=true`.

### numbat (perplexityai/numbat) — NOT a data reader
- Integrates via the supported hooks surface: installs hook refs into
  `crush.json`, preserving unknown fields on re-marshal (nice JSON
  tolerance). `.crush/crush.db` appears only as a documentation string.
- Reclassify: hooks consumer, not an on-disk consumer.

### deja-vu (vshulcz/deja-vu) — no Crush code on main
- The Crush parser exists only as issue #2949 (proposal/patch); `main`
  has zero crush references. Reclassify as "proposed, not shipped".

## Comparison verdicts vs go-crush-data

**Where we are uniquely ahead** (validated by this review):
1. Only systematic schema-capability probing with a strict
   error-semantics split (everyone else hardcodes or half-probes).
2. Only typed, tolerant parts surface (sealed Part interface +
   UnknownPart) usable as a library.
3. Only agent-subtree queries, todos decoding, day-filtered stats with a
   parity contract, read_files access.
4. Only discovery that is a strict superset: CRUSH_GLOBAL_DATA → XDG →
   LOCALAPPDATA (upstream's own order) + `crush projects --json` CLI
   fallback + DataDir dedupe.
5. First-class Windows support with CI.

**What others do that we don't (candidates to adopt):**
1. `busy_timeout` on the DSN (mnemo 3000ms, observer 2000ms) — our reads
   can fail immediately with SQLITE_BUSY if Crush holds a write lock.
2. `query_only` pragma as defense-in-depth (observer, crush-tmux) —
   mode=ro already blocks writes, but query_only also guards attached-DB
   and driver-level escape hatches.
3. Cross-mount path translation for WSL readers (observer) — niche, but
   real for WSL users whose projects live on /mnt/c.
4. Incremental `updated_at > ?` watermark reads (observer) — our library
   is batch-oriented by design; consumers could build this on top, but a
   documented watermark helper would serve daemon-style consumers.

## Timestamp-lie casualty count

2 of 6 on-disk readers (openusage, mnemo) inherited the millisecond
claim from the migration comment and shipped date bugs. Two others
(observer, us) independently discovered the truth; one (crush) got lucky
with `time.Unix`; one (crush-tmux) never converts units. This is the
strongest possible evidence for PR #3576 (fix the comment) and for the
upstream discussion draft: the format's only "documentation" is a
comment that lies, and half the ecosystem believed it.
