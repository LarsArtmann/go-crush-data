package crushdata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// TestAgentGraphSiblingsOrderedByCreatedNotUpdated pins the preorder
// contract against updated_at anti-correlation: sessions arrive newest-
// updated-first, and reversing that order is NOT created order. Siblings
// must be sequenced by created_at itself.
func TestAgentGraphSiblingsOrderedByCreatedNotUpdated(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		// child-a is created first but updated last; child-b the reverse.
		insertSession(t, db, "root", "", "Root", 1, fixtureBase, fixtureBase+10)
		insertSession(t, db, "child-a", "root", "Child A", 1, fixtureBase+1, fixtureBase+9)
		insertSession(t, db, "child-b", "root", "Child B", 1, fixtureBase+2, fixtureBase+3)
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

	want := []string{"root", "child-a", "child-b"}
	if len(graph.Nodes) != len(want) {
		t.Fatalf("nodes = %d, want %d: %+v", len(graph.Nodes), len(want), graph.Nodes)
	}

	for i, id := range want {
		if graph.Nodes[i].Session.ID != id {
			t.Fatalf(
				"node[%d] = %q, want %q (siblings by created_at, not updated_at)",
				i, graph.Nodes[i].Session.ID, id,
			)
		}
	}
}

// TestAgentGraphDepthCapExceeded pins the recursion bound: a chain one link
// deeper than maxAgentDepth fails with ErrGraphDepthExceeded instead of
// exhausting the stack.
func TestAgentGraphDepthCapExceeded(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "root", "", "Root", 0, fixtureBase, fixtureBase)

		// root is depth 0; child i sits at depth i. The child at depth
		// maxAgentDepth + 1 trips the cap.

		parent := "root"

		for i := 1; i <= maxAgentDepth+1; i++ {
			id := fmt.Sprintf("chain-%03d", i)
			insertSession(t, db, id, parent, "Chain", 0, fixtureBase+int64(i), fixtureBase+int64(i))
			parent = id
		}
	})

	db, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	_, err = db.AgentGraph(context.Background(), "root")
	if !errors.Is(err, ErrGraphDepthExceeded) {
		t.Fatalf("err = %v, want ErrGraphDepthExceeded", err)
	}
}

// TestAgentGraphWideFanOut stress-checks preorder traversal breadth: 100
// children of one root arrive in created_at order with no reordering or
// truncation.
func TestAgentGraphWideFanOut(t *testing.T) {
	t.Parallel()

	const childCount = 100

	dataDir := t.TempDir()

	createDBAt(t, filepath.Join(dataDir, DBName), schemaCurrent, func(db *sql.DB) {
		insertSession(t, db, "root", "", "Root", 0, fixtureBase, fixtureBase)

		for i := range childCount {
			id := fmt.Sprintf("fan-%03d", i)
			createdAt := fixtureBase + int64(i) + 1
			insertSession(t, db, id, "root", "Fan child", 0, createdAt, createdAt)
		}
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

	if len(graph.Nodes) != childCount+1 {
		t.Fatalf("nodes = %d, want %d", len(graph.Nodes), childCount+1)
	}

	if graph.Nodes[1].Session.ID != "fan-000" || graph.Nodes[childCount].Session.ID != "fan-099" {
		t.Fatalf(
			"children out of created_at order: first=%s last=%s",
			graph.Nodes[1].Session.ID, graph.Nodes[childCount].Session.ID,
		)
	}

	for i, node := range graph.Nodes[1:] {
		if node.Depth != 1 || node.ParentID != "root" {
			t.Fatalf("node[%d] = %+v, want depth 1 under root", i+1, node)
		}
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
