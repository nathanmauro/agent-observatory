package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/models"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	f, err := os.CreateTemp("", "observatory-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	d, err := Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenAndMigrate(t *testing.T) {
	d := testDB(t)

	var count int
	err := d.sql.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("sessions table not created")
	}
}

func TestAgentCRUD(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	agent := models.Agent{
		ID: "agent-1", Type: "claude", Name: "Claude Code",
		RootPath: "/home/user/.claude", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := d.UpsertAgent(ctx, agent); err != nil {
		t.Fatal(err)
	}

	agents, err := d.ListAgents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(agents))
	}
	if agents[0].ID != "agent-1" {
		t.Errorf("ID = %q, want agent-1", agents[0].ID)
	}
	if agents[0].Type != "claude" {
		t.Errorf("Type = %q, want claude", agents[0].Type)
	}

	agent.Name = "Claude Code v2"
	if err := d.UpsertAgent(ctx, agent); err != nil {
		t.Fatal(err)
	}
	agents, _ = d.ListAgents(ctx)
	if agents[0].Name != "Claude Code v2" {
		t.Errorf("upsert did not update name")
	}
}

func TestSessionCRUD(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d.UpsertAgent(ctx, models.Agent{ID: "agent-1", Type: "claude", Name: "Claude", CreatedAt: now, UpdatedAt: now})

	session := models.Session{
		ID: "sess-1", AgentID: "agent-1", ExternalID: "ext-1",
		SourcePath: "/test/session.jsonl", ProjectPath: "/home/user/project",
		Title: "Test session", Status: "active",
		StartedAt: now, UpdatedAt: now,
		MessageCount: 5,
	}
	if err := d.UpsertSession(ctx, session); err != nil {
		t.Fatal(err)
	}

	got, err := d.GetSession(ctx, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("session not found")
	}
	if got.Title != "Test session" {
		t.Errorf("Title = %q, want %q", got.Title, "Test session")
	}
	if got.MessageCount != 5 {
		t.Errorf("MessageCount = %d, want 5", got.MessageCount)
	}

	missing, err := d.GetSession(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Error("expected nil for missing session")
	}
}

