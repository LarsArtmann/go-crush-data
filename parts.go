package crushdata

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Part is one entry of a message's parts array. The set of implementations
// is sealed: TextPart, ReasoningPart, ToolCallPart, ToolResultPart,
// FinishPart, ShellCommandPart, and UnknownPart. Match on the concrete type
// with a type switch; parts Crush adds in future releases decode as
// UnknownPart instead of failing.
//
// The on-disk shape is a discriminated union — `{"type": ..., "data": ...}`
// — mirroring charmbracelet/crush's internal message content parts.
type Part interface {
	isPart()
}

// TextPart carries human-readable message text.
type TextPart struct {
	Text string `json:"text"`
}

// ReasoningPart carries the model's chain-of-thought text plus its own
// start/finish timestamps (unix seconds; zero when the part omits them).
type ReasoningPart struct {
	Thinking   string `json:"thinking"`
	StartedAt  int64  `json:"started_at"`
	FinishedAt int64  `json:"finished_at"`
}

// ToolCallPart records the model requesting a tool. Input is the raw JSON
// argument object exactly as Crush stored it — decode it against the
// expected shape for the tool named by Name.
type ToolCallPart struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Input            string `json:"input"`
	ProviderExecuted bool   `json:"provider_executed"`
	Finished         bool   `json:"finished"`
}

// ToolResultPart records the outcome of a [ToolCallPart] with the same
// ToolCallID. IsError marks failed executions.
type ToolResultPart struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Data       string `json:"data"`
	MIMEType   string `json:"mime_type"`
	Metadata   string `json:"metadata"`
	IsError    bool   `json:"is_error"`
}

// FinishPart marks the end of a message. Reason "stop" on a user message
// marks a user turn boundary in Crush's protocol.
type FinishPart struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// ShellCommandPart records a bang-mode shell command the user ran.
type ShellCommandPart struct {
	Command  string `json:"command"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

// UnknownPart passes through a part this library does not decode: either a
// discriminator Crush added after this release, or — in [DB.Messages]'
// tolerant decode — a known part whose payload did not match its expected
// shape. The discriminator and raw payload are kept for inspection and
// forward compatibility.
type UnknownPart struct {
	Type string
	Data json.RawMessage
}

func (TextPart) isPart()         {}
func (ReasoningPart) isPart()    {}
func (ToolCallPart) isPart()     {}
func (ToolResultPart) isPart()   {}
func (FinishPart) isPart()       {}
func (ShellCommandPart) isPart() {}
func (UnknownPart) isPart()      {}

// The part discriminators Crush writes today
// (charmbracelet/crush internal/message/content.go).
const (
	partText         = "text"
	partReasoning    = "reasoning"
	partToolCall     = "tool_call"
	partToolResult   = "tool_result"
	partFinish       = "finish"
	partShellCommand = "shell_command"
	partImageURL     = "image_url"
	partBinary       = "binary"
)

// jsonNull is the JSON keyword Crush writes for a part payload that carries
// no data; comparing against the named constant keeps the keyword greppable
// instead of hiding it as a bare string literal.
const jsonNull = "null"

// rawPart is the on-disk shape of every parts array entry.
type rawPart struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// DecodeParts parses one message's parts JSON, preserving declaration
// order, and is the strict all-or-nothing decode: empty or "[]" input
// returns nil, while malformed JSON or a known part whose payload does not
// match its expected shape returns an error naming the part. [DB.Messages]
// instead keeps every well-formed sibling and degrades a single malformed
// entry to [UnknownPart]; see decodeParts for that tolerant mode.
func DecodeParts(raw string) ([]Part, error) {
	return decodeParts(raw, true)
}

// decodeParts implements both decode modes. Strict mode fails the whole
// message on the first malformed part. Tolerant mode — used by
// [DB.Messages] — substitutes UnknownPart carrying the type discriminator
// and raw payload, so one corrupted entry never hides its well-formed
// siblings. Input that is not parseable as a whole (not a JSON array) fails
// both modes: no entries can be recovered.
func decodeParts(raw string, strict bool) ([]Part, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" {
		return nil, nil
	}

	var rawParts []rawPart
	if err := json.Unmarshal([]byte(trimmed), &rawParts); err != nil {
		return nil, fmt.Errorf("decode parts: %w", err)
	}

	parts := make([]Part, 0, len(rawParts))
	for _, entry := range rawParts {
		// Entries with no type or a null payload carry no information —
		// Crush has been observed to write such rows.
		if entry.Type == "" || len(entry.Data) == 0 || string(entry.Data) == jsonNull {
			continue
		}

		part, err := decodePart(entry)
		if err != nil {
			if strict {
				return nil, err
			}

			part = UnknownPart(entry)
		}

		parts = append(parts, part)
	}

	return parts, nil
}

// decodePart converts one raw entry into its typed variant. The entry must
// carry a type and a non-null payload (see DecodeParts).
func decodePart(entry rawPart) (Part, error) {
	var (
		part Part
		err  error
	)

	switch entry.Type {
	case partText:
		part, err = decodePartData[TextPart](entry)
	case partReasoning:
		part, err = decodePartData[ReasoningPart](entry)
	case partToolCall:
		part, err = decodePartData[ToolCallPart](entry)
	case partToolResult:
		part, err = decodePartData[ToolResultPart](entry)
	case partFinish:
		part, err = decodePartData[FinishPart](entry)
	case partShellCommand:
		part, err = decodePartData[ShellCommandPart](entry)
	case partImageURL, partBinary:
		// Attachments pass through as UnknownPart: the payloads are base64
		// previews that generic readers have no use for, but keeping the
		// cases explicit makes a future discriminator change loud here.
		return UnknownPart(entry), nil
	default:
		return UnknownPart(entry), nil
	}

	return part, err
}

// decodePartData unmarshals a part payload into a freshly allocated T,
// wrapping failures with the part type so a bad payload is identifiable.
func decodePartData[T Part](entry rawPart) (T, error) {
	var part T

	if err := json.Unmarshal(entry.Data, &part); err != nil {
		return part, fmt.Errorf("decode %s part: %w", entry.Type, err)
	}

	return part, nil
}
