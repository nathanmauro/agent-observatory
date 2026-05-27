package sources

import (
	"io"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/models"
)

type ParseResult struct {
	SessionID    string
	Title        string
	ProjectPath  string
	Events       []models.SessionEvent
	StartedAt    time.Time
	UpdatedAt    time.Time
	UserMsgCount int
	OffsetBytes  int64
}

type Source interface {
	AgentInfo() models.Agent
	ParserVersion() string
	DiscoverRoots() ([]string, error)
	DiscoverSessions() ([]string, error)
	ProjectPathFromSource(sourcePath string) string
	ParseFile(r io.Reader, agentID string, sourcePath string, startOffset int64) (*ParseResult, error)
	SupportsIncremental() bool
	WatchExtensions() []string
}
