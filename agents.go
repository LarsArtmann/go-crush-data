package crushdata

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// maxAgentDepth bounds the depth of [DB.AgentGraph]. Session graphs are
// shallow in practice; the cap turns a hypothetical parent-cycle into a
// bounded read instead of a stack overflow.
const maxAgentDepth = 64

// AgentGraph returns the subagent tree rooted at the session with the given
// ID: the root node plus every descendant linked through
// sessions.parent_session_id, in preorder (root first, each subtree ordered
// by creation time). A missing root fails with [ErrSessionNotFound].
//
// When the database predates the parent_session_id column no children can be
// linked, so the graph degrades to the root node alone — call [DB.Schema] to
// detect that condition and warn users about reduced coverage.
func (db *DB) AgentGraph(ctx context.Context, rootID string) (*AgentGraph, error) {
	root, err := db.Session(ctx, rootID)
	if err != nil {
		return nil, err
	}

	graph := &AgentGraph{
		Root:  AgentNode{Session: root, ParentID: "", Depth: 0},
		Nodes: []AgentNode{{Session: root, ParentID: "", Depth: 0}},
	}

	if !db.schema.SessionsParentSessionID {
		return graph, nil
	}

	descendants, err := db.descendantSessions(ctx, rootID)
	if err != nil {
		return nil, err
	}

	children := childIndex(descendants)

	for _, child := range children[rootID] {
		if err := db.appendSubtree(graph, children, child, 1); err != nil {
			return nil, err
		}
	}

	return graph, nil
}

// appendSubtree adds node and its descendants to the graph in preorder,
// walking the in-memory children index instead of querying per node.
func (db *DB) appendSubtree(graph *AgentGraph, children map[string][]Session, session Session, depth int) error {
	if depth > maxAgentDepth {
		return fmt.Errorf("%w: under %s in %s", ErrGraphDepthExceeded, session.ID, db.path)
	}

	graph.Nodes = append(graph.Nodes, AgentNode{
		Session:  session,
		ParentID: session.ParentSessionID,
		Depth:    depth,
	})

	for _, child := range children[session.ID] {
		if err := db.appendSubtree(graph, children, child, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// childIndex groups sessions by parent ID, each group ordered oldest-first
// by creation time — the sibling order of the preorder traversal.
func childIndex(sessions []Session) map[string][]Session {
	children := make(map[string][]Session)
	for _, session := range sessions {
		children[session.ParentSessionID] = append(children[session.ParentSessionID], session)
	}

	for _, group := range children {
		sortByCreated(group)
	}

	return children
}

// descendantSessions reads every session below rootID in one recursive-CTE
// query instead of one round trip per node (the previous N+1 shape). The
// CTE generates rows through depth maxAgentDepth+1 so an over-deep chain
// stays visible to the preorder walk — which reports it as
// [ErrGraphDepthExceeded] — instead of being silently truncated at the SQL
// boundary. The parent_session_id capability must be present.
func (db *DB) descendantSessions(ctx context.Context, rootID string) ([]Session, error) {
	costExpr := costColumn
	recursiveCostExpr := "s." + costColumn

	if !db.schema.SessionsCost {
		costExpr = "0"
		recursiveCostExpr = "0"
	}

	// query is composed from hardcoded literals and schema-gated expressions; every caller value arrives via parameterized args
	query := fmt.Sprintf(`
		WITH RECURSIVE subtree AS (
			SELECT id, title, parent_session_id, message_count, prompt_tokens, completion_tokens, %s AS cost, updated_at, created_at, todos, 1 AS depth
			FROM sessions
			WHERE parent_session_id = ?
			UNION ALL
			SELECT s.id, s.title, s.parent_session_id, s.message_count, s.prompt_tokens, s.completion_tokens, %s AS cost, s.updated_at, s.created_at, s.todos, subtree.depth + 1
			FROM sessions s
			JOIN subtree ON s.parent_session_id = subtree.id
			WHERE subtree.depth < ?
		)
		SELECT id, title, parent_session_id, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at, todos
		FROM subtree
	`, costExpr, recursiveCostExpr)

	rows, err := db.handle.QueryContext(ctx, query, rootID, maxAgentDepth+1)
	if err != nil {
		return nil, fmt.Errorf("read agent subtree of %s in %s: %w", rootID, db.path, err)
	}

	return collectRows(rows, "agent subtree", scanSession)
}

// sortByCreated orders sessions oldest-first by creation time, with the ID as
// a deterministic tiebreak. Rows arrive newest-updated-first from the sessions
// query; updated_at anti-correlates with created_at whenever a parent touched
// an older child after a newer one, so sibling order must be derived from
// created_at itself, never from a reversal of the updated_at ordering.
func sortByCreated(sessions []Session) {
	slices.SortFunc(sessions, func(a, b Session) int {
		if order := a.CreatedAt.Compare(b.CreatedAt); order != 0 {
			return order
		}

		return strings.Compare(a.ID, b.ID)
	})
}
