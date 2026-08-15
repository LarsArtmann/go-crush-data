package crushdata

import (
	"context"
	"fmt"
)

// maxAgentDepth bounds the recursion of [DB.AgentGraph]. Session graphs are
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

	children, err := db.childSessions(ctx, rootID)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		if err := db.appendSubtree(ctx, graph, child, 1); err != nil {
			return nil, err
		}
	}

	return graph, nil
}

// appendSubtree adds node and its descendants to the graph in preorder.
func (db *DB) appendSubtree(ctx context.Context, graph *AgentGraph, session Session, depth int) error {
	if depth > maxAgentDepth {
		return fmt.Errorf("%w: under %s in %s", ErrGraphDepthExceeded, session.ID, db.path)
	}

	node := AgentNode{
		Session:  session,
		ParentID: session.ParentSessionID,
		Depth:    depth,
	}

	graph.Nodes = append(graph.Nodes, node)

	children, err := db.childSessions(ctx, session.ID)
	if err != nil {
		return err
	}

	for _, child := range children {
		if err := db.appendSubtree(ctx, graph, child, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// childSessions returns the direct children of parentID ordered by creation
// time. The parent_session_id capability must be present.
func (db *DB) childSessions(ctx context.Context, parentID string) ([]Session, error) {
	//nolint:exhaustruct // the parent filter is the only relevant condition here
	sessions, err := db.Sessions(ctx, SessionFilter{ParentID: parentID})
	if err != nil {
		return nil, err
	}

	reverseByCreated(sessions)

	return sessions, nil
}

// reverseByCreated flips newest-first row order into oldest-first, the
// preorder traversal order.
func reverseByCreated(sessions []Session) {
	for i, j := 0, len(sessions)-1; i < j; i, j = i+1, j-1 {
		sessions[i], sessions[j] = sessions[j], sessions[i]
	}
}
