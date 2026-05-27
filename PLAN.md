# Agent Observatory — Plan

## Current Phase

**Phase 1: MVP** (Claude-only, REST-only)

## Status: Built — ready to merge

## What's Done

- [x] Phase 0: format discovery (all 4 agents documented)
- [x] SPEC.md written and reviewed (4-agent orchestrated review, corrections applied)
- [x] Claude Code skill installed (~/.claude/skills/agent-observatory/)
- [x] Repo bootstrapped (github.com/nathanmauro/agent-observatory, public)
- [x] Stack decision: Go + SolidJS (intentional learning project)

### Phase 1 MVP (18 files, ~2180 lines)

#### Backend (Go — 1543 lines)
- [x] `go mod init` + dependencies (modernc.org/sqlite, chi, uuid)
- [x] Shared models (models.go) — Agent, Session, SessionEvent, IngestionState, SearchResult, pagination
- [x] SQLite migrations — agents, sessions, session_events, ingestion_state + FTS5 (sessions_fts, events_fts) + sync triggers + indexes
- [x] DB layer — prepared statements, UpsertAgent, UpsertSession, InsertEvents (batch), ListSessions (cursor pagination + FTS), Search (dual FTS5), GetIngestion
- [x] Claude Code JSONL parser — streaming line-by-line, byte-offset incremental, 4MB buffer, handles user/assistant/system/ai-title records, extracts text/tool_use/tool_result/thinking content blocks, SHA-256 content hashing, sidechain marking
- [x] Session discovery — walks ~/.claude/projects/**/*.jsonl, filesystem-probing project path decoder
- [x] Indexer — discovers sessions, checks ingestion state, incremental parse from last offset, deterministic session IDs (SHA-256 of source path), batch writes (500)
- [x] REST API (chi) — /health, /api/v1/agents, /api/v1/sessions, /api/v1/sessions/{id}, /api/v1/search, /api/v1/reindex + CORS + request logging
- [x] Static SolidJS embed via embed.FS with SPA fallback
- [x] main.go — flag parsing, graceful shutdown, sync initial index on startup

#### Frontend (SolidJS — 638 lines)
- [x] Vite + SolidJS + TypeScript scaffold
- [x] Typed API client with proxy config for dev
- [x] SessionList — paginated, searchable, cursor-based load more
- [x] SessionDetail — event rendering with kind-specific layouts (message, tool_use, tool_result, thinking, system)
- [x] SearchBar component
- [x] Dense dark dashboard CSS with role-colored event borders
- [x] Build → embed pipeline (Makefile)

#### Validated against real data
- 200+ sessions indexed from ~/.claude/projects/
- FTS search working across sessions and events
- Event rendering: message, tool_use, tool_result, thinking kinds
- Cursor pagination working
- Sub-second API responses

### Testing (deferred to post-merge)
- [ ] Redacted fixture files for Claude Code JSONL
- [ ] Parser golden tests
- [ ] API smoke tests

## What's Next

Phase 2: live runtime — process monitor, recursive fsnotify watcher, WebSocket broker, frontend live stores.

## Blockers

None.

## Session Log

| Date | Branch | Status | Summary |
|------|--------|--------|---------|
| 2026-05-27 | main | handed-off | Spec written (Codex gpt-5.5 enhanced), skill installed, repo bootstrapped, 4-agent orchestrated plan review, spec corrections applied. Ready for Phase 1 build. |
| 2026-05-27 | feature/phase-1-mvp | built | Phase 1 MVP complete. 3-agent parallel build (DB, parser, API) + manual frontend. Go backend parsing real Claude Code JSONL, SolidJS dashboard with session list/detail/search. Single binary with embedded frontend. |
