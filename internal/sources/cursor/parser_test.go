package cursor

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

	result, err := ParseSession(f, "agent-cursor", "/test/session.jsonl", 0)
	if err != nil {
		t.Fatal(err)
	}

	if result.Title != "Add unit tests for the parser" {
		t.Errorf("Title = %q, want %q", result.Title, "Add unit tests for the parser")
	}
	if result.UserMsgCount != 2 {
		t.Errorf("UserMsgCount = %d, want 2", result.UserMsgCount)
	}

	wantKinds := []models.EventKind{
		models.EventMessage,
		models.EventMessage,
		models.EventToolUse,
		models.EventToolResult,
		models.EventMessage,
		models.EventMessage,
	}
	if len(result.Events) != len(wantKinds) {
		t.Fatalf("got %d events, want %d", len(result.Events), len(wantKinds))
	}
	for i, ev := range result.Events {
		if ev.Kind != wantKinds[i] {
			t.Errorf("event[%d].Kind = %q, want %q", i, ev.Kind, wantKinds[i])
		}
	}

	if result.Events[0].Content == "" {
		t.Error("first user message should have content")
	}
	if result.Events[0].Role != "user" {
		t.Errorf("event[0].Role = %q, want user", result.Events[0].Role)
	}

	toolUse := result.Events[2]
	if toolUse.ToolName != "write_file" {
		t.Errorf("tool_use.ToolName = %q, want %q", toolUse.ToolName, "write_file")
	}

	toolResult := result.Events[3]
	if toolResult.ToolOutput != "File written successfully" {
		t.Errorf("tool_result.ToolOutput = %q", toolResult.ToolOutput)
	}
}

func TestParseSessionStripsUserQuery(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	result, err := ParseSession(f, "agent-cursor", "/test/session.jsonl", 0)
	if err != nil {
		t.Fatal(err)
	}

	firstMsg := result.Events[0].Content
	if firstMsg != "Add unit tests for the parser" {
		t.Errorf("user_query tags not stripped: got %q", firstMsg)
	}
}

func TestParseSessionSkipsRedacted(t *testing.T) {
	data := `{"role":"user","message":{"content":"[REDACTED]"}}
{"role":"assistant","message":{"content":"I can help with that."}}
`
	f, err := os.CreateTemp("", "cursor*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(data)
	f.Seek(0, 0)

	result, err := ParseSession(f, "agent-cursor", f.Name(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 1 {
		t.Errorf("got %d events, want 1 (redacted should be skipped)", len(result.Events))
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
		_, err := ParseSession(r, "agent-cursor", "/bench/session.jsonl", 0)
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
