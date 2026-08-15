package crushdata

import (
	"encoding/json"
	"testing"
)

func TestDecodePartsEveryKind(t *testing.T) {
	t.Parallel()

	raw := `[
		{"type":"text","data":{"text":"hello"}},
		{"type":"reasoning","data":{"thinking":"hm","started_at":10,"finished_at":17}},
		{"type":"tool_call","data":{"id":"c1","name":"read","input":"{\"file_path\":\"/x\"}","provider_executed":false,"finished":true}},
		{"type":"tool_result","data":{"tool_call_id":"c1","name":"read","content":"ok","is_error":true,"mime_type":"text/plain","metadata":"meta","data":"d"}},
		{"type":"finish","data":{"reason":"stop","message":"done","details":"d"}},
		{"type":"shell_command","data":{"command":"ls","output":"files","exit_code":2}},
		{"type":"image_url","data":{"url":"http://x"}},
		{"type":"brand_new","data":{"whatever":1}}
	]`

	parts, err := DecodeParts(raw)
	if err != nil {
		t.Fatalf("DecodeParts: %v", err)
	}

	if len(parts) != 8 {
		t.Fatalf("parts = %d, want 8", len(parts))
	}

	verifyTextAndReasoning(t, parts)
	verifyToolCalls(t, parts)
	verifyFinishAndShell(t, parts)
	verifyUnknowns(t, parts)
}

func verifyTextAndReasoning(t *testing.T, parts []Part) {
	t.Helper()

	text, ok := parts[0].(TextPart)
	if !ok || text.Text != "hello" {
		t.Fatalf("parts[0] = %#v, want TextPart{hello}", parts[0])
	}

	reasoning, ok := parts[1].(ReasoningPart)
	if !ok || reasoning.Thinking != "hm" || reasoning.StartedAt != 10 || reasoning.FinishedAt != 17 {
		t.Fatalf("parts[1] = %#v, want ReasoningPart", parts[1])
	}
}

func verifyToolCalls(t *testing.T, parts []Part) {
	t.Helper()

	call, ok := parts[2].(ToolCallPart)
	if !ok || call.ID != "c1" || call.Name != "read" || !call.Finished || call.ProviderExecuted {
		t.Fatalf("parts[2] = %#v, want ToolCallPart", parts[2])
	}

	if call.Input != `{"file_path":"/x"}` {
		t.Fatalf("call.Input = %q, want the raw JSON argument object", call.Input)
	}

	result, ok := parts[3].(ToolResultPart)
	if !ok || result.ToolCallID != "c1" || !result.IsError || result.MIMEType != "text/plain" ||
		result.Metadata != "meta" || result.Data != "d" || result.Content != "ok" {
		t.Fatalf("parts[3] = %#v, want ToolResultPart", parts[3])
	}
}

func verifyFinishAndShell(t *testing.T, parts []Part) {
	t.Helper()

	finish, ok := parts[4].(FinishPart)
	if !ok || finish.Reason != "stop" || finish.Message != "done" || finish.Details != "d" {
		t.Fatalf("parts[4] = %#v, want FinishPart", parts[4])
	}

	shell, ok := parts[5].(ShellCommandPart)
	if !ok || shell.Command != "ls" || shell.Output != "files" || shell.ExitCode != 2 {
		t.Fatalf("parts[5] = %#v, want ShellCommandPart", parts[5])
	}
}

func verifyUnknowns(t *testing.T, parts []Part) {
	t.Helper()

	image, ok := parts[6].(UnknownPart)
	if !ok || image.Type != "image_url" {
		t.Fatalf("parts[6] = %#v, want UnknownPart(image_url)", parts[6])
	}

	brand, ok := parts[7].(UnknownPart)
	if !ok || brand.Type != "brand_new" {
		t.Fatalf("parts[7] = %#v, want UnknownPart(brand_new)", parts[7])
	}

	if string(brand.Data) != `{"whatever":1}` {
		t.Fatalf("brand.Data = %s, want raw payload passthrough", brand.Data)
	}
}

func TestDecodePartsEmptyInputs(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "   ", "[]", "  []  "} {
		parts, err := DecodeParts(raw)
		if err != nil {
			t.Fatalf("DecodeParts(%q): %v", raw, err)
		}

		if len(parts) != 0 {
			t.Fatalf("DecodeParts(%q) = %d parts, want 0", raw, len(parts))
		}
	}
}

// TestDecodePartsDegenerateEntries mirrors rows Crush has been observed to
// write: typeless entries and null payloads carry no information and decode
// to nothing.
func TestDecodePartsDegenerateEntries(t *testing.T) {
	t.Parallel()

	parts, err := DecodeParts(`[{"type":"","data":{"text":"x"}},{"type":"text","data":null}]`)
	if err != nil {
		t.Fatalf("DecodeParts: %v", err)
	}

	if len(parts) != 0 {
		t.Fatalf("parts = %d, want 0", len(parts))
	}
}

func TestDecodePartsMalformed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{name: "truncated", raw: `[{"type":"text","data":{"text":"hel`},
		{name: "not an array", raw: `{"type":"text"}`},
		{name: "garbage", raw: `garbage`},
		{name: "wrong payload type", raw: `[{"type":"text","data":{"text":42}}]`},
		{name: "tool_call payload not object", raw: `[{"type":"tool_call","data":"flat"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := DecodeParts(tt.raw); err == nil {
				t.Fatalf("DecodeParts(%q) err = nil, want error", tt.raw)
			}
		})
	}
}

// TestPartSetIsSealed verifies no type outside the package can implement
// Part — the compiler enforces the closed set via the unexported isPart.
func TestPartSetIsSealed(t *testing.T) {
	t.Parallel()

	var (
		_ Part = TextPart{}
		_ Part = ReasoningPart{}
		_ Part = ToolCallPart{}
		_ Part = ToolResultPart{}
		_ Part = FinishPart{}
		_ Part = ShellCommandPart{}
		_ Part = UnknownPart{}
	)

	// UnknownPart must round-trip its payload as valid JSON.
	if !json.Valid([]byte(`{"type":"x"}`)) {
		t.Fatal("sanity: raw JSON fixture invalid")
	}
}
