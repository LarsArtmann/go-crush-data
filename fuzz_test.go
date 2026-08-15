package crushdata

import "testing"

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
