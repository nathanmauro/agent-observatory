package cursor

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

const ParserVersion = "cursor-v1"

type rawRecord struct {
	Role    string          `json:"role"`
	Message messageEnvelope `json:"message"`
}

type messageEnvelope struct {
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
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
	now := time.Now().UTC()
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

		lineHash := hashLine(line)

		if rec.Role == "user" {
			result.UserMsgCount++
		}

		var blocks []contentBlock
		if len(rec.Message.Content) > 0 && rec.Message.Content[0] == '[' {
			if err := json.Unmarshal(rec.Message.Content, &blocks); err != nil {
				continue
			}
		} else if len(rec.Message.Content) > 0 && rec.Message.Content[0] == '"' {
			var text string
			if err := json.Unmarshal(rec.Message.Content, &text); err != nil {
				continue
			}
			blocks = []contentBlock{{Type: "text", Text: text}}
		} else {
			continue
		}

		for i, block := range blocks {
			blockHash := fmt.Sprintf("%s:%d", lineHash, i)
			ev := models.SessionEvent{
				ID:          uuid.New().String(),
				AgentID:     agentID,
				Timestamp:   now,
				Role:        rec.Role,
				ContentHash: blockHash,
			}

			switch block.Type {
			case "text":
				text := block.Text
				text = strings.TrimPrefix(text, "<user_query>\n")
				text = strings.TrimSuffix(text, "\n</user_query>")
				if strings.TrimSpace(text) == "" || text == "[REDACTED]" {
					continue
				}
				ev.Kind = models.EventMessage
				ev.Content = truncate(text, 10000)
				if firstUserMsg == "" && rec.Role == "user" {
					firstUserMsg = text
				}
			case "tool_use":
				ev.Kind = models.EventToolUse
				ev.ToolName = block.Name
				if block.Input != nil {
					ev.ToolInputJSON = truncate(string(block.Input), 50000)
				}
			case "tool_result":
				ev.Kind = models.EventToolResult
				if block.Text != "" {
					ev.ToolOutput = truncate(block.Text, 50000)
				}
			default:
				continue
			}

			seq++
			ev.Sequence = seq
			result.Events = append(result.Events, ev)
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scanner: %w", err)
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
