package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/models"
)

func (d *DB) ListAgents(ctx context.Context) ([]models.Agent, error) {
	rows, err := d.stmtListAgents.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		var enabled int
		var createdStr, updatedStr string
		if err := rows.Scan(&a.ID, &a.Type, &a.Name, &a.RootPath, &enabled, &a.MetadataJSON, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		a.Enabled = enabled != 0
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func (d *DB) GetSession(ctx context.Context, id string) (*models.Session, error) {
	var s models.Session
	var startedStr, updatedStr string
	err := d.stmtGetSession.QueryRowContext(ctx, id).Scan(
		&s.ID, &s.AgentID, &s.ExternalID, &s.SourcePath, &s.ProjectPath,
		&s.Title, &s.Status, &startedStr, &updatedStr,
		&s.MessageCount, &s.Summary, &s.MetadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.StartedAt, _ = time.Parse(time.RFC3339, startedStr)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &s, nil
}

func (d *DB) ListSessions(ctx context.Context, agentID, projectPath, query string, limit int, cursor string) (*models.PagedResponse[models.Session], error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if query != "" {
		q := `SELECT s.id, s.agent_id, s.external_id, s.source_path, s.project_path,
				s.title, s.status, s.started_at, s.updated_at, s.message_count, s.summary, s.metadata_json
			FROM sessions s
			JOIN sessions_fts f ON f.rowid = s.rowid
			WHERE sessions_fts MATCH ?`
		args := []any{query}

		if agentID != "" {
			q += " AND s.agent_id = ?"
			args = append(args, agentID)
		}
		if projectPath != "" {
			q += " AND s.project_path = ?"
			args = append(args, projectPath)
		}
		if cursor != "" {
			q += " AND s.updated_at < ?"
			args = append(args, cursor)
		}
		q += " ORDER BY s.updated_at DESC LIMIT ?"
		args = append(args, limit+1)
		rows, err = d.sql.QueryContext(ctx, q, args...)
	} else {
		q := `SELECT id, agent_id, external_id, source_path, project_path,
				title, status, started_at, updated_at, message_count, summary, metadata_json
			FROM sessions WHERE 1=1`
		args := []any{}

		if agentID != "" {
			q += " AND agent_id = ?"
			args = append(args, agentID)
		}
		if projectPath != "" {
			q += " AND project_path = ?"
			args = append(args, projectPath)
		}
		if cursor != "" {
			q += " AND updated_at < ?"
			args = append(args, cursor)
		}
		q += " ORDER BY updated_at DESC LIMIT ?"
		args = append(args, limit+1)
		rows, err = d.sql.QueryContext(ctx, q, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var startedStr, updatedStr string
		if err := rows.Scan(
			&s.ID, &s.AgentID, &s.ExternalID, &s.SourcePath, &s.ProjectPath,
			&s.Title, &s.Status, &startedStr, &updatedStr,
			&s.MessageCount, &s.Summary, &s.MetadataJSON,
		); err != nil {
			return nil, err
		}
		s.StartedAt, _ = time.Parse(time.RFC3339, startedStr)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	resp := &models.PagedResponse[models.Session]{Data: sessions}
	if len(sessions) > limit {
		resp.Data = sessions[:limit]
		resp.NextCursor = sessions[limit-1].UpdatedAt.Format(time.RFC3339)
	}
	return resp, nil
}

func (d *DB) GetSessionEvents(ctx context.Context, sessionID string, limit int, cursor string) (*models.PagedResponse[models.SessionEvent], error) {
	if limit <= 0 {
		limit = 100
	}

	q := `SELECT id, session_id, agent_id, sequence, timestamp, role, kind,
			content, tool_name, tool_input_json, tool_output, files_json, raw_json, content_hash
		FROM session_events WHERE session_id = ?`
	args := []any{sessionID}

	if cursor != "" {
		seq, err := strconv.Atoi(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		q += " AND sequence > ?"
		args = append(args, seq)
	}
	q += " ORDER BY sequence ASC LIMIT ?"
	args = append(args, limit+1)

	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.SessionEvent
	for rows.Next() {
		var e models.SessionEvent
		var tsStr string
		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.AgentID, &e.Sequence, &tsStr,
			&e.Role, &e.Kind, &e.Content, &e.ToolName, &e.ToolInputJSON,
			&e.ToolOutput, &e.FilesJSON, &e.RawJSON, &e.ContentHash,
		); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	resp := &models.PagedResponse[models.SessionEvent]{Data: events}
	if len(events) > limit {
		resp.Data = events[:limit]
		resp.NextCursor = strconv.Itoa(events[limit-1].Sequence)
	}
	return resp, nil
}

func (d *DB) Search(ctx context.Context, query string, limit int) ([]models.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	sessionQ := `SELECT s.id, a.type, s.title, s.project_path, s.summary, s.updated_at, f.rank
		FROM sessions_fts f
		JOIN sessions s ON f.rowid = s.rowid
		LEFT JOIN agents a ON s.agent_id = a.id
		WHERE sessions_fts MATCH ?
		ORDER BY f.rank
		LIMIT ?`

	eventQ := `SELECT e.session_id, a.type, e.tool_name, e.content, e.timestamp, f.rank
		FROM events_fts f
		JOIN session_events e ON f.rowid = e.rowid
		LEFT JOIN agents a ON e.agent_id = a.id
		WHERE events_fts MATCH ?
		ORDER BY f.rank
		LIMIT ?`

	var results []models.SearchResult

	sRows, err := d.sql.QueryContext(ctx, sessionQ, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search sessions: %w", err)
	}
	defer sRows.Close()

	for sRows.Next() {
		var r models.SearchResult
		var agentType, summary, updatedAt sql.NullString
		if err := sRows.Scan(&r.ID, &agentType, &r.Title, &r.Project, &summary, &updatedAt, &r.Rank); err != nil {
			return nil, err
		}
		r.Type = "session"
		r.AgentType = agentType.String
		r.Snippet = summary.String
		r.Timestamp = updatedAt.String
		results = append(results, r)
	}
	if err := sRows.Err(); err != nil {
		return nil, err
	}

	eRows, err := d.sql.QueryContext(ctx, eventQ, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search events: %w", err)
	}
	defer eRows.Close()

	for eRows.Next() {
		var r models.SearchResult
		var agentType, content, timestamp sql.NullString
		if err := eRows.Scan(&r.ID, &agentType, &r.Title, &content, &timestamp, &r.Rank); err != nil {
			return nil, err
		}
		r.Type = "event"
		r.AgentType = agentType.String
		r.Snippet = content.String
		r.Timestamp = timestamp.String
		results = append(results, r)
	}
	if err := eRows.Err(); err != nil {
		return nil, err
	}

	memQ := `SELECT m.id, a.type, m.title, m.source_path, m.project_path, m.mtime, f.rank
		FROM memory_fts f
		JOIN memory_docs m ON f.rowid = m.rowid
		LEFT JOIN agents a ON m.agent_id = a.id
		WHERE memory_fts MATCH ?
		ORDER BY f.rank
		LIMIT ?`

	mRows, err := d.sql.QueryContext(ctx, memQ, query, limit)
	if err == nil {
		defer mRows.Close()
		for mRows.Next() {
			var r models.SearchResult
			var agentType, path, project, mtime sql.NullString
			if err := mRows.Scan(&r.ID, &agentType, &r.Title, &path, &project, &mtime, &r.Rank); err == nil {
				r.Type = "memory"
				r.AgentType = agentType.String
				r.Path = path.String
				r.Project = project.String
				r.Timestamp = mtime.String
				results = append(results, r)
			}
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (d *DB) UpsertAgent(ctx context.Context, a models.Agent) error {
	_, err := d.stmtUpsertAgent.ExecContext(ctx,
		a.ID, a.Type, a.Name, a.RootPath, a.Enabled, a.MetadataJSON,
		a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339))
	return err
}

func (d *DB) UpsertSession(ctx context.Context, s models.Session) error {
	_, err := d.stmtUpsertSession.ExecContext(ctx,
		s.ID, s.AgentID, s.ExternalID, s.SourcePath, s.ProjectPath,
		s.Title, s.Status,
		s.StartedAt.Format(time.RFC3339), s.UpdatedAt.Format(time.RFC3339),
		s.MessageCount, s.Summary, s.MetadataJSON)
	return err
}

func (d *DB) InsertEvent(ctx context.Context, e models.SessionEvent) error {
	_, err := d.stmtInsertEvent.ExecContext(ctx,
		e.ID, e.SessionID, e.AgentID, e.Sequence, e.Timestamp.Format(time.RFC3339Nano),
		e.Role, e.Kind, e.Content, e.ToolName, e.ToolInputJSON,
		e.ToolOutput, e.FilesJSON, e.RawJSON, e.ContentHash)
	return err
}

func (d *DB) InsertEvents(ctx context.Context, events []models.SessionEvent) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt := tx.StmtContext(ctx, d.stmtInsertEvent)
	for _, e := range events {
		_, err := stmt.ExecContext(ctx,
			e.ID, e.SessionID, e.AgentID, e.Sequence, e.Timestamp.Format(time.RFC3339Nano),
			e.Role, e.Kind, e.Content, e.ToolName, e.ToolInputJSON,
			e.ToolOutput, e.FilesJSON, e.RawJSON, e.ContentHash)
		if err != nil {
			return fmt.Errorf("insert event %s: %w", e.ID, err)
		}
	}
	return tx.Commit()
}

func (d *DB) UpsertIngestion(ctx context.Context, s models.IngestionState) error {
	_, err := d.stmtUpsertIngestion.ExecContext(ctx,
		s.SourcePath, s.AgentID, s.ParserVersion,
		s.SizeBytes, s.Mtime.Format(time.RFC3339),
		s.Checksum, s.OffsetBytes,
		s.LastIngestedAt.Format(time.RFC3339), s.Error)
	return err
}

func (d *DB) UpsertMemoryDoc(ctx context.Context, m models.MemoryDoc) error {
	_, err := d.stmtUpsertMemory.ExecContext(ctx,
		m.ID, m.AgentID, m.SourcePath, m.ProjectPath,
		m.Title, m.Content, m.SizeBytes,
		m.Mtime.Format(time.RFC3339), m.Checksum, m.MetadataJSON)
	return err
}

func (d *DB) GetMemoryDoc(ctx context.Context, id string) (*models.MemoryDoc, error) {
	var m models.MemoryDoc
	var mtimeStr string
	err := d.stmtGetMemory.QueryRowContext(ctx, id).Scan(
		&m.ID, &m.AgentID, &m.SourcePath, &m.ProjectPath,
		&m.Title, &m.Content, &m.SizeBytes, &mtimeStr,
		&m.Checksum, &m.MetadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Mtime, _ = time.Parse(time.RFC3339, mtimeStr)
	return &m, nil
}

func (d *DB) ListMemoryDocs(ctx context.Context, agentID, projectPath, query string, limit int, cursor string) (*models.PagedResponse[models.MemoryDoc], error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if query != "" {
		q := `SELECT m.id, m.agent_id, m.source_path, m.project_path, m.title,
				m.content, m.size_bytes, m.mtime, m.checksum, m.metadata_json
			FROM memory_docs m
			JOIN memory_fts f ON f.rowid = m.rowid
			WHERE memory_fts MATCH ?`
		args := []any{query}
		if agentID != "" {
			q += " AND m.agent_id = ?"
			args = append(args, agentID)
		}
		if projectPath != "" {
			q += " AND m.project_path = ?"
			args = append(args, projectPath)
		}
		if cursor != "" {
			q += " AND m.mtime < ?"
			args = append(args, cursor)
		}
		q += " ORDER BY m.mtime DESC LIMIT ?"
		args = append(args, limit+1)
		rows, err = d.sql.QueryContext(ctx, q, args...)
	} else {
		q := `SELECT id, agent_id, source_path, project_path, title,
				content, size_bytes, mtime, checksum, metadata_json
			FROM memory_docs WHERE 1=1`
		args := []any{}
		if agentID != "" {
			q += " AND agent_id = ?"
			args = append(args, agentID)
		}
		if projectPath != "" {
			q += " AND project_path = ?"
			args = append(args, projectPath)
		}
		if cursor != "" {
			q += " AND mtime < ?"
			args = append(args, cursor)
		}
		q += " ORDER BY mtime DESC LIMIT ?"
		args = append(args, limit+1)
		rows, err = d.sql.QueryContext(ctx, q, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.MemoryDoc
	for rows.Next() {
		var m models.MemoryDoc
		var mtimeStr string
		if err := rows.Scan(
			&m.ID, &m.AgentID, &m.SourcePath, &m.ProjectPath,
			&m.Title, &m.Content, &m.SizeBytes, &mtimeStr,
			&m.Checksum, &m.MetadataJSON,
		); err != nil {
			return nil, err
		}
		m.Mtime, _ = time.Parse(time.RFC3339, mtimeStr)
		docs = append(docs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	resp := &models.PagedResponse[models.MemoryDoc]{Data: docs}
	if len(docs) > limit {
		resp.Data = docs[:limit]
		resp.NextCursor = docs[limit-1].Mtime.Format(time.RFC3339)
	}
	return resp, nil
}

func (d *DB) InsertTimelineItem(ctx context.Context, t models.TimelineItem) error {
	_, err := d.stmtInsertTimeline.ExecContext(ctx,
		t.ID, t.Timestamp.Format(time.RFC3339), t.AgentID,
		t.SessionID, t.MemoryDocID, t.Kind, t.Title, t.Body, t.MetadataJSON)
	return err
}

func (d *DB) ListTimelineItems(ctx context.Context, agentID, kind string, limit int, cursor string) (*models.PagedResponse[models.TimelineItem], error) {
	if limit <= 0 {
		limit = 50
	}

	q := `SELECT t.id, t.timestamp, t.agent_id, COALESCE(a.type, ''), t.session_id, t.memory_doc_id,
			t.kind, t.title, t.body, t.metadata_json
		FROM timeline_items t
		LEFT JOIN agents a ON t.agent_id = a.id
		WHERE 1=1`
	args := []any{}
	if agentID != "" {
		q += " AND t.agent_id = ?"
		args = append(args, agentID)
	}
	if kind != "" {
		q += " AND t.kind = ?"
		args = append(args, kind)
	}
	if cursor != "" {
		q += " AND t.timestamp < ?"
		args = append(args, cursor)
	}
	q += " ORDER BY t.timestamp DESC LIMIT ?"
	args = append(args, limit+1)

	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.TimelineItem
	for rows.Next() {
		var t models.TimelineItem
		var tsStr string
		if err := rows.Scan(
			&t.ID, &tsStr, &t.AgentID, &t.AgentType, &t.SessionID, &t.MemoryDocID,
			&t.Kind, &t.Title, &t.Body, &t.MetadataJSON,
		); err != nil {
			return nil, err
		}
		t.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	resp := &models.PagedResponse[models.TimelineItem]{Data: items}
	if len(items) > limit {
		resp.Data = items[:limit]
		resp.NextCursor = items[limit-1].Timestamp.Format(time.RFC3339)
	}
	return resp, nil
}

func (d *DB) GetStats(ctx context.Context) (*models.Stats, error) {
	stats := &models.Stats{AgentCounts: make(map[string]int)}

	d.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&stats.TotalSessions)
	d.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM session_events").Scan(&stats.TotalEvents)
	d.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM memory_docs").Scan(&stats.TotalMemory)

	rows, err := d.sql.QueryContext(ctx, `SELECT a.type, COUNT(s.id) FROM sessions s JOIN agents a ON s.agent_id = a.id GROUP BY a.type`)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()
	for rows.Next() {
		var agentType string
		var count int
		if err := rows.Scan(&agentType, &count); err == nil {
			stats.AgentCounts[agentType] = count
		}
	}
	return stats, nil
}

func (d *DB) GetIngestion(ctx context.Context, sourcePath string) (*models.IngestionState, error) {
	row := d.stmtGetIngestion.QueryRowContext(ctx, sourcePath)
	var s models.IngestionState
	var mtimeStr, lastStr string
	var errStr sql.NullString
	var checksum sql.NullString
	var offset sql.NullInt64
	err := row.Scan(&s.SourcePath, &s.AgentID, &s.ParserVersion,
		&s.SizeBytes, &mtimeStr, &checksum, &offset, &lastStr, &errStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Mtime, _ = time.Parse(time.RFC3339, mtimeStr)
	s.LastIngestedAt, _ = time.Parse(time.RFC3339, lastStr)
	if checksum.Valid {
		s.Checksum = checksum.String
	}
	s.OffsetBytes = offset.Int64
	if errStr.Valid {
		s.Error = errStr.String
	}
	return &s, nil
}
