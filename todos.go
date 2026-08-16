package crushdata

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TodoStatus is the lifecycle state of a [Todo]. Crush writes "pending",
// "in_progress", and "completed"; unknown values are preserved as-is so a
// Crush release adding a status never breaks reads.
type TodoStatus string

// The statuses Crush writes today.
const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
)

// A Todo is one entry of a session's todo list, as written by Crush's todo
// tool. ActiveForm is the gerund shown while the item is in progress
// ("Fixing memory path"); Content is the imperative form.
//
// The on-disk shape is pinned by a census of 71,747 items across the 287
// databases of a real registry (2026-08-16): every item carried exactly
// content, status, and active_form. Fields a future Crush version adds are
// ignored by decoding, not rejected.
type Todo struct {
	Content    string     `json:"content"`
	Status     TodoStatus `json:"status"`
	ActiveForm string     `json:"active_form"`
}

// DecodeTodos parses the raw JSON carried by [Session.Todos] into typed
// [Todo] values.
//
// Decoding is best-effort at the field level, mirroring how Crush writes
// the list: unknown status values pass through, and fields beyond the three
// known ones are ignored, so a Crush release extending the shape decodes
// with zero values for whatever it adds. nil and empty input (a NULL
// column) decode to nil without error, as does an empty JSON array; the
// bare JSON null literal decodes to nil too, since unmarshalling it into a
// slice leaves the slice nil. Input that is not a JSON array of objects
// fails with an error; the caller still holds the raw bytes and can decode
// them its own way.
func DecodeTodos(raw json.RawMessage) ([]Todo, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "[]" {
		return nil, nil
	}

	var todos []Todo
	if err := json.Unmarshal(raw, &todos); err != nil {
		return nil, fmt.Errorf("decode todos: %w", err)
	}

	return todos, nil
}
