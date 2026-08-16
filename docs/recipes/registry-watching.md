# Watching the project registry (live consumers)

Dashboards and daemons want to react when Crush learns a new project. This
library deliberately does not watch anything: watching needs fsnotify-class
dependencies, and this module's contract is zero weight beyond
`modernc.org/sqlite` (see the recorded non-decision in
[ROADMAP.md](../ROADMAP.md)). Watching belongs to the consumer, and the
composition is small: watch the **global data directory** (file watchers
watch directories, not single files), filter events down to
`projects.json`, debounce, then re-run `DiscoverProjects`.

## Recipe with go-filewatcher

[go-filewatcher](https://github.com/LarsArtmann/go-filewatcher) v2 wraps
fsnotify with recursion, filtering, and debouncing. Combined with this
library the live loop is:

```go
package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	crushdata "github.com/LarsArtmann/go-crush-data"
	filewatcher "github.com/larsartmann/go-filewatcher/v2"
)

func main() {
	globalDir := crushdata.GlobalDataDir()

	watcher, err := filewatcher.New(
		[]string{globalDir}, // the DIRECTORY holding the registry
		filewatcher.WithExtensions(".json"),
		filewatcher.WithDebounce(300*time.Millisecond), // Crush rewrites the file on every access
	)
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = watcher.Close() }()

	events, err := watcher.Watch(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for event := range events {
		if filepath.Base(event.Path) != crushdata.RegistryName {
			continue // some other .json in the global dir
		}

		// The registry changed: re-read it (one file read — cheap).
		projects, err := crushdata.DiscoverProjects(context.Background(), crushdata.DiscoverOptions{})
		if err != nil {
			log.Printf("re-discover: %v", err)

			continue
		}

		log.Printf("registry now lists %d projects", len(projects))
		// Open new data dirs with crushdata.Open(project.DataDir) and
		// Close() the ones that disappeared.
	}
}
```

## Semantics worth knowing

- **Registry events mean project-set changes, not data changes.** Crush
  rewrites `projects.json` when projects are accessed or registered; the
  session data itself lives in each `crush.db` and changes constantly while
  Crush runs. For fresh session data, re-query through an open
  `crushdata.Open` handle — reads are safe alongside a live Crush — or
  reopen the DB.
- **Discovery skips data dirs without a crush.db.** A freshly registered
  project appears only after its first session exists.
- **Re-discovery is the whole refresh.** `DiscoverProjects` reads one small
  JSON file; caching its result on top of the watcher adds staleness for
  no measurable gain.
- **Watch the directory, filter to the file.** fsnotify-class backends
  watch directories; go-filewatcher's extension filter plus the
  `filepath.Base` check keeps the noise out. Events are absolute paths.
- **Open handles are unaffected.** Registry events do not invalidate open
  `crushdata.Open` handles; each handle stays bound to its data dir.

## Verified

The composition above was executed end-to-end on 2026-08-16 against a
throwaway registry: baseline discovery returned 1 project, a registry
rewrite (the way Crush registers a new project) fired the debounced
projects.json event, and re-discovery returned 2 projects.
