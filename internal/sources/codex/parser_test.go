package codex

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

	result, err := ParseSession(f, "agent-codex", "/test/session.jsonl", 0)
	if err != nil {
		t.Fatal(err)
	}

	if result.SessionID != "codex-sess-001" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "codex-sess-001")
	}
	if result.ProjectPath != "/home/user/api" {
		t.Errorf("ProjectPath = %q, want %q", result.ProjectPath, "/home/user/api")
	}
	if result.Title != "Add rate limiting to the API endpoints" {
		t.Errorf("Title = %q, want %q", result.Title, "Add rate limiting to the API endpoints")
	}
	if result.UserMsgCount != 1 {
		t.Errorf("UserMsgCount = %d, want 1", result.UserMsgCount)
	}
	if result.StartedAt.IsZero() {
		t.Error("StartedAt is zero")
	}

	wantKinds := []models.EventKind{
		models.EventMeta,
		models.EventMessage,
		models.EventMessage,
		models.EventToolUse,
		models.EventToolResult,
		models.EventThinking,
		models.EventMessage,
		models.EventMeta,
	}
	if len(result.Events) != len(wantKinds) {
		t.Fatalf("got %d events, want %d", len(result.Events), len(wantKinds))
	}
	for i, ev := range result.Events {
		if ev.Kind != wantKinds[i] {
			t.Errorf("event[%d].Kind = %q, want %q", i, ev.Kind, wantKinds[i])
		}
		if ev.AgentID != "agent-codex" {
			t.Errorf("event[%d].AgentID = %q, want %q", i, ev.AgentID, "agent-codex")
		}
	}

	fcEv := result.Events[3]
	if fcEv.ToolName != "write_file" {
		t.Errorf("function_call.ToolName = %q, want %q", fcEv.ToolName, "write_file")
	}

	fcoEv := result.Events[4]
	if fcoEv.ToolOutput != "File written successfully" {
		t.Errorf("function_call_output.ToolOutput = %q", fcoEv.ToolOutput)
	}

	reasoningEv := result.Events[5]
	if reasoningEv.Content != "[encrypted reasoning]" {
		t.Errorf("reasoning.Content = %q, want %q", reasoningEv.Content, "[encrypted reasoning]")
	}
}

func TestParseSessionIncremental(t *testing.T) {
	f1, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	full, err := ParseSession(f1, "agent-codex", "/test/session.jsonl", 0)
	if err != nil {
		t.Fatal(err)
	}

	f2, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	incr, err := ParseSession(f2, "agent-codex", "/test/session.jsonl", full.OffsetBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(incr.Events) != 0 {
		t.Errorf("incremental parse from end returned %d events, want 0", len(incr.Events))
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
		_, err := ParseSession(r, "agent-codex", "/bench/session.jsonl", 0)
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
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
