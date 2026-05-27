package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB

	stmtUpsertAgent      *sql.Stmt
	stmtListAgents       *sql.Stmt
	stmtUpsertSession    *sql.Stmt
	stmtGetSession       *sql.Stmt
	stmtInsertEvent      *sql.Stmt
	stmtUpsertIngestion  *sql.Stmt
	stmtGetIngestion     *sql.Stmt
	stmtUpsertMemory     *sql.Stmt
	stmtGetMemory        *sql.Stmt
	stmtInsertTimeline   *sql.Stmt
}

func Open(path string) (*DB, error) {
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := raw.Exec(p); err != nil {
			raw.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	ctx := context.Background()
	if err := RunMigrations(ctx, raw); err != nil {
		raw.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	d := &DB{sql: raw}
	if err := d.prepareStatements(); err != nil {
		raw.Close()
		return nil, fmt.Errorf("prepare statements: %w", err)
	}

	return d, nil
}

func (d *DB) Close() error {
	stmts := []*sql.Stmt{
		d.stmtUpsertAgent,
		d.stmtListAgents,
		d.stmtUpsertSession,
		d.stmtGetSession,
		d.stmtInsertEvent,
		d.stmtUpsertIngestion,
		d.stmtGetIngestion,
		d.stmtUpsertMemory,
		d.stmtGetMemory,
		d.stmtInsertTimeline,
	}
	for _, s := range stmts {
		if s != nil {
			s.Close()
		}
	}
	return d.sql.Close()
}

func (d *DB) prepareStatements() error {
	var err error

	d.stmtUpsertAgent, err = d.sql.Prepare(`
		INSERT INTO agents (id, type, name, root_path, enabled, metadata_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type=excluded.type, name=excluded.name, root_path=excluded.root_path,
			enabled=excluded.enabled, metadata_json=excluded.metadata_json, updated_at=excluded.updated_at`)
	if err != nil {
		return fmt.Errorf("upsert agent: %w", err)
	}

	d.stmtListAgents, err = d.sql.Prepare(`SELECT id, type, name, root_path, enabled, metadata_json, created_at, updated_at FROM agents ORDER BY name`)
	if err != nil {
		return fmt.Errorf("list agents: %w", err)
	}

	d.stmtUpsertSession, err = d.sql.Prepare(`
		INSERT INTO sessions (id, agent_id, external_id, source_path, project_path, title, status, started_at, updated_at, message_count, summary, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent_id, external_id, source_path) DO UPDATE SET
			title=excluded.title, status=excluded.status, updated_at=excluded.updated_at,
			message_count=excluded.message_count, summary=excluded.summary,
			metadata_json=excluded.metadata_json, project_path=excluded.project_path`)
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	d.stmtGetSession, err = d.sql.Prepare(`SELECT id, agent_id, external_id, source_path, project_path, title, status, started_at, updated_at, message_count, summary, metadata_json FROM sessions WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	d.stmtInsertEvent, err = d.sql.Prepare(`
		INSERT INTO session_events (id, session_id, agent_id, sequence, timestamp, role, kind, content, tool_name, tool_input_json, tool_output, files_json, raw_json, content_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, sequence, content_hash) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	d.stmtUpsertIngestion, err = d.sql.Prepare(`
		INSERT INTO ingestion_state (source_path, agent_id, parser_version, size_bytes, mtime, checksum, offset_bytes, last_ingested_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_path) DO UPDATE SET
			agent_id=excluded.agent_id, parser_version=excluded.parser_version,
			size_bytes=excluded.size_bytes, mtime=excluded.mtime, checksum=excluded.checksum,
			offset_bytes=excluded.offset_bytes, last_ingested_at=excluded.last_ingested_at, error=excluded.error`)
	if err != nil {
		return fmt.Errorf("upsert ingestion: %w", err)
	}

	d.stmtGetIngestion, err = d.sql.Prepare(`SELECT source_path, agent_id, parser_version, size_bytes, mtime, checksum, offset_bytes, last_ingested_at, error FROM ingestion_state WHERE source_path = ?`)
	if err != nil {
		return fmt.Errorf("get ingestion: %w", err)
	}

	d.stmtUpsertMemory, err = d.sql.Prepare(`
		INSERT INTO memory_docs (id, agent_id, source_path, project_path, title, content, size_bytes, mtime, checksum, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent_id, source_path) DO UPDATE SET
			title=excluded.title, content=excluded.content, size_bytes=excluded.size_bytes,
			mtime=excluded.mtime, checksum=excluded.checksum, project_path=excluded.project_path,
			metadata_json=excluded.metadata_json`)
	if err != nil {
		return fmt.Errorf("upsert memory: %w", err)
	}

	d.stmtGetMemory, err = d.sql.Prepare(`SELECT id, agent_id, source_path, project_path, title, content, size_bytes, mtime, checksum, metadata_json FROM memory_docs WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("get memory: %w", err)
	}

	d.stmtInsertTimeline, err = d.sql.Prepare(`
		INSERT OR IGNORE INTO timeline_items (id, timestamp, agent_id, session_id, memory_doc_id, kind, title, body, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("insert timeline: %w", err)
	}

	return nil
}
