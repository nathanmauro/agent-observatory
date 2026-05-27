package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/db"
	"github.com/nathanmauro/agent-observatory/internal/events"
	"github.com/nathanmauro/agent-observatory/internal/indexer"
	"github.com/nathanmauro/agent-observatory/internal/models"
	"github.com/nathanmauro/agent-observatory/internal/processes"
	"github.com/nathanmauro/agent-observatory/internal/ws"
)

func testServer(t *testing.T) (*httptest.Server, *db.DB) {
	t.Helper()
	f, err := os.CreateTemp("", "observatory-api-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	database, err := db.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })

	bus := events.NewBus()
	ix := indexer.New(database, bus, nil)
	broker := ws.NewBroker(bus)
	mon := processes.NewMonitor(bus)

	router := NewRouter(database, ix, broker, mon)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, database
}

func seedData(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	database.UpsertAgent(ctx, models.Agent{ID: "a1", Type: "claude", Name: "Claude Code", CreatedAt: now, UpdatedAt: now})
	database.UpsertSession(ctx, models.Session{
		ID: "sess-1", AgentID: "a1", ExternalID: "e1", SourcePath: "/test/s.jsonl",
		ProjectPath: "/home/user/project", Title: "Test session",
		Status: "active", StartedAt: now, UpdatedAt: now, MessageCount: 3,
	})
	database.InsertEvents(ctx, []models.SessionEvent{
		{ID: "ev-1", SessionID: "sess-1", AgentID: "a1", Sequence: 1, Timestamp: now, Role: "user", Kind: models.EventMessage, Content: "Hello world", ContentHash: "h1"},
		{ID: "ev-2", SessionID: "sess-1", AgentID: "a1", Sequence: 2, Timestamp: now, Role: "assistant", Kind: models.EventMessage, Content: "Hi there", ContentHash: "h2"},
	})
	database.UpsertMemoryDoc(ctx, models.MemoryDoc{
		ID: "mem-1", AgentID: "a1", SourcePath: "/claude.md",
		Title: "CLAUDE.md", Content: "project instructions", SizeBytes: 20, Mtime: now, Checksum: "abc",
	})
	database.InsertTimelineItem(ctx, models.TimelineItem{
		ID: "tl-1", Timestamp: now, AgentID: "a1", Kind: "session.created", Title: "New session",
	})
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := testServer(t)

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestListAgents(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/agents")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var agents []models.Agent
	json.NewDecoder(resp.Body).Decode(&agents)
	if len(agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(agents))
	}
}

func TestListSessions(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/sessions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result struct {
		Data       []models.Session `json:"data"`
		NextCursor string           `json:"next_cursor"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Data) != 1 {
		t.Fatalf("got %d sessions, want 1", len(result.Data))
	}
}

func TestGetSession(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/sessions/sess-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var result struct {
		Session models.Session       `json:"session"`
		Events  []models.SessionEvent `json:"events"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Session.Title != "Test session" {
		t.Errorf("title = %q", result.Session.Title)
	}
	if len(result.Events) != 2 {
		t.Errorf("got %d events, want 2", len(result.Events))
	}
}

func TestGetSessionNotFound(t *testing.T) {
	srv, _ := testServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/sessions/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestSearchEndpoint(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/search?q=Hello")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var results []models.SearchResult
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) == 0 {
		t.Error("search returned no results")
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	srv, _ := testServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestListMemory(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/memory")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []models.MemoryDoc `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Data) != 1 {
		t.Fatalf("got %d memory docs, want 1", len(result.Data))
	}
}

func TestGetMemory(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/memory/mem-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestGetMemoryNotFound(t *testing.T) {
	srv, _ := testServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/memory/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestListTimeline(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/timeline")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []models.TimelineItem `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Data) != 1 {
		t.Fatalf("got %d timeline items, want 1", len(result.Data))
	}
}

func TestGetStats(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats models.Stats
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", stats.TotalSessions)
	}
	if stats.TotalEvents != 2 {
		t.Errorf("TotalEvents = %d, want 2", stats.TotalEvents)
	}
}

func TestListProcesses(t *testing.T) {
	srv, _ := testServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/processes")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestLimitClamping(t *testing.T) {
	srv, database := testServer(t)
	seedData(t, database)

	resp, err := http.Get(srv.URL + "/api/v1/sessions?limit=999")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
