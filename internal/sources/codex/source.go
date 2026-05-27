package codex

import (
	"io"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/models"
	"github.com/nathanmauro/agent-observatory/internal/sources"
)

type Source struct{}

func NewSource() *Source { return &Source{} }

func (s *Source) AgentInfo() models.Agent {
	return models.Agent{
		ID:           "codex-default",
		Type:         models.AgentCodex,
		Name:         "Codex CLI",
		RootPath:     "~/.codex/sessions",
		Enabled:      true,
		MetadataJSON: "{}",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

func (s *Source) ParserVersion() string                { return ParserVersion }
func (s *Source) DiscoverRoots() ([]string, error)     { return DiscoverRoots("") }
func (s *Source) DiscoverSessions() ([]string, error)  { return DiscoverSessions("") }
func (s *Source) DiscoverMemoryFiles() ([]string, error) { return DiscoverMemoryFiles() }
func (s *Source) SupportsIncremental() bool            { return true }
func (s *Source) WatchExtensions() []string            { return []string{".jsonl"} }

func (s *Source) ProjectPathFromSource(sourcePath string) string {
	return ProjectPathFromSource(sourcePath)
}

func (s *Source) ParseFile(r io.Reader, agentID string, sourcePath string, startOffset int64) (*sources.ParseResult, error) {
	return ParseSession(r, agentID, sourcePath, startOffset)
}
