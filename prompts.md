# Agent Observatory — Prompt Log

## 2026-05-27: Initial Prompt (user → Claude Code)

> i need to have a live view webapp of everything happening on my computer when it comes to agents.. i want to be able to see any processes that are occurring regardless of agent.. and i want to be able to search and view all memory files associated with each agent.. I understand that this might be able to tie in to my claude-sessions/agentseq dash.. i want to have a little bit of a start fresh approach here but maybe tie in to that..

## 2026-05-27: Enhanced Prompt (Claude Code → Codex gpt-5.5 xhigh → SPEC.md)

Original prompt sent to Codex for enhancement:

> Build a real-time cross-agent monitoring dashboard webapp that shows all AI agent processes running on macOS, lets you search and view memory files from each agent (Claude Code, Augment, Cursor, Codex, etc.), displays live session activity, and provides a unified view of agent orchestration across the system. Should be maximally performant with live WebSocket updates.

Context provided:
- 4 agents discovered: Claude Code (267MB), Codex CLI (1.6GB), Augment (28MB), Cursor (210MB)
- Existing agentseq project: FastAPI + React + SQLite FTS5 (Claude-only)
- Architecture decision: Go + SolidJS + SQLite FTS5 + WebSocket + fsnotify

Codex returned full spec saved to `SPEC.md` with:
- Detailed SQLite schema (6 tables + 3 FTS5 virtual tables)
- Source adapter interface contract
- Per-agent parsing strategies
- WebSocket protocol with seq-based replay
- REST API (13 endpoints, cursor pagination)
- SolidJS component tree
- Performance budgets
- 7-phase build orchestration plan
- Claude Code skill plan

## 2026-05-27: Architecture Decision

WASM: Alive but wrong tool for I/O-bound dashboard. Go + SolidJS chosen for:
- Go: single binary, goroutines for concurrent file watching, excellent websocket perf
- SolidJS: no virtual DOM, fine-grained reactivity = fastest DOM updates for live data
- SQLite FTS5: proven in agentseq, handles 2.2GB of agent data
