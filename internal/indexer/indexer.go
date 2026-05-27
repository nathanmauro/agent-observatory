package indexer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/db"
	"github.com/nathanmauro/agent-observatory/internal/events"
	"github.com/nathanmauro/agent-observatory/internal/models"
	"github.com/nathanmauro/agent-observatory/internal/sources/claude"
)

const (
	agentID       = "claude-default"
	parserVersion = "claude-v1"
)

type Indexer struct {
	db  *db.DB
	bus *events.Bus
}

func New(database *db.DB, bus *events.Bus) *Indexer {
	return &Indexer{db: database, bus: bus}
}

func (ix *Indexer) IndexAll(ctx context.Context) error {
	if err := ix.ensureAgent(ctx); err != nil {
		return fmt.Errorf("ensure agent: %w", err)
	}

	paths, err := claude.DiscoverSessions("")
	if err != nil {
		return fmt.Errorf("discover sessions: %w", err)
	}

	for _, path := range paths {
		if err := ix.indexFile(ctx, path); err != nil {
			log.Printf("index %s: %v", path, err)
		}
	}
	return nil
}

func (ix *Indexer) ensureAgent(ctx context.Context) error {
	now := time.Now().UTC()
	return ix.db.UpsertAgent(ctx, models.Agent{
		ID:           agentID,
		Type:         models.AgentClaude,
		Name:         "Claude Code",
		RootPath:     "~/.claude/projects",
		Enabled:      true,
		MetadataJSON: "{}",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (ix *Indexer) IndexFile(ctx context.Context, path string) error {
	return ix.indexFile(ctx, path)
}

func (ix *Indexer) indexFile(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	existing, err := ix.db.GetIngestion(ctx, path)
	if err != nil {
		return fmt.Errorf("get ingestion: %w", err)
	}

	var startOffset int64
	if existing != nil && existing.ParserVersion == parserVersion {
		sameMtime := existing.Mtime.Unix() == info.ModTime().Unix()
		sameSize := existing.SizeBytes == info.Size()

		if sameMtime && sameSize {
			return nil
		}

		if info.Size() >= existing.SizeBytes {
			startOffset = existing.OffsetBytes
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	result, err := claude.ParseSession(f, agentID, path, startOffset)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if result.SessionID == "" {
		return nil
	}

	projectPath := claude.ProjectPathFromSource(path)

	// Multiple JSONL files (parent + subagents) share the same sessionId, so
	// derive a DB-unique session ID from the source path. The JSONL sessionId
	// stays in ExternalID for cross-referencing.
	sessionID := deterministicID(path)
	status := "unknown"
	if result.UserMsgCount > 0 {
		status = "active"
	}

	for i := range result.Events {
		result.Events[i].SessionID = sessionID
	}

	sess := models.Session{
		ID:           sessionID,
		AgentID:      agentID,
		ExternalID:   result.SessionID,
		SourcePath:   path,
		ProjectPath:  projectPath,
		Title:        result.Title,
		Status:       status,
		StartedAt:    result.StartedAt,
		UpdatedAt:    result.UpdatedAt,
		MessageCount: result.UserMsgCount,
		MetadataJSON: "{}",
	}
	if err := ix.db.UpsertSession(ctx, sess); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	if ix.bus != nil {
		ix.bus.Publish(events.Event{
			Type:  "session.updated",
			Topic: "sessions",
			Data:  sess,
		})
	}

	const batchSize = 500
	for i := 0; i < len(result.Events); i += batchSize {
		end := i + batchSize
		if end > len(result.Events) {
			end = len(result.Events)
		}
		if err := ix.db.InsertEvents(ctx, result.Events[i:end]); err != nil {
			return fmt.Errorf("insert events batch at %d: %w", i, err)
		}
	}
	if ix.bus != nil && len(result.Events) > 0 {
		ix.bus.Publish(events.Event{
			Type:  "event.appended",
			Topic: "sessions",
			Data: map[string]any{
				"session_id":  sessionID,
				"event_count": len(result.Events),
			},
		})
	}

	now := time.Now().UTC()
	if err := ix.db.UpsertIngestion(ctx, models.IngestionState{
		SourcePath:     path,
		AgentID:        agentID,
		ParserVersion:  parserVersion,
		SizeBytes:      info.Size(),
		Mtime:          info.ModTime(),
		OffsetBytes:    result.OffsetBytes,
		LastIngestedAt: now,
	}); err != nil {
		return fmt.Errorf("upsert ingestion: %w", err)
	}

	return nil
}

func deterministicID(sourcePath string) string {
	h := sha256.Sum256([]byte(sourcePath))
	return fmt.Sprintf("%x", h[:16])
}
