package models

import "time"

type AgentType string

const (
	AgentClaude AgentType = "claude"
	AgentCodex  AgentType = "codex"
	AgentAuggie AgentType = "auggie"
	AgentCursor AgentType = "cursor"
)

type Agent struct {
	ID           string    `json:"id"`
	Type         AgentType `json:"type"`
	Name         string    `json:"name"`
	RootPath     string    `json:"root_path"`
	Enabled      bool      `json:"enabled"`
	MetadataJSON string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID           string    `json:"id"`
	AgentID      string    `json:"agent_id"`
	ExternalID   string    `json:"external_id,omitempty"`
	SourcePath   string    `json:"source_path,omitempty"`
	ProjectPath  string    `json:"project_path,omitempty"`
	Title        string    `json:"title,omitempty"`
	Status       string    `json:"status,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	Summary      string    `json:"summary,omitempty"`
	MetadataJSON string    `json:"-"`
}

type EventKind string

const (
	EventMessage    EventKind = "message"
	EventToolUse    EventKind = "tool_use"
	EventToolResult EventKind = "tool_result"
	EventThinking   EventKind = "thinking"
	EventSystem     EventKind = "system"
	EventMeta       EventKind = "meta"
)

type SessionEvent struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	AgentID       string    `json:"agent_id"`
	Sequence      int       `json:"sequence"`
	Timestamp     time.Time `json:"timestamp"`
	Role          string    `json:"role,omitempty"`
	Kind          EventKind `json:"kind"`
	Content       string    `json:"content,omitempty"`
	ToolName      string    `json:"tool_name,omitempty"`
	ToolInputJSON string    `json:"tool_input,omitempty"`
	ToolOutput    string    `json:"tool_output,omitempty"`
	FilesJSON     string    `json:"-"`
	RawJSON       string    `json:"-"`
	ContentHash   string    `json:"-"`
}

type IngestionState struct {
	SourcePath     string    `json:"source_path"`
	AgentID        string    `json:"agent_id"`
	ParserVersion  string    `json:"parser_version"`
	SizeBytes      int64     `json:"size_bytes"`
	Mtime          time.Time `json:"mtime"`
	Checksum       string    `json:"checksum,omitempty"`
	OffsetBytes    int64     `json:"offset_bytes,omitempty"`
	LastIngestedAt time.Time `json:"last_ingested_at"`
	Error          string    `json:"error,omitempty"`
}

type SearchResult struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	AgentType string `json:"agent_type,omitempty"`
	Title     string `json:"title"`
	Path      string `json:"path,omitempty"`
	Project   string `json:"project,omitempty"`
	Snippet   string `json:"snippet,omitempty"`
	Rank      float64 `json:"rank,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type Process struct {
	PID        int       `json:"pid"`
	PPID       int       `json:"ppid"`
	Name       string    `json:"name"`
	AgentType  AgentType `json:"agent_type"`
	Command    string    `json:"command"`
	CPUPercent float64   `json:"cpu_percent"`
	RSSBytes   int64     `json:"rss_bytes"`
	StartTime  time.Time `json:"start_time,omitempty"`
	Status     string    `json:"status"`
}

type WSMessage struct {
	Seq           uint64 `json:"seq"`
	SchemaVersion int    `json:"schema_version"`
	Type          string `json:"type"`
	Topic         string `json:"topic"`
	SentAt        string `json:"sent_at"`
	Data          any    `json:"data"`
}

type WSClientMessage struct {
	Type    string   `json:"type"`
	LastSeq uint64   `json:"last_seq,omitempty"`
	Topics  []string `json:"topics,omitempty"`
}

type MemoryDoc struct {
	ID           string    `json:"id"`
	AgentID      string    `json:"agent_id"`
	SourcePath   string    `json:"source_path"`
	ProjectPath  string    `json:"project_path,omitempty"`
	Title        string    `json:"title"`
	Content      string    `json:"content,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	Mtime        time.Time `json:"mtime"`
	Checksum     string    `json:"-"`
	MetadataJSON string    `json:"-"`
}

type TimelineItem struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	AgentID      string    `json:"agent_id"`
	AgentType    string    `json:"agent_type,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	MemoryDocID  string    `json:"memory_doc_id,omitempty"`
	Kind         string    `json:"kind"`
	Title        string    `json:"title"`
	Body         string    `json:"body,omitempty"`
	MetadataJSON string    `json:"-"`
}

type Stats struct {
	TotalSessions int            `json:"total_sessions"`
	TotalEvents   int            `json:"total_events"`
	TotalMemory   int            `json:"total_memory_docs"`
	AgentCounts   map[string]int `json:"agent_counts"`
}

type Pagination struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit"`
}

type PagedResponse[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	Total      int    `json:"total,omitempty"`
}
