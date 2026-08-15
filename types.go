package crushdata

import (
	"encoding/json"
	"time"
)

// Project is one entry from the Crush project registry: a working directory
// (Path), the data directory that holds its sessions (DataDir), and when
// Crush last used it (LastAccessed). The JSON tags mirror the registry's
// on-disk shape so a []Project decodes projects.json directly.
//
// The registry can map many working directories to the same DataDir (for
// example, a parent directory and all its subprojects can share one data
// directory). [DiscoverProjects] de-duplicates such entries down to one
// Project per DataDir, keeping the most recently accessed Path.
type Project struct {
	Path         string    `json:"path"`
	DataDir      string    `json:"data_dir"`
	LastAccessed time.Time `json:"last_accessed"`
}

// Session is one row of the sessions table: a conversation with an agent,
// either a root session started by a user or an auxiliary session spawned by
// the agent tool (ParentSessionID != ""). The Todos field carries the raw
// JSON of the Todos column (json.RawMessage) so callers decode it into
// whatever shape their Crush version writes; it is nil when the column is
// NULL.
type Session struct {
	ID               string
	Title            string
	ParentSessionID  string
	MessageCount     int
	PromptTokens     int64
	CompletionTokens int64
	CostUSD          float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Todos            json.RawMessage
}

// Role is the author of a message. Crush currently writes "user",
// "assistant", and "tool"; unknown values are preserved as-is so a Crush
// release adding a role never breaks reads.
type Role string

// The roles Crush writes today.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one row of the messages table with its parts JSON decoded into
// typed [Part] values. Parts is nil when the stored JSON is empty or
// malformed — a single corrupted message never fails the read, mirroring how
// tolerant readers must behave against a drifting schema. Use [DecodeParts]
// directly for strict decoding.
type Message struct {
	ID         string
	SessionID  string
	Role       Role
	Parts      []Part
	Model      string
	Provider   string
	CreatedAt  time.Time
	FinishedAt time.Time
}

// AgentNode is one session in an [AgentGraph].
type AgentNode struct {
	Session  Session
	ParentID string // parent session ID; "" for the root node
	Depth    int    // 0 for the root, 1 for its children, and so on
}

// AgentGraph is the subagent tree below (and including) a root session,
// built from the parent_session_id column. When the database predates that
// column the graph degrades to the root node alone.
type AgentGraph struct {
	Root  AgentNode
	Nodes []AgentNode // preorder: root first, then each subtree by created_at
}

// Stats aggregates a day of activity. The zero value means "no activity".
//
// Counts mirror the session table's own aggregates: MessageCount sums the
// sessions.message_count column, not a live count of message rows, and token
// and cost sums come from the per-session columns. [Stats.ModelBreakdown]
// is the only member that joins messages (to learn which model each session
// ran) and therefore counts message rows directly.
type Stats struct {
	SessionCount     int
	MessageCount     int
	PromptTokens     int64
	CompletionTokens int64
	CostUSD          float64
	Models           []string
	Providers        []string
	SessionTitles    []string
	HourHistogram    [24]int
	ModelBreakdown   []ModelStat
}

// ModelStat is one model's share of a day's activity. See [Stats] for the
// count semantics.
type ModelStat struct {
	Model            string
	SessionCount     int
	MessageCount     int
	PromptTokens     int64
	CompletionTokens int64
	CostUSD          float64
}
