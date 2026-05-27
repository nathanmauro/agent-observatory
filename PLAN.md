# Agent Observatory — Plan

## Current Phase

**Phase 4: Product UI** — multi-page routing, agent filter chips, memory browser, timeline, unified search

## Status: Built — ready to merge

## What's Done

- [x] Phase 0: format discovery (all 4 agents documented)
- [x] SPEC.md written and reviewed (4-agent orchestrated review, corrections applied)
- [x] Claude Code skill installed (~/.claude/skills/agent-observatory/)
- [x] Repo bootstrapped (github.com/nathanmauro/agent-observatory, public)
- [x] Stack decision: Go + SolidJS (intentional learning project)

### Phase 1 MVP (merged)

- [x] Go backend with SQLite + FTS5, Claude Code JSONL parser, REST API
- [x] SolidJS dashboard with session list/detail/search, dark theme
- [x] Single binary with embedded frontend

### Phase 2 Live Runtime (merged)

- [x] WebSocket broker for real-time updates
- [x] Recursive fsnotify file watcher
- [x] Process monitor for running agent detection

### Phase 3 Multi-Agent Adapters (merged)

- [x] Shared Source interface, refactored indexer
- [x] Codex CLI, Augment/Auggie, and Cursor adapters
- [x] All 4 agents indexed from real data (600 sessions, 52k events)

### Phase 4 Product UI

#### Backend
- [x] `memory_docs` table with FTS5 search, triggers, indexes
- [x] `timeline_items` table with indexes for timestamp, agent, kind
- [x] Memory discovery: Claude (CLAUDE.md, settings, project memory), Codex (520 memory files from ~/.codex/memories/)
- [x] Memory indexing with checksum-based change detection
- [x] Timeline generation from session creation/updates and memory changes
- [x] API endpoints: GET /memory, GET /memory/{id}, GET /timeline, GET /stats
- [x] Enhanced unified search across sessions, events, AND memory docs
- [x] Stats endpoint with per-agent session counts

#### Frontend
- [x] @solidjs/router with 5 routes (/, /sessions/:id?, /timeline, /memory/:id?, /search)
- [x] App shell with top nav bar, agent filter chips, connection status, reindex button
- [x] Agent filter chips (toggle per agent, filters all pages)
- [x] OverviewPage: stat cards, running processes, recent sessions, timeline preview
- [x] SessionsPage: sidebar session list with search + detail pane, agent dots per session
- [x] TimelinePage: chronological feed grouped by date, linked to sessions/memory
- [x] MemoryPage: sidebar file list with search + content viewer
- [x] SearchPage: unified search across sessions, events, memory with type grouping
- [x] WebSocket subscriptions for memory + timeline topics
- [x] Full dark theme CSS for all new pages and components

#### Validated against real data
- 600 sessions across 4 agents (claude 433, codex 79, auggie 76, cursor 12)
- 52k+ events indexed
- 523 memory files indexed (Claude + Codex)
- All API endpoints returning correct data
- Frontend builds clean (TypeScript + Vite)

### Testing (deferred)
- [ ] Redacted fixture files per agent
- [ ] Parser golden tests
- [ ] API smoke tests

## What's Next

Phase 5: hardening — benchmarks, Playwright smoke tests, packaging, config file support, launch instructions.

## Blockers

None.

## Session Log

| Date | Branch | Status | Summary |
|------|--------|--------|---------|
| 2026-05-27 | main | handed-off | Spec written (Codex gpt-5.5 enhanced), skill installed, repo bootstrapped, 4-agent orchestrated plan review, spec corrections applied. Ready for Phase 1 build. |
| 2026-05-27 | feature/phase-1-mvp | merged | Phase 1 MVP complete. Go backend parsing real Claude Code JSONL, SolidJS dashboard. |
| 2026-05-27 | feature/phase-2-live | merged | Phase 2: WebSocket broker, file watcher, process monitor. |
| 2026-05-27 | feature/phase-3-multi-agent | merged | Phase 3: Source interface, Codex/Augment/Cursor adapters. All 4 agents indexed from real data. |
| 2026-05-27 | feature/phase-4-product-ui | built | Phase 4: Multi-page routing, agent chips, memory browser (523 files), timeline, unified search, stats dashboard. |
