package db

import (
	"context"
	"database/sql"
	"fmt"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		root_path TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL REFERENCES agents(id),
		external_id TEXT,
		source_path TEXT,
		project_path TEXT,
		title TEXT,
		status TEXT DEFAULT 'unknown',
		started_at TEXT,
		updated_at TEXT,
		message_count INTEGER NOT NULL DEFAULT 0,
		summary TEXT,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		UNIQUE(agent_id, external_id, source_path)
	)`,

	`CREATE TABLE IF NOT EXISTS session_events (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		agent_id TEXT NOT NULL,
		sequence INTEGER NOT NULL,
		timestamp TEXT,
		role TEXT,
		kind TEXT NOT NULL,
		content TEXT,
		tool_name TEXT,
		tool_input_json TEXT,
		tool_output TEXT,
		files_json TEXT NOT NULL DEFAULT '[]',
		raw_json TEXT,
		content_hash TEXT NOT NULL,
		UNIQUE(session_id, sequence, content_hash)
	)`,

	`CREATE TABLE IF NOT EXISTS ingestion_state (
		source_path TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		parser_version TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		mtime TEXT NOT NULL,
		checksum TEXT,
		offset_bytes INTEGER,
		last_ingested_at TEXT NOT NULL,
		error TEXT
	)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(
		title, summary, project_path,
		content='sessions',
		content_rowid='rowid'
	)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
		content, tool_name, tool_output,
		content='session_events',
		content_rowid='rowid'
	)`,

	`CREATE TRIGGER IF NOT EXISTS sessions_ai AFTER INSERT ON sessions BEGIN
		INSERT INTO sessions_fts(rowid, title, summary, project_path) VALUES (new.rowid, new.title, new.summary, new.project_path);
	END`,

	`CREATE TRIGGER IF NOT EXISTS sessions_au AFTER UPDATE ON sessions BEGIN
		INSERT INTO sessions_fts(sessions_fts, rowid, title, summary, project_path) VALUES('delete', old.rowid, old.title, old.summary, old.project_path);
		INSERT INTO sessions_fts(rowid, title, summary, project_path) VALUES (new.rowid, new.title, new.summary, new.project_path);
	END`,

	`CREATE TRIGGER IF NOT EXISTS events_ai AFTER INSERT ON session_events BEGIN
		INSERT INTO events_fts(rowid, content, tool_name, tool_output) VALUES (new.rowid, new.content, new.tool_name, new.tool_output);
	END`,

	`CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_id)`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_events_session ON session_events(session_id, sequence)`,
	`CREATE INDEX IF NOT EXISTS idx_ingestion_agent ON ingestion_state(agent_id)`,

	`CREATE TABLE IF NOT EXISTS memory_docs (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL REFERENCES agents(id),
		source_path TEXT NOT NULL,
		project_path TEXT,
		title TEXT,
		content TEXT,
		size_bytes INTEGER NOT NULL DEFAULT 0,
		mtime TEXT NOT NULL,
		checksum TEXT NOT NULL,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		UNIQUE(agent_id, source_path)
	)`,

	`CREATE TABLE IF NOT EXISTS timeline_items (
		id TEXT PRIMARY KEY,
		timestamp TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		session_id TEXT,
		memory_doc_id TEXT,
		kind TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT,
		metadata_json TEXT NOT NULL DEFAULT '{}'
	)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
		title, content, source_path,
		content='memory_docs',
		content_rowid='rowid'
	)`,

	`CREATE TRIGGER IF NOT EXISTS memory_ai AFTER INSERT ON memory_docs BEGIN
		INSERT INTO memory_fts(rowid, title, content, source_path) VALUES (new.rowid, new.title, new.content, new.source_path);
	END`,

	`CREATE TRIGGER IF NOT EXISTS memory_au AFTER UPDATE ON memory_docs BEGIN
		INSERT INTO memory_fts(memory_fts, rowid, title, content, source_path) VALUES('delete', old.rowid, old.title, old.content, old.source_path);
		INSERT INTO memory_fts(rowid, title, content, source_path) VALUES (new.rowid, new.title, new.content, new.source_path);
	END`,

	`CREATE INDEX IF NOT EXISTS idx_memory_agent ON memory_docs(agent_id)`,
	`CREATE INDEX IF NOT EXISTS idx_timeline_ts ON timeline_items(timestamp DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_timeline_agent ON timeline_items(agent_id)`,
	`CREATE INDEX IF NOT EXISTS idx_timeline_kind ON timeline_items(kind)`,
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	for i, m := range migrations {
		if _, err := db.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}
	return nil
}
