package claude

import (
	"io"
	"os"
	"testing"

	"github.com/nathanmauro/agent-observatory/internal/models"
)

func TestParseSession(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	result, err := ParseSession(f, "agent-claude", "/test/session.jsonl", 0)
	if err != nil {
		t.Fatal(err)
	}

	if result.SessionID != "sess-abc-123" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "sess-abc-123")
	}
	if result.Title != "Fix authentication bug" {
		t.Errorf("Title = %q, want %q", result.Title, "Fix authentication bug")
	}
	if result.UserMsgCount != 1 {
		t.Errorf("UserMsgCount = %d, want 1", result.UserMsgCount)
	}
	if result.StartedAt.IsZero() {
		t.Error("StartedAt is zero")
	}
	if result.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
	if result.OffsetBytes == 0 {
		t.Error("OffsetBytes is 0")
	}

	wantKinds := []models.EventKind{
		models.EventSystem,
		models.EventMessage,
		models.EventThinking,
		models.EventMessage,
		models.EventToolUse,
		models.EventToolResult,
		models.EventMessage,
	}
	if len(result.Events) != len(wantKinds) {
		t.Fatalf("got %d events, want %d", len(result.Events), len(wantKinds))
	}
	for i, ev := range result.Events {
		if ev.Kind != wantKinds[i] {
			t.Errorf("event[%d].Kind = %q, want %q", i, ev.Kind, wantKinds[i])
		}
		if ev.AgentID != "agent-claude" {
			t.Errorf("event[%d].AgentID = %q, want %q", i, ev.AgentID, "agent-claude")
		}
		if ev.ID == "" {
			t.Errorf("event[%d].ID is empty", i)
		}
		if ev.ContentHash == "" {
			t.Errorf("event[%d].ContentHash is empty", i)
		}
	}

	thinkingEv := result.Events[2]
	if thinkingEv.Content == "" {
		t.Error("thinking event has empty content")
	}

	toolUseEv := result.Events[4]
	if toolUseEv.ToolName != "Read" {
		t.Errorf("tool_use.ToolName = %q, want %q", toolUseEv.ToolName, "Read")
	}

	toolResultEv := result.Events[5]
	if toolResultEv.ToolOutput == "" {
		t.Error("tool_result has empty ToolOutput")
	}
}

func TestParseSessionIncremental(t *testing.T) {
	f1, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	full, err := ParseSession(f1, "agent-claude", "/test/session.jsonl", 0)
	if err != nil {
		t.Fatal(err)
	}

	if full.OffsetBytes == 0 {
		t.Fatal("full parse returned 0 offset")
	}

	fi, _ := os.Stat("testdata/session.jsonl")
	if full.OffsetBytes != fi.Size() {
		t.Errorf("OffsetBytes = %d, want file size %d", full.OffsetBytes, fi.Size())
	}

	f2, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	incr, err := ParseSession(f2, "agent-claude", "/test/session.jsonl", full.OffsetBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(incr.Events) != 0 {
		t.Errorf("incremental parse from end returned %d events, want 0", len(incr.Events))
	}
}

func TestParseSessionEmpty(t *testing.T) {
	f, err := os.CreateTemp("", "empty*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	result, err := ParseSession(f, "agent-claude", f.Name(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 0 {
		t.Errorf("empty file produced %d events", len(result.Events))
	}
}

func BenchmarkParseSession(b *testing.B) {
	data, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		r := readerFromBytes(data)
		_, err := ParseSession(r, "agent-claude", "/bench/session.jsonl", 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

type bytesReader struct {
	data []byte
	pos  int
}

func readerFromBytes(data []byte) *bytesReader {
	return &bytesReader{data: append([]byte(nil), data...)}
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
