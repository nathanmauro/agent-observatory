# Agent Observatory — Plan

## Current Phase

**Phase 3: Multi-Agent Adapters** — Codex, Augment, Cursor

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

### Phase 3 Multi-Agent Adapters

#### Source interface + refactored indexer
- [x] Shared `sources.Source` interface with ParseResult, discovery, parsing, watch extensions
- [x] Refactored indexer to iterate over `[]sources.Source` instead of hardcoded Claude
- [x] Updated watcher to support multiple file extensions (.jsonl + .json)
- [x] Each source registers its own agent, roots, parser version, file extensions

#### Codex CLI adapter (`internal/sources/codex/`)
- [x] Discover sessions under `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`
- [x] JSONL parser: session_meta, event_msg (user_message, agent_message, task_started, task_complete), response_item (message, function_call, function_call_output, reasoning)
- [x] Incremental parsing by byte offset
- [x] Project path from session_meta.cwd
- [x] Title from first user message (with XML tag stripping)

#### Augment/Auggie adapter (`internal/sources/augment/`)
- [x] Discover sessions under `~/.augment/sessions/*.json`
- [x] Monolithic JSON parser: chatHistory[].exchange (request_message + response_text)
- [x] Full re-parse on change (no incremental — Augment rewrites entire file)
- [x] Changed files tracked per exchange
- [x] Title from customTitle or first non-system user message

#### Cursor adapter (`internal/sources/cursor/`)
- [x] Discover transcripts under `~/.cursor/projects/<encoded>/agent-transcripts/**/*.jsonl`
- [x] JSONL parser: Claude-like format (role + message.content blocks)
- [x] Tool use extraction (Shell, Glob, Grep, etc.)
- [x] Incremental parsing by byte offset
- [x] Project path decoder (greedy filesystem probing, like Claude)
- [x] Title from first user message (with XML/plugin_info stripping, user_query unwrapping)

#### Validated against real data
- Claude Code: 410 sessions, 32k+ events
- Codex CLI: 75 sessions, 16k events
- Augment (Auggie): 76 sessions, 583 events
- Cursor: 12 sessions, 755 events
- All four agents visible in API `/api/v1/agents`
- FTS search works across all agents

### Testing (deferred)
- [ ] Redacted fixture files per agent
- [ ] Parser golden tests
- [ ] API smoke tests

## What's Next

Phase 4: product UI — sessions page with agent filter, memory browser, unified search, timeline.

## Blockers

None.

## Session Log

| Date | Branch | Status | Summary |
|------|--------|--------|---------|
| 2026-05-27 | main | handed-off | Spec written (Codex gpt-5.5 enhanced), skill installed, repo bootstrapped, 4-agent orchestrated plan review, spec corrections applied. Ready for Phase 1 build. |
| 2026-05-27 | feature/phase-1-mvp | merged | Phase 1 MVP complete. Go backend parsing real Claude Code JSONL, SolidJS dashboard. |
| 2026-05-27 | feature/phase-2-live | merged | Phase 2: WebSocket broker, file watcher, process monitor. |
| 2026-05-27 | feature/phase-3-multi-agent | built | Phase 3: Source interface, Codex/Augment/Cursor adapters. All 4 agents indexed from real data. |
