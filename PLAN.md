# Agent Observatory — Plan

## Current Phase

**Phase 5: Hardening** — tests, benchmarks, config file, README

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

### Phase 4 Product UI (merged)

- [x] Multi-page routing, agent filter chips, memory browser, timeline, unified search
- [x] All 5 pages verified in headless browser with real data

### Phase 5 Hardening

#### Tests
- [x] Redacted fixture files per agent (Claude JSONL, Codex JSONL, Augment JSON, Cursor JSONL)
- [x] Parser golden tests: Claude (3 tests), Codex (2), Augment (3), Cursor (4)
- [x] DB tests: migrations, agent/session/event CRUD, pagination, FTS search, memory, timeline, ingestion state, stats (10 tests)
- [x] API smoke tests: all endpoints including 404s, search validation, limit clamping (14 tests)
- [x] 35 tests total, all passing

#### Benchmarks
- [x] Parser throughput: Claude 57 MB/s, Codex 62 MB/s, Augment 132 MB/s, Cursor 45 MB/s
- [x] FTS search latency: 215μs/op (100 sessions)

#### Config
- [x] JSON config file support (~/.config/agent-observatory/config.json)
- [x] Per-agent enable/disable, addr, db_path
- [x] CLI flags override config file values

#### Packaging
- [x] Makefile targets: build, dev, test, bench, install, clean
- [x] README with quickstart, commands, configuration, architecture, privacy

## What's Next

Done for now. Future ideas: Playwright E2E tests, goreleaser, Homebrew tap.

## Blockers

None.

## Session Log

| Date | Branch | Status | Summary |
|------|--------|--------|---------|
| 2026-05-27 | main | handed-off | Spec written (Codex gpt-5.5 enhanced), skill installed, repo bootstrapped, 4-agent orchestrated plan review, spec corrections applied. Ready for Phase 1 build. |
| 2026-05-27 | feature/phase-1-mvp | merged | Phase 1 MVP complete. Go backend parsing real Claude Code JSONL, SolidJS dashboard. |
| 2026-05-27 | feature/phase-2-live | merged | Phase 2: WebSocket broker, file watcher, process monitor. |
| 2026-05-27 | feature/phase-3-multi-agent | merged | Phase 3: Source interface, Codex/Augment/Cursor adapters. All 4 agents indexed from real data. |
| 2026-05-27 | feature/phase-4-product-ui | merged | Phase 4: Multi-page routing, agent chips, memory browser (523 files), timeline, unified search, stats dashboard. |
| 2026-05-27 | feature/phase-5-hardening | built | Phase 5: 35 tests + 5 benchmarks, config file, README, Makefile targets. All pages verified in headless browser. |
