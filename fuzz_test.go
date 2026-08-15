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
		_, _ = DecodeParts(raw) //nolint:errcheck // the invariant is: no panic
	})
}
