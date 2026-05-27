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
