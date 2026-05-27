package codex

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
	"github.com/nathanmauro/agent-observatory/internal/sources"
)

const ParserVersion = "codex-v1"

type rawRecord struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type sessionMeta struct {
	ID         string `json:"id"`
	Cwd        string `json:"cwd"`
	Originator string `json:"originator"`
	CLIVersion string `json:"cli_version"`
	Source     string `json:"source"`
}

type eventMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Phase   string `json:"phase"`
}

type responseItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Name    string          `json:"name"`
	CallID  string          `json:"call_id"`
	Content json.RawMessage `json:"content"`
	Output  string          `json:"output"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func ParseSession(r io.Reader, agentID string, sourcePath string, startOffset int64) (*sources.ParseResult, error) {
	result := &sources.ParseResult{
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
	var projectPath string
	var firstUserMsg string

	for scanner.Scan() {
		line := scanner.Bytes()
		lineLen := int64(len(line)) + 1
		result.OffsetBytes += lineLen

		if len(line) == 0 {
			continue
		}

		var rec rawRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
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
		case "session_meta":
			var meta sessionMeta
			if json.Unmarshal(rec.Payload, &meta) == nil {
				if result.SessionID == "" && meta.ID != "" {
					result.SessionID = meta.ID
				}
				if meta.Cwd != "" {
					projectPath = meta.Cwd
				}
			}

		case "event_msg":
			var msg eventMsg
			if json.Unmarshal(rec.Payload, &msg) == nil {
				switch msg.Type {
				case "user_message":
					result.UserMsgCount++
					if firstUserMsg == "" && msg.Message != "" {
						firstUserMsg = msg.Message
					}
					seq++
					result.Events = append(result.Events, models.SessionEvent{
						ID:          uuid.New().String(),
						AgentID:     agentID,
						Sequence:    seq,
						Timestamp:   ts,
						Role:        "user",
						Kind:        models.EventMessage,
						Content:     truncate(msg.Message, 10000),
						ContentHash: lineHash,
					})
				case "agent_message":
					seq++
					result.Events = append(result.Events, models.SessionEvent{
						ID:          uuid.New().String(),
						AgentID:     agentID,
						Sequence:    seq,
						Timestamp:   ts,
						Role:        "assistant",
						Kind:        models.EventMessage,
						Content:     truncate(msg.Message, 10000),
						ContentHash: lineHash,
					})
				case "task_started":
					seq++
					result.Events = append(result.Events, models.SessionEvent{
						ID:          uuid.New().String(),
						AgentID:     agentID,
						Sequence:    seq,
						Timestamp:   ts,
						Role:        "system",
						Kind:        models.EventMeta,
						Content:     "task_started",
						ContentHash: lineHash,
					})
				case "task_complete":
					seq++
					result.Events = append(result.Events, models.SessionEvent{
						ID:          uuid.New().String(),
						AgentID:     agentID,
						Sequence:    seq,
						Timestamp:   ts,
						Role:        "system",
						Kind:        models.EventMeta,
						Content:     "task_complete",
						ContentHash: lineHash,
					})
				}
			}

		case "response_item":
			var item responseItem
			if json.Unmarshal(rec.Payload, &item) == nil {
				switch item.Type {
				case "message":
					text := extractResponseText(item.Content)
					if text != "" {
						seq++
						result.Events = append(result.Events, models.SessionEvent{
							ID:          uuid.New().String(),
							AgentID:     agentID,
							Sequence:    seq,
							Timestamp:   ts,
							Role:        roleOrDefault(item.Role, "assistant"),
							Kind:        models.EventMessage,
							Content:     truncate(text, 10000),
							ContentHash: lineHash,
						})
					}
				case "function_call":
					seq++
					result.Events = append(result.Events, models.SessionEvent{
						ID:            uuid.New().String(),
						AgentID:       agentID,
						Sequence:      seq,
						Timestamp:     ts,
						Role:          "assistant",
						Kind:          models.EventToolUse,
						ToolName:      item.Name,
						ToolInputJSON: truncate(string(item.Content), 50000),
						ContentHash:   fmt.Sprintf("%s:fc", lineHash),
					})
				case "function_call_output":
					seq++
					result.Events = append(result.Events, models.SessionEvent{
						ID:          uuid.New().String(),
						AgentID:     agentID,
						Sequence:    seq,
						Timestamp:   ts,
						Role:        "tool",
						Kind:        models.EventToolResult,
						ToolName:    item.CallID,
						ToolOutput:  truncate(item.Output, 50000),
						ContentHash: fmt.Sprintf("%s:fco", lineHash),
					})
				case "reasoning":
					seq++
					result.Events = append(result.Events, models.SessionEvent{
						ID:          uuid.New().String(),
						AgentID:     agentID,
						Sequence:    seq,
						Timestamp:   ts,
						Role:        "assistant",
						Kind:        models.EventThinking,
						Content:     "[encrypted reasoning]",
						ContentHash: lineHash,
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scanner: %w", err)
	}

	if projectPath != "" {
		result.ProjectPath = projectPath
	}
	if firstUserMsg != "" {
		result.Title = cleanTitle(firstUserMsg)
	}

	return result, nil
}

func cleanTitle(s string) string {
	for strings.HasPrefix(s, "<") {
		end := strings.Index(s, ">")
		if end == -1 {
			break
		}
		tagEnd := end + 1
		closeTag := "</" + s[1:strings.IndexAny(s[1:], " >")+1] + ">"
		if idx := strings.Index(s[tagEnd:], closeTag); idx != -1 {
			s = strings.TrimSpace(s[tagEnd+idx+len(closeTag):])
		} else {
			s = strings.TrimSpace(s[tagEnd:])
		}
	}
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

func extractResponseText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '[' {
		var blocks []contentBlock
		if json.Unmarshal(raw, &blocks) == nil {
			for _, b := range blocks {
				if b.Type == "output_text" && b.Text != "" {
					return b.Text
				}
			}
		}
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
	}
	return ""
}

func roleOrDefault(role, def string) string {
	if role == "" {
		return def
	}
	return role
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
