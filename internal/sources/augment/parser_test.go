package augment

import (
	"io"
	"os"
	"testing"

	"github.com/nathanmauro/agent-observatory/internal/models"
)

func TestParseSession(t *testing.T) {
	f, err := os.Open("testdata/session.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	result, err := ParseSession(f, "agent-augment", "/test/session.json", 0)
	if err != nil {
		t.Fatal(err)
	}

	if result.SessionID != "augment-sess-001" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "augment-sess-001")
	}
	if result.Title != "Refactor database layer" {
		t.Errorf("Title = %q, want %q", result.Title, "Refactor database layer")
	}
	if result.UserMsgCount != 2 {
		t.Errorf("UserMsgCount = %d, want 2", result.UserMsgCount)
	}
	if result.StartedAt.IsZero() {
		t.Error("StartedAt is zero")
	}
	if result.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}

	wantKinds := []models.EventKind{
		models.EventMessage,
		models.EventMessage,
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

	if result.Events[0].Role != "user" {
		t.Errorf("event[0].Role = %q, want user", result.Events[0].Role)
	}
	if result.Events[1].Role != "assistant" {
		t.Errorf("event[1].Role = %q, want assistant", result.Events[1].Role)
	}

	if result.Events[1].FilesJSON == "" {
		t.Error("first assistant response should have changed files")
	}

	if result.OffsetBytes == 0 {
		t.Error("OffsetBytes is 0")
	}
}

func TestParseSessionNoTitle(t *testing.T) {
	data := `{
		"sessionId": "no-title-sess",
		"created": "2025-06-01T10:00:00.000Z",
		"modified": "2025-06-01T10:01:00.000Z",
		"chatHistory": [
			{
				"exchange": {
					"request_message": "Short task description here",
					"response_text": "Done.",
					"request_id": "r1"
				},
				"completed": true,
				"sequenceId": 1,
				"finishedAt": "2025-06-01T10:01:00.000Z",
				"changedFiles": [],
				"source": "chat"
			}
		]
	}`

	f, err := os.CreateTemp("", "augment*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(data)
	f.Seek(0, 0)

	result, err := ParseSession(f, "agent-augment", f.Name(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Short task description here" {
		t.Errorf("Title = %q, want derived from first message", result.Title)
	}
}

func BenchmarkParseSession(b *testing.B) {
	data, err := os.ReadFile("testdata/session.json")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		r := readerFromBytes(data)
		_, err := ParseSession(r, "agent-augment", "/bench/session.json", 0)
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
