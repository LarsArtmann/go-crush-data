# go-crush-data

Typed, read-only Go access to [Crush](https://github.com/charmbracelet/crush)
local session data: project discovery, sessions, messages (with decoded
parts), subagent graphs, and daily usage statistics.

[![CI](https://github.com/LarsArtmann/go-crush-data/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/go-crush-data/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/LarsArtmann/go-crush-data.svg)](https://pkg.go.dev/github.com/LarsArtmann/go-crush-data)
![Coverage](https://img.shields.io/badge/coverage-%E2%89%A585%25%20enforced-success)
[![Go Report Card](https://goreportcard.com/badge/github.com/LarsArtmann/go-crush-data)](https://goreportcard.com/report/github.com/LarsArtmann/go-crush-data)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Why

Crush ships **no SDK and no documented schema**. It stores sessions in the global project registry
(`~/.local/share/crush/projects.json` on Unix, `%LOCALAPPDATA%\crush\projects.json`
on Windows) and one SQLite database (`crush.db`) per project data directory. The same reading logic has
been reverse-engineered independently three times — and the schema
**drifts across Crush versions** (columns like `sessions.cost`,
`messages.model`, and `sessions.parent_session_id` were added in later
migrations).

This library centralizes that knowledge once, with the drift defense built
in: every `Open` probes the schema and every read substitutes zero values for
absent columns instead of failing. As a public repo it doubles as a
community drift sentinel — when Crush changes its schema, PRs land here.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"time"

	crushdata "github.com/LarsArtmann/go-crush-data"
)

func main() {
	ctx := context.Background()

	// Discovery: registry first, `crush projects --json` as fallback.
	projects, err := crushdata.DiscoverProjects(ctx, crushdata.DiscoverOptions{CLIFallback: true})
	if err != nil {
		panic(err)
	}

	for _, project := range projects {
		// Read-only open — safe alongside a running Crush.
		db, err := crushdata.Open(project.DataDir)
		if err != nil {
			continue
		}

		// Day filters use the filter value's own location: time.Now()
		// draws the day boundary in local time (see Timestamps below).
		yesterday := time.Now().AddDate(0, 0, -1)

		sessions, _ := db.Sessions(ctx, crushdata.SessionFilter{Day: yesterday, RootOnly: true})
		stats, _ := db.Stats(ctx, crushdata.StatsFilter{Day: yesterday})

		fmt.Printf("%s: %d sessions, %d tokens, $%.4f\n",
			project.Path, len(sessions), stats.PromptTokens+stats.CompletionTokens, stats.CostUSD)

		_ = db.Close()
	}
}
```

Messages decode their `parts` JSON into a sealed type — type-switch over
`crushdata.TextPart`, `ToolCallPart`, `ToolResultPart`, `ReasoningPart`,
`FinishPart`, `ShellCommandPart`, and `UnknownPart` (forward compatibility:
parts Crush invents tomorrow pass through instead of failing):

```go
messages, _ := db.Messages(ctx, sessionID)
for _, message := range messages {
	for _, part := range message.Parts {
		if call, ok := part.(crushdata.ToolCallPart); ok {
			fmt.Println("tool call:", call.Name, call.Input)
		}
	}
}
```

Subagent trees come from the `parent_session_id` column:

```go
graph, _ := db.AgentGraph(ctx, rootSessionID)
for _, node := range graph.Nodes {
	fmt.Printf("%sdepth %d: %s\n", strings.Repeat("  ", node.Depth), node.Depth, node.Session.Title)
}
```

## Schema drift

| Capability | Column | Missing on old databases |
|---|---|---|
| Cost | `sessions.cost` | `CostUSD` reads 0 |
| Subagents | `sessions.parent_session_id` | `AgentGraph` returns the root only |
| Models | `messages.model` | model fields read empty |
| Providers | `messages.provider` | provider fields read empty |
| Finish times | `messages.finished_at` | `FinishedAt` reads zero |

`db.Schema().MissingColumns()` tells you exactly what an old database lacks.
Databases without the required `sessions`/`messages` tables fail `Open` with
`ErrUnsupportedSchema`.

## Design

- **Read-only by construction**: SQLite `mode=ro`, single connection, never
  writes — safe to run while Crush is open.
- **Errors as values**: sentinel errors (`ErrRegistryNotFound`,
  `ErrDatabaseNotFound`, `ErrUnsupportedSchema`, `ErrSessionNotFound`) testable
  with `errors.Is`.
- **Sealed `Part` interface**: the part set is closed; unknown parts pass
  through with their raw payload.
- **Zero dependencies** beyond `modernc.org/sqlite` (pure Go, no CGO).
- **Timestamps**: Crush stores Unix seconds; this library returns
  `time.Time` in UTC. Day filters (`SessionFilter.Day`, `StatsFilter.Day`)
  match the calendar day in the filter value's own location; pass local
  midnight to bucket sessions by your local day.

## Install

```
go get github.com/LarsArtmann/go-crush-data
```

Requires Go 1.26 or newer.

## Development

```
nix develop       # dev shell with Go, golangci-lint, govulncheck
nix run .#lint    # golangci-lint (~90 linters, see .golangci.yml)
go test ./...     # full suite (fixture databases are generated per-test)
nix flake check   # build + format checks
```

See also: [FEATURES.md](FEATURES.md) (honest feature inventory),
[ROADMAP.md](ROADMAP.md) (direction and recorded non-decisions),
[CONTRIBUTING.md](CONTRIBUTING.md) (parity contract, benchmarks),
[RELEASING.md](RELEASING.md) (release procedure and tag integrity).

## License

[MIT](LICENSE)
