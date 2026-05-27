package indexer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/db"
	"github.com/nathanmauro/agent-observatory/internal/events"
	"github.com/nathanmauro/agent-observatory/internal/models"
	"github.com/nathanmauro/agent-observatory/internal/sources"
)

type Indexer struct {
	db      *db.DB
	bus     *events.Bus
	srcs    []sources.Source
	srcMap  map[string]sources.Source
}

func New(database *db.DB, bus *events.Bus, srcs []sources.Source) *Indexer {
	srcMap := make(map[string]sources.Source, len(srcs))
	for _, s := range srcs {
		srcMap[s.AgentInfo().ID] = s
	}
	return &Indexer{db: database, bus: bus, srcs: srcs, srcMap: srcMap}
}

func (ix *Indexer) IndexAll(ctx context.Context) error {
	for _, src := range ix.srcs {
		agent := src.AgentInfo()
		if err := ix.ensureAgent(ctx, agent); err != nil {
			log.Printf("ensure agent %s: %v", agent.ID, err)
			continue
		}

		paths, err := src.DiscoverSessions()
		if err != nil {
			log.Printf("discover %s sessions: %v", agent.ID, err)
			continue
		}

		log.Printf("indexing %s: %d session files", agent.Name, len(paths))
		for _, path := range paths {
			if err := ix.indexFile(ctx, path, src); err != nil {
				log.Printf("index %s [%s]: %v", path, agent.ID, err)
			}
		}

		memPaths, err := src.DiscoverMemoryFiles()
		if err != nil {
			log.Printf("discover %s memory: %v", agent.ID, err)
			continue
		}
		if len(memPaths) > 0 {
			log.Printf("indexing %s: %d memory files", agent.Name, len(memPaths))
			for _, path := range memPaths {
				if err := ix.indexMemoryFile(ctx, path, src); err != nil {
					log.Printf("index memory %s [%s]: %v", path, agent.ID, err)
				}
			}
		}
	}
	return nil
}

func (ix *Indexer) ensureAgent(ctx context.Context, agent models.Agent) error {
	now := time.Now().UTC()
	agent.CreatedAt = now
	agent.UpdatedAt = now
	return ix.db.UpsertAgent(ctx, agent)
}

func (ix *Indexer) IndexFile(ctx context.Context, path string) error {
	src := ix.sourceForPath(path)
	if src == nil {
		return fmt.Errorf("no source adapter for path: %s", path)
	}
	return ix.indexFile(ctx, path, src)
}

func (ix *Indexer) sourceForPath(path string) sources.Source {
	for _, src := range ix.srcs {
		roots, err := src.DiscoverRoots()
		if err != nil || len(roots) == 0 {
			continue
		}
		for _, root := range roots {
			rel, err := filepath.Rel(root, path)
			if err == nil && len(rel) > 0 && rel[0] != '.' {
				return src
			}
		}
	}
	return nil
}

func (ix *Indexer) indexFile(ctx context.Context, path string, src sources.Source) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	agent := src.AgentInfo()
	pv := src.ParserVersion()

	existing, err := ix.db.GetIngestion(ctx, path)
	if err != nil {
		return fmt.Errorf("get ingestion: %w", err)
	}

	var startOffset int64
	if existing != nil && existing.ParserVersion == pv {
		sameMtime := existing.Mtime.Unix() == info.ModTime().Unix()
		sameSize := existing.SizeBytes == info.Size()

		if sameMtime && sameSize {
			return nil
		}

		if src.SupportsIncremental() && info.Size() >= existing.SizeBytes {
			startOffset = existing.OffsetBytes
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	result, err := src.ParseFile(f, agent.ID, path, startOffset)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if result == nil || (result.SessionID == "" && len(result.Events) == 0) {
		return nil
	}

	projectPath := src.ProjectPathFromSource(path)
	if projectPath == "" {
		projectPath = result.ProjectPath
	}

	sessionID := deterministicID(path)
	status := "unknown"
	if result.UserMsgCount > 0 {
		status = "active"
	}

	for i := range result.Events {
		result.Events[i].SessionID = sessionID
	}

	title := result.Title
	if title == "" && result.SessionID != "" {
		title = result.SessionID
	}

	sess := models.Session{
		ID:           sessionID,
		AgentID:      agent.ID,
		ExternalID:   result.SessionID,
		SourcePath:   path,
		ProjectPath:  projectPath,
		Title:        title,
		Status:       status,
		StartedAt:    result.StartedAt,
		UpdatedAt:    result.UpdatedAt,
		MessageCount: result.UserMsgCount,
		MetadataJSON: "{}",
	}
	isNewSession := existing == nil || existing.ParserVersion != pv
	if err := ix.db.UpsertSession(ctx, sess); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	ix.emitTimelineForSession(ctx, sess, isNewSession)
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
		AgentID:        agent.ID,
		ParserVersion:  pv,
		SizeBytes:      info.Size(),
		Mtime:          info.ModTime(),
		OffsetBytes:    result.OffsetBytes,
		LastIngestedAt: now,
	}); err != nil {
		return fmt.Errorf("upsert ingestion: %w", err)
	}

	return nil
}

