package augment

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
		ID:           "augment-default",
		Type:         models.AgentAuggie,
		Name:         "Augment (Auggie)",
		RootPath:     "~/.augment/sessions",
		Enabled:      true,
		MetadataJSON: "{}",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

func (s *Source) ParserVersion() string              { return ParserVersion }
func (s *Source) DiscoverRoots() ([]string, error)   { return DiscoverRoots("") }
func (s *Source) DiscoverSessions() ([]string, error) { return DiscoverSessions("") }
func (s *Source) ProjectPathFromSource(_ string) string { return "" }
func (s *Source) SupportsIncremental() bool          { return false }
func (s *Source) WatchExtensions() []string          { return []string{".json"} }

func (s *Source) ParseFile(r io.Reader, agentID string, sourcePath string, startOffset int64) (*sources.ParseResult, error) {
	return ParseSession(r, agentID, sourcePath, startOffset)
}
