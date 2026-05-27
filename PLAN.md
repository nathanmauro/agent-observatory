# Agent Observatory — Plan

## Current Phase

**Phase 1: MVP** (Claude-only, REST-only)

## Status: Ready to Build

## What's Done

- [x] Phase 0: format discovery (all 4 agents documented)
- [x] SPEC.md written and reviewed (4-agent orchestrated review, corrections applied)
- [x] Claude Code skill installed (~/.claude/skills/agent-observatory/)
- [x] Repo bootstrapped (github.com/nathanmauro/agent-observatory, public)
- [x] Stack decision: Go + SolidJS (intentional learning project)

## What's Next

Phase 1 MVP scope (~20 files, ~830 lines):

### Backend (Go)
- [ ] `go mod init` + dependencies (modernc.org/sqlite, chi router)
- [ ] SQLite migrations (full schema: agents, sessions, session_events, memory_docs, timeline_items, ingestion_state + FTS5)
- [ ] Claude Code JSONL parser (line-by-line, byte-offset incremental, mixed record types)
- [ ] Indexer (batch writes, ingestion state tracking)
- [ ] REST API (5 endpoints: health, agents, sessions, sessions/{id}, reindex)
- [ ] Static SolidJS embed via Go embed.FS

### Frontend (SolidJS)
- [ ] Vite + SolidJS project scaffold
- [ ] SessionList page (paginated, searchable)
- [ ] SessionDetail page (event rendering)
- [ ] SearchBar component (FTS queries)
- [ ] API client (typed fetch helpers)

### Testing
- [ ] Redacted fixture files for Claude Code JSONL
- [ ] Parser golden tests
- [ ] API smoke tests

## Blockers

None.

## Session Log

| Date | Branch | Status | Summary |
|------|--------|--------|---------|
| 2026-05-27 | main | handed-off | Spec written (Codex gpt-5.5 enhanced), skill installed, repo bootstrapped, 4-agent orchestrated plan review, spec corrections applied. Ready for Phase 1 build. |
