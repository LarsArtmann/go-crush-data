package crushdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// FuzzDecodeParts verifies the parts decoder never panics on arbitrary
// input: malformed, truncated, and adversarial JSON must all return normally.
func FuzzDecodeParts(f *testing.F) {
	f.Add(`[{"type":"text","data":{"text":"hello"}}]`)
	f.Add(`[{"type":"tool_call","data":{"id":"c1","name":"read","input":"{}"}}]`)
	f.Add(`[]`)
	f.Add(`malformed`)
	f.Add(`{"not":"array"}`)
	f.Add(`[{"type":"tool_result","data":{"tool_call_id":"x","content":null}}]`)
	f.Add(`[{"type":"finish","data":{"reason":"stop","time":0}}]`)
	f.Add(`[{"type":"`)

	f.Fuzz(func(t *testing.T, raw string) {
		_, _ = DecodeParts(raw)
	})
}

// FuzzDecodeTodos verifies the todos decoder never panics on arbitrary
// input and that a successful decode is internally consistent: every item
// decodes again from its own re-marshalled JSON.
func FuzzDecodeTodos(f *testing.F) {
	f.Add(`[{"content":"ship it","status":"completed","active_form":"Shipping it"}]`)
	f.Add(`[{"content":"?","status":"in_progress"}]`)
	f.Add(`[]`)
	f.Add(`null`)
	f.Add(`[1]`)
	f.Add(`{"not":"array"}`)
	f.Add(`[{"status":"drifted"}]`)
	f.Add(`truncated`)

	f.Fuzz(func(t *testing.T, raw string) {
		todos, err := DecodeTodos([]byte(raw))
		if err != nil {
			return
		}

		for i, todo := range todos {
			again, err := DecodeTodos(mustMarshalTodos(t, todo))
			if err != nil || len(again) != 1 || again[0] != todo {
				t.Fatalf("todo %d does not roundtrip: %+v -> %v, %v", i, todo, again, err)
			}
		}
	})
}

func mustMarshalTodos(t *testing.T, todo Todo) []byte {
	t.Helper()

	encoded, err := json.Marshal([]Todo{todo})
	if err != nil {
		t.Fatal(err)
	}

	return encoded
}

// FuzzParseProjectsOutput verifies the CLI-output parser never panics on
// arbitrary input, including payloads with log noise around the JSON (the
// extraction path must stay robust, not just the clean-JSON path).
func FuzzParseProjectsOutput(f *testing.F) {
	f.Add(`{"projects":[]}`)
	f.Add(`INFO noise {"projects":[{"path":"/p","data_dir":"/d","last_accessed":"x"}]} WARN`)
	f.Add(`null`)
	f.Add(`no braces here`)
	f.Add(`{"projects":`)

	f.Fuzz(func(t *testing.T, raw string) {
		projects, err := ParseProjectsOutput([]byte(raw))
		if err == nil && projects == nil && len(raw) > 0 {
			t.Fatalf("ParseProjectsOutput(%q) = nil slice with nil error on non-empty input", raw)
		}
	})
}

// FuzzLoadRegistry verifies the registry parser never panics on arbitrary
// file contents: the projects.json shape must be tolerated for any byte
// sequence without crashing.
func FuzzLoadRegistry(f *testing.F) {
	f.Add(`{"projects":[]}`)
	f.Add(`{"projects":[{"path":"/p","data_dir":"/d","last_accessed":"2026-01-01T00:00:00Z"}]}`)
	f.Add(`null`)
	f.Add(`malformed`)
	f.Add(`{"projects":[{"path":"","data_dir":"","last_accessed":""}]}`)
	f.Add(`{"projects":[null]}`)

	f.Fuzz(func(t *testing.T, raw string) {
		dir := t.TempDir()

		if err := os.WriteFile(filepath.Join(dir, RegistryName), []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}

		_, _ = loadRegistry(dir)
	})
}
