package crushdata

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestAgentGraphRootWithChild(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	graph, err := db.AgentGraph(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("AgentGraph: %v", err)
	}

	if graph.Root.Session.ID != "fixture-root" || graph.Root.Depth != 0 {
		t.Fatalf("root = %+v", graph.Root)
	}

	if len(graph.Nodes) != 2 {
		t.Fatalf("nodes = %d, want 2 (root + child): %+v", len(graph.Nodes), graph.Nodes)
	}

	child := graph.Nodes[1]
	if child.Session.ID != "m_assistant_1$$call_agent_1" {
		t.Fatalf("child = %q", child.Session.ID)
	}

	if child.ParentID != "fixture-root" || child.Depth != 1 {
		t.Fatalf("child = %+v", child)
	}
}

// TestAgentGraphNestedSubagents builds root → child → grandchild and checks
// preorder traversal reaches depth 2.
func TestAgentGraphNestedSubagents(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "root", "", "Root", 1, fixtureBase, fixtureBase+10)
		insertSession(t, db, "child-a", "root", "Child A", 1, fixtureBase+1, fixtureBase+5)
		insertSession(t, db, "child-b", "root", "Child B", 1, fixtureBase+2, fixtureBase+6)
		insertSession(t, db, "grandchild", "child-a", "Grandchild", 1, fixtureBase+3, fixtureBase+4)
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	graph, err := db.AgentGraph(context.Background(), "root")
	if err != nil {
		t.Fatalf("AgentGraph: %v", err)
	}

	wantOrder := []string{"root", "child-a", "grandchild", "child-b"}
	if len(graph.Nodes) != len(wantOrder) {
		t.Fatalf("nodes = %d, want %d: %+v", len(graph.Nodes), len(wantOrder), graph.Nodes)
	}

	for i, want := range wantOrder {
		if graph.Nodes[i].Session.ID != want {
			t.Fatalf("node[%d] = %q, want %q (preorder, children by created_at)", i, graph.Nodes[i].Session.ID, want)
		}
	}

	if graph.Nodes[2].Depth != 2 || graph.Nodes[2].ParentID != "child-a" {
		t.Fatalf("grandchild = %+v", graph.Nodes[2])
	}
}

func TestAgentGraphMissingRoot(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaCurrent)

	_, err := db.AgentGraph(context.Background(), "no-such-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

// TestAgentGraphLegacySchemaFlatFallback: without parent_session_id no
// children can be linked; the graph degrades to the root node alone.
func TestAgentGraphLegacySchemaFlatFallback(t *testing.T) {
	t.Parallel()

	db := openFixture(t, schemaLegacy)

	graph, err := db.AgentGraph(context.Background(), "fixture-root")
	if err != nil {
		t.Fatalf("AgentGraph on legacy schema: %v", err)
	}

	if len(graph.Nodes) != 1 || graph.Nodes[0].Session.ID != "fixture-root" {
		t.Fatalf("nodes = %+v, want root only", graph.Nodes)
	}
}