func (ix *Indexer) indexMemoryFile(ctx context.Context, path string, src sources.Source) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	agent := src.AgentInfo()
	checksum := fmt.Sprintf("%d-%d", info.Size(), info.ModTime().UnixNano())

	existing, err := ix.db.GetIngestion(ctx, "mem:"+path)
	if err != nil {
		return fmt.Errorf("get ingestion: %w", err)
	}
	if existing != nil && existing.Checksum == checksum {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	title := filepath.Base(path)
	projectPath := src.ProjectPathFromSource(path)

	docID := deterministicID("mem:" + path)
	doc := models.MemoryDoc{
		ID:           docID,
		AgentID:      agent.ID,
		SourcePath:   path,
		ProjectPath:  projectPath,
		Title:        title,
		Content:      string(content),
		SizeBytes:    info.Size(),
		Mtime:        info.ModTime(),
		Checksum:     checksum,
		MetadataJSON: "{}",
	}
	if err := ix.db.UpsertMemoryDoc(ctx, doc); err != nil {
		return fmt.Errorf("upsert memory doc: %w", err)
	}

	isNew := existing == nil
	tlKind := "memory.updated"
	if isNew {
		tlKind = "memory.created"
	}
	ix.db.InsertTimelineItem(ctx, models.TimelineItem{
		ID:           deterministicID(fmt.Sprintf("tl:%s:%s", tlKind, path)),
		Timestamp:    info.ModTime(),
		AgentID:      agent.ID,
		MemoryDocID:  docID,
		Kind:         tlKind,
		Title:        fmt.Sprintf("%s: %s", agent.Name, title),
		MetadataJSON: "{}",
	})

	if ix.bus != nil {
		ix.bus.Publish(events.Event{
			Type:  "memory.updated",
			Topic: "memory",
			Data:  doc,
		})
	}

	now := time.Now().UTC()
	return ix.db.UpsertIngestion(ctx, models.IngestionState{
		SourcePath:     "mem:" + path,
		AgentID:        agent.ID,
		ParserVersion:  "mem-v1",
		SizeBytes:      info.Size(),
		Mtime:          info.ModTime(),
		Checksum:       checksum,
		LastIngestedAt: now,
	})
}

func (ix *Indexer) emitTimelineForSession(ctx context.Context, sess models.Session, isNew bool) {
	kind := "session.updated"
	if isNew {
		kind = "session.created"
	}
	ix.db.InsertTimelineItem(ctx, models.TimelineItem{
		ID:           deterministicID(fmt.Sprintf("tl:%s:%s", kind, sess.ID)),
		Timestamp:    sess.UpdatedAt,
		AgentID:      sess.AgentID,
		SessionID:    sess.ID,
		Kind:         kind,
		Title:        sess.Title,
		Body:         sess.Summary,
		MetadataJSON: "{}",
	})
}

func deterministicID(sourcePath string) string {
	h := sha256.Sum256([]byte(sourcePath))
	return fmt.Sprintf("%x", h[:16])
}
