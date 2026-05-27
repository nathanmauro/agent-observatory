# agent-observatory

![Status](https://img.shields.io/badge/status-pre--alpha-orange)

Real-time cross-agent monitoring dashboard for macOS. Tracks live processes, sessions, and memory files across Claude Code, Codex CLI, Augment, Cursor, and other AI agents.

## Stack

Go backend + SolidJS frontend + SQLite FTS5 + WebSocket live updates

## Development

```bash
# Backend
go run ./cmd/agent-observatory

# Frontend
cd web && npm run dev
```

See [SPEC.md](SPEC.md) for the full architecture and build plan.