func TestSessionListAndPagination(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	base := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	d.UpsertAgent(ctx, models.Agent{ID: "agent-1", Type: "claude", Name: "Claude", CreatedAt: base, UpdatedAt: base})

	for i := 0; i < 5; i++ {
		s := models.Session{
			ID: "sess-" + string(rune('a'+i)), AgentID: "agent-1",
			ExternalID: "ext-" + string(rune('a'+i)),
			SourcePath: "/test/" + string(rune('a'+i)) + ".jsonl",
			Title:  "Session " + string(rune('A'+i)),
			Status: "active", StartedAt: base, UpdatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if err := d.UpsertSession(ctx, s); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := d.ListSessions(ctx, "", "", "", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 3 {
		t.Fatalf("got %d sessions, want 3", len(resp.Data))
	}
	if resp.NextCursor == "" {
		t.Error("expected next_cursor for pagination")
	}

	resp2, err := d.ListSessions(ctx, "", "", "", 3, resp.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp2.Data) != 2 {
		t.Fatalf("page 2: got %d sessions, want 2", len(resp2.Data))
	}
}

func TestEventsInsertAndQuery(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d.UpsertAgent(ctx, models.Agent{ID: "a1", Type: "claude", Name: "Claude", CreatedAt: now, UpdatedAt: now})
	d.UpsertSession(ctx, models.Session{
		ID: "sess-1", AgentID: "a1", ExternalID: "e1", SourcePath: "/x",
		StartedAt: now, UpdatedAt: now,
	})

	events := []models.SessionEvent{
		{ID: "ev-1", SessionID: "sess-1", AgentID: "a1", Sequence: 1, Timestamp: now, Role: "user", Kind: models.EventMessage, Content: "Hello", ContentHash: "h1"},
		{ID: "ev-2", SessionID: "sess-1", AgentID: "a1", Sequence: 2, Timestamp: now, Role: "assistant", Kind: models.EventMessage, Content: "Hi there", ContentHash: "h2"},
		{ID: "ev-3", SessionID: "sess-1", AgentID: "a1", Sequence: 3, Timestamp: now, Role: "assistant", Kind: models.EventToolUse, ToolName: "Read", ContentHash: "h3"},
	}
	if err := d.InsertEvents(ctx, events); err != nil {
		t.Fatal(err)
	}

	resp, err := d.GetSessionEvents(ctx, "sess-1", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 3 {
		t.Fatalf("got %d events, want 3", len(resp.Data))
	}
	if resp.Data[0].Content != "Hello" {
		t.Errorf("event[0].Content = %q", resp.Data[0].Content)
	}

	resp2, err := d.GetSessionEvents(ctx, "sess-1", 10, "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp2.Data) != 2 {
		t.Fatalf("cursor query: got %d events, want 2", len(resp2.Data))
	}
}

func TestFTSSearch(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d.UpsertAgent(ctx, models.Agent{ID: "a1", Type: "claude", Name: "Claude", CreatedAt: now, UpdatedAt: now})
	d.UpsertSession(ctx, models.Session{
		ID: "sess-1", AgentID: "a1", ExternalID: "e1", SourcePath: "/x",
		Title: "authentication bug fix", StartedAt: now, UpdatedAt: now,
	})
	d.InsertEvent(ctx, models.SessionEvent{
		ID: "ev-1", SessionID: "sess-1", AgentID: "a1", Sequence: 1,
		Timestamp: now, Role: "user", Kind: models.EventMessage,
		Content: "Fix the login validation error", ContentHash: "h1",
	})

	results, err := d.Search(ctx, "authentication", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("FTS search returned no results")
	}
	found := false
	for _, r := range results {
		if r.Type == "session" && r.ID == "sess-1" {
			found = true
		}
	}
	if !found {
		t.Error("session not found in search results")
	}

	results2, err := d.Search(ctx, "login validation", 10)
	if err != nil {
		t.Fatal(err)
	}
	foundEvent := false
	for _, r := range results2 {
		if r.Type == "event" {
			foundEvent = true
		}
	}
	if !foundEvent {
		t.Error("event not found in FTS search")
	}
}

func TestMemoryDocCRUD(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d.UpsertAgent(ctx, models.Agent{ID: "a1", Type: "claude", Name: "Claude", CreatedAt: now, UpdatedAt: now})

	mem := models.MemoryDoc{
		ID: "mem-1", AgentID: "a1", SourcePath: "/home/.claude/CLAUDE.md",
		Title: "CLAUDE.md", Content: "# Project instructions\nUse Go for backend.",
		SizeBytes: 42, Mtime: now, Checksum: "abc123",
	}
	if err := d.UpsertMemoryDoc(ctx, mem); err != nil {
		t.Fatal(err)
	}

	got, err := d.GetMemoryDoc(ctx, "mem-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("memory doc not found")
	}
	if got.Content != mem.Content {
		t.Errorf("Content mismatch")
	}

	resp, err := d.ListMemoryDocs(ctx, "", "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("got %d memory docs, want 1", len(resp.Data))
	}

	results, err := d.Search(ctx, "instructions", 10)
	if err != nil {
		t.Fatal(err)
	}
	foundMem := false
	for _, r := range results {
		if r.Type == "memory" && r.ID == "mem-1" {
			foundMem = true
		}
	}
	if !foundMem {
		t.Error("memory doc not found in FTS search")
	}
}

func TestTimelineInsertAndList(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d.UpsertAgent(ctx, models.Agent{ID: "a1", Type: "claude", Name: "Claude", CreatedAt: now, UpdatedAt: now})

	items := []models.TimelineItem{
		{ID: "tl-1", Timestamp: now, AgentID: "a1", Kind: "session.created", Title: "New session"},
		{ID: "tl-2", Timestamp: now.Add(time.Minute), AgentID: "a1", Kind: "memory.updated", Title: "CLAUDE.md updated"},
		{ID: "tl-3", Timestamp: now.Add(2 * time.Minute), AgentID: "a1", Kind: "session.updated", Title: "Session update"},
	}
	for _, item := range items {
		if err := d.InsertTimelineItem(ctx, item); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := d.ListTimelineItems(ctx, "", "", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 3 {
		t.Fatalf("got %d timeline items, want 3", len(resp.Data))
	}
	if resp.Data[0].Title != "Session update" {
		t.Errorf("first item should be newest, got %q", resp.Data[0].Title)
	}

	resp2, err := d.ListTimelineItems(ctx, "", "memory.updated", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp2.Data) != 1 {
		t.Fatalf("kind filter: got %d, want 1", len(resp2.Data))
	}
}

func TestIngestionState(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	state := models.IngestionState{
		SourcePath: "/test/session.jsonl", AgentID: "a1",
		ParserVersion: "claude-v1", SizeBytes: 1024,
		Mtime: now, OffsetBytes: 512,
		LastIngestedAt: now,
	}
	if err := d.UpsertIngestion(ctx, state); err != nil {
		t.Fatal(err)
	}

	got, err := d.GetIngestion(ctx, "/test/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("ingestion state not found")
	}
	if got.OffsetBytes != 512 {
		t.Errorf("OffsetBytes = %d, want 512", got.OffsetBytes)
	}

	missing, err := d.GetIngestion(ctx, "/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Error("expected nil for missing ingestion state")
	}
}

func TestGetStats(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d.UpsertAgent(ctx, models.Agent{ID: "a1", Type: "claude", Name: "Claude", CreatedAt: now, UpdatedAt: now})
	d.UpsertAgent(ctx, models.Agent{ID: "a2", Type: "codex", Name: "Codex", CreatedAt: now, UpdatedAt: now})
	d.UpsertSession(ctx, models.Session{ID: "s1", AgentID: "a1", ExternalID: "e1", SourcePath: "/x", StartedAt: now, UpdatedAt: now})
	d.UpsertSession(ctx, models.Session{ID: "s2", AgentID: "a1", ExternalID: "e2", SourcePath: "/y", StartedAt: now, UpdatedAt: now})
	d.UpsertSession(ctx, models.Session{ID: "s3", AgentID: "a2", ExternalID: "e3", SourcePath: "/z", StartedAt: now, UpdatedAt: now})
	d.InsertEvent(ctx, models.SessionEvent{ID: "ev1", SessionID: "s1", AgentID: "a1", Sequence: 1, Timestamp: now, Kind: models.EventMessage, ContentHash: "h1"})
	d.UpsertMemoryDoc(ctx, models.MemoryDoc{ID: "m1", AgentID: "a1", SourcePath: "/mem", Title: "T", Mtime: now, Checksum: "c"})

	stats, err := d.GetStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", stats.TotalSessions)
	}
	if stats.TotalEvents != 1 {
		t.Errorf("TotalEvents = %d, want 1", stats.TotalEvents)
	}
	if stats.TotalMemory != 1 {
		t.Errorf("TotalMemory = %d, want 1", stats.TotalMemory)
	}
	if stats.AgentCounts["claude"] != 2 {
		t.Errorf("claude count = %d, want 2", stats.AgentCounts["claude"])
	}
	if stats.AgentCounts["codex"] != 1 {
		t.Errorf("codex count = %d, want 1", stats.AgentCounts["codex"])
	}
}

func BenchmarkFTSSearch(b *testing.B) {
	f, err := os.CreateTemp("", "observatory-bench-*.db")
	if err != nil {
		b.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	d, err := Open(f.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	d.UpsertAgent(ctx, models.Agent{ID: "a1", Type: "claude", Name: "Claude", CreatedAt: now, UpdatedAt: now})

	for i := 0; i < 100; i++ {
		id := "sess-" + time.Now().Format("150405.000") + string(rune('a'+i%26))
		d.UpsertSession(ctx, models.Session{
			ID: id, AgentID: "a1", ExternalID: id, SourcePath: "/bench/" + id,
			Title: "Session about authentication and database queries",
			StartedAt: now, UpdatedAt: now,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := d.Search(ctx, "authentication", 20)
		if err != nil {
			b.Fatal(err)
		}
	}
}
