package claude

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nathanmauro/agent-observatory/internal/models"
)

const parserVersion = "claude-v1"

type ParseResult struct {
	SessionID    string
	Title        string
	Events       []models.SessionEvent
	StartedAt    time.Time
	UpdatedAt    time.Time
	UserMsgCount int
	OffsetBytes  int64
}

type rawRecord struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	UUID      string          `json:"uuid"`
	ParentUUID json.RawMessage `json:"parentUuid"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
	AITitle   string          `json:"aiTitle"`
	IsSidechain bool          `json:"isSidechain"`
	Cwd       string          `json:"cwd"`
	Version   string          `json:"version"`
	GitBranch string          `json:"gitBranch"`
}

type messageEnvelope struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	Model      string          `json:"model"`
	Usage      json.RawMessage `json:"usage"`
	StopReason *string         `json:"stop_reason"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

func ParseSession(r io.Reader, agentID string, sourcePath string, startOffset int64) (*ParseResult, error) {
	result := &ParseResult{
		OffsetBytes: startOffset,
	}

	if startOffset > 0 {
		if seeker, ok := r.(io.Seeker); ok {
			if _, err := seeker.Seek(startOffset, io.SeekStart); err != nil {
				return nil, fmt.Errorf("seek to offset %d: %w", startOffset, err)
			}
		} else {
			if _, err := io.CopyN(io.Discard, r, startOffset); err != nil {
				return nil, fmt.Errorf("skip to offset %d: %w", startOffset, err)
			}
		}
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	seq := 0
	if startOffset > 0 {
		seq = -1 // will be set properly; events from incremental parse get higher sequences
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		lineLen := int64(len(line)) + 1 // +1 for newline
		result.OffsetBytes += lineLen

		if len(line) == 0 {
			continue
		}

		var rec rawRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}

		if result.SessionID == "" && rec.SessionID != "" {
			result.SessionID = rec.SessionID
		}

		ts := parseTimestamp(rec.Timestamp)
		if !ts.IsZero() {
			if result.StartedAt.IsZero() || ts.Before(result.StartedAt) {
				result.StartedAt = ts
			}
			if ts.After(result.UpdatedAt) {
				result.UpdatedAt = ts
			}
		}

		lineHash := hashLine(line)

		switch rec.Type {
		case "ai-title":
			if rec.AITitle != "" {
				result.Title = rec.AITitle
			}

		case "user", "assistant":
			if rec.Message == nil {
				continue
			}
			var msg messageEnvelope
			if err := json.Unmarshal(rec.Message, &msg); err != nil {
				continue
			}

			role := msg.Role
			if role == "" {
				role = rec.Type
			}
			if role == "user" {
				result.UserMsgCount++
			}

			events := extractEvents(msg, role, rec, agentID, lineHash, ts, sourcePath)
			for i := range events {
				seq++
				events[i].Sequence = seq
			}
			result.Events = append(result.Events, events...)

		case "system":
			seq++
			result.Events = append(result.Events, models.SessionEvent{
				ID:          uuid.New().String(),
				AgentID:     agentID,
				Sequence:    seq,
				Timestamp:   ts,
				Role:        "system",
				Kind:        models.EventSystem,
				Content:     truncate(string(line), 10000),
				ContentHash: lineHash,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scanner: %w", err)
	}

	return result, nil
}

func extractEvents(msg messageEnvelope, role string, rec rawRecord, agentID, lineHash string, ts time.Time, sourcePath string) []models.SessionEvent {
	var blocks []contentBlock
	if len(msg.Content) > 0 && msg.Content[0] == '[' {
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			return nil
		}
	} else if len(msg.Content) > 0 && msg.Content[0] == '"' {
		var text string
		if err := json.Unmarshal(msg.Content, &text); err != nil {
			return nil
		}
		blocks = []contentBlock{{Type: "text", Text: text}}
	} else {
		return nil
	}

	var events []models.SessionEvent
	for i, block := range blocks {
		blockHash := fmt.Sprintf("%s:%d", lineHash, i)
		ev := models.SessionEvent{
			ID:          uuid.New().String(),
			AgentID:     agentID,
			Timestamp:   ts,
			Role:        role,
			ContentHash: blockHash,
		}

		if rec.IsSidechain {
			ev.FilesJSON = `{"sidechain":true}`
		}

		switch block.Type {
		case "text":
			ev.Kind = models.EventMessage
			ev.Content = block.Text
		case "thinking":
			ev.Kind = models.EventThinking
			ev.Content = block.Thinking
		case "tool_use":
			ev.Kind = models.EventToolUse
			ev.ToolName = block.Name
			if block.Input != nil {
				ev.ToolInputJSON = truncate(string(block.Input), 50000)
			}
		case "tool_result":
			ev.Kind = models.EventToolResult
			ev.ToolName = block.ToolUseID
			ev.ToolOutput = extractToolResultContent(block.Content)
		default:
			continue
		}

		events = append(events, ev)
	}

	return events
}

func extractToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return truncate(s, 50000)
		}
	}
	if raw[0] == '[' {
		var blocks []contentBlock
		if json.Unmarshal(raw, &blocks) == nil {
			var parts []string
			for _, b := range blocks {
				if b.Text != "" {
					parts = append(parts, b.Text)
				}
			}
			return truncate(strings.Join(parts, "\n"), 50000)
		}
	}
	return truncate(string(raw), 50000)
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}

func hashLine(line []byte) string {
	h := sha256.Sum256(line)
	return fmt.Sprintf("%x", h[:16])
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
