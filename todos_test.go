package crushdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestDecodeTodos pins the DecodeTodos contract: the census shape decodes
// fully, empty shapes yield nil, drift (unknown statuses, extra fields)
// passes through instead of failing, and only input that is not a JSON
// array of objects errors.
func TestDecodeTodos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []Todo
		wantErr bool
	}{
		{name: "nil input (NULL column)", raw: "", want: nil},
		{name: "json null", raw: "null", want: nil},
		{name: "empty array", raw: "[]", want: nil},
		{name: "padded json null", raw: " null ", want: nil},
		{
			name: "census shape",
			raw:  `[{"content":"Ship it","status":"completed","active_form":"Shipping it"},{"content":"Write tests","status":"pending","active_form":"Writing tests"}]`,
			want: []Todo{
				{Content: "Ship it", Status: TodoCompleted, ActiveForm: "Shipping it"},
				{Content: "Write tests", Status: TodoPending, ActiveForm: "Writing tests"},
			},
		},
		{
			name: "unknown status passes through",
			raw:  `[{"content":"Drifted","status":"blocked","active_form":"Waiting"}]`,
			want: []Todo{{Content: "Drifted", Status: TodoStatus("blocked"), ActiveForm: "Waiting"}},
		},
		{
			name: "fields Crush adds later are ignored",
			raw:  `[{"content":"Ship it","status":"pending","active_form":"Shipping it","due":"2027-01-01"}]`,
			want: []Todo{{Content: "Ship it", Status: TodoPending, ActiveForm: "Shipping it"}},
		},
		{
			name: "fields Crush drops decode as zero values",
			raw:  `[{"content":"Ship it"}]`,
			want: []Todo{{Content: "Ship it"}},
		},
		{name: "object instead of array", raw: `{}`, wantErr: true},
		{name: "array of non-objects", raw: `[1]`, wantErr: true},
		{name: "not json", raw: `pending`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var raw json.RawMessage
			if tt.raw != "" {
				raw = json.RawMessage(tt.raw)
			}

			got, err := DecodeTodos(raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("DecodeTodos(%s) = %v, want error", tt.raw, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("DecodeTodos(%s): %v", tt.raw, err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("DecodeTodos(%s) = %v, want %v", tt.raw, got, tt.want)
			}

			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("DecodeTodos(%s)[%d] = %+v, want %+v", tt.raw, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestDecodeTodosCensusShape pins the on-disk item shape against a verbatim
// sample from a real database: when Crush changes how it writes todo lists,
// this test is the tripwire that says the pinned shape needs revisiting.
func TestDecodeTodosCensusShape(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[` +
		`{"content":"Fix memory path: remove fake picoclaw path, adapt to how Crush actually works","status":"completed","active_form":"Fixing memory path"},` +
		`{"content":"Audit all Nix configs for errors","status":"in_progress","active_form":"Auditing all Nix configs for errors"},` +
		`{"content":"Verify all builds pass","status":"pending","active_form":"Verifying all builds pass"}` +
		`]`)

	todos, err := DecodeTodos(raw)
	if err != nil {
		t.Fatalf("DecodeTodos: %v", err)
	}

	if len(todos) != 3 {
		t.Fatalf("len(todos) = %d, want 3", len(todos))
	}

	want := []struct {
		status     TodoStatus
		content    string
		activeForm string
	}{
		{
			status:     TodoCompleted,
			content:    "Fix memory path: remove fake picoclaw path, adapt to how Crush actually works",
			activeForm: "Fixing memory path",
		},
		{
			status:     TodoInProgress,
			content:    "Audit all Nix configs for errors",
			activeForm: "Auditing all Nix configs for errors",
		},
		{status: TodoPending, content: "Verify all builds pass", activeForm: "Verifying all builds pass"},
	}

	for i, expected := range want {
		if todos[i].Status != expected.status || todos[i].Content != expected.content ||
			todos[i].ActiveForm != expected.activeForm {
			t.Fatalf(
				"todos[%d] = %+v, want status=%s content=%q active_form=%q",
				i,
				todos[i],
				expected.status,
				expected.content,
				expected.activeForm,
			)
		}
	}
}

// TestDecodeTodosFromSession proves the composition end to end: the bytes a
// read hands back decode into typed values without an intermediate copy.
func TestDecodeTodosFromSession(t *testing.T) {
	t.Parallel()

	rawTodos := `[{"content":"Ship it","status":"in_progress","active_form":"Shipping it"},{"content":"Write tests","status":"pending","active_form":"Writing tests"}]`

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "with-todos", "", "With todos", 0, fixtureBase, fixtureBase)

		if _, err := db.ExecContext(
			context.Background(),
			"UPDATE sessions SET todos = ? WHERE id = 'with-todos'",
			rawTodos,
		); err != nil {
			t.Fatal(err)
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	session, err := db.Session(context.Background(), "with-todos")
	if err != nil {
		t.Fatalf("Session: %v", err)
	}

	todos, err := DecodeTodos(session.Todos)
	if err != nil {
		t.Fatalf("DecodeTodos(session.Todos): %v", err)
	}

	if len(todos) != 2 || todos[0].Status != TodoInProgress || todos[1].Status != TodoPending {
		t.Fatalf("todos = %+v, want 2 items (in_progress, pending)", todos)
	}
}
