package augment

import (
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

const ParserVersion = "augment-v1"

type sessionFile struct {
	SessionID           string        `json:"sessionId"`
	Created             string        `json:"created"`
	Modified            string        `json:"modified"`
	ChatHistory         []chatEntry   `json:"chatHistory"`
	CustomTitle         string        `json:"customTitle"`
	SubAgentCreditsUsed float64       `json:"subAgentCreditsUsed"`
	SubAgentCostUsd     float64       `json:"subAgentCostUsd"`
}

type chatEntry struct {
	Exchange            exchange `json:"exchange"`
	Completed           bool     `json:"completed"`
	SequenceID          int      `json:"sequenceId"`
	FinishedAt          string   `json:"finishedAt"`
	ChangedFiles        []string `json:"changedFiles"`
	ChangedFilesSkipped []string `json:"changedFilesSkipped"`
	Source              string   `json:"source"`
}

type exchange struct {
	RequestMessage string          `json:"request_message"`
	ResponseText   string          `json:"response_text"`
	RequestID      string          `json:"request_id"`
	RequestNodes   json.RawMessage `json:"request_nodes"`
	ResponseNodes  json.RawMessage `json:"response_nodes"`
}

// ParseSession reads a full Augment session JSON file.
// startOffset is ignored — Augment rewrites the entire file on every change.
func ParseSession(r io.Reader, agentID string, sourcePath string, _ int64) (*sources.ParseResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	title := sf.CustomTitle
	if title == "" {
		for _, entry := range sf.ChatHistory {
			msg := strings.TrimSpace(entry.Exchange.RequestMessage)
			if msg == "" {
				continue
			}
			if strings.HasPrefix(msg, "[") {
				if end := strings.Index(msg, "]"); end != -1 {
					msg = strings.TrimSpace(msg[end+1:])
				}
			}
			if strings.HasPrefix(msg, "<") || msg == "" {
				continue
			}
			if len(msg) > 80 {
				msg = msg[:80]
			}
			title = msg
			break
		}
	}

	result := &sources.ParseResult{
		SessionID:   sf.SessionID,
		Title:       title,
		OffsetBytes: int64(len(data)),
	}

	result.StartedAt = parseTimestamp(sf.Created)
	result.UpdatedAt = parseTimestamp(sf.Modified)

	seq := 0
	for _, entry := range sf.ChatHistory {
		ts := parseTimestamp(entry.FinishedAt)
		if ts.IsZero() {
			ts = result.StartedAt
		}

		if entry.Exchange.RequestMessage != "" {
			seq++
			result.UserMsgCount++
			result.Events = append(result.Events, models.SessionEvent{
				ID:          uuid.New().String(),
				AgentID:     agentID,
				Sequence:    seq,
				Timestamp:   ts,
				Role:        "user",
				Kind:        models.EventMessage,
				Content:     truncate(entry.Exchange.RequestMessage, 10000),
				ContentHash: hashString(fmt.Sprintf("user:%d:%s", entry.SequenceID, entry.Exchange.RequestID)),
			})
		}

		if entry.Exchange.ResponseText != "" {
			seq++

			var filesJSON string
			if len(entry.ChangedFiles) > 0 {
				if b, err := json.Marshal(entry.ChangedFiles); err == nil {
					filesJSON = string(b)
				}
			}

			result.Events = append(result.Events, models.SessionEvent{
				ID:          uuid.New().String(),
				AgentID:     agentID,
				Sequence:    seq,
				Timestamp:   ts,
				Role:        "assistant",
				Kind:        models.EventMessage,
				Content:     truncate(entry.Exchange.ResponseText, 10000),
				FilesJSON:   filesJSON,
				ContentHash: hashString(fmt.Sprintf("assistant:%d:%s", entry.SequenceID, entry.Exchange.RequestID)),
			})
		}
	}

	return result, nil
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			t, _ = time.Parse("2006-01-02T15:04:05.000Z", s)
		}
	}
	return t
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:16])
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
