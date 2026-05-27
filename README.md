# Agent Observatory

Local dashboard for monitoring AI coding agents. Indexes sessions, events, and memory from Claude Code, OpenAI Codex, Augment, and Cursor into a single searchable interface.

## Quick start

```bash
make build
./agent-observatory
```

Opens at [http://127.0.0.1:3284](http://127.0.0.1:3284).

## Requirements

- Go 1.22+
- Node.js 20+

## Development

```bash
# Backend only (uses embedded static assets)
make dev

# Frontend dev server with hot reload (proxies API to :3284)
cd web && npm install && npm run dev
```

## Commands

```
make build      # Build frontend + backend
make dev        # Run backend in dev mode
make test       # Run all tests
make bench      # Run parser and search benchmarks
make install    # Install binary to ~/.local/bin/
make clean      # Remove build artifacts
```

## Configuration

CLI flags:

```
-addr     Listen address (default 127.0.0.1:3284)
-db       SQLite database path (default observatory.db)
-config   Config file path (default ~/.config/agent-observatory/config.json)
```

Optional config file (`~/.config/agent-observatory/config.json`):

```json
{
  "addr": "127.0.0.1:3284",
  "db_path": "observatory.db",
  "agents": {
    "claude":  { "enabled": true },
    "codex":   { "enabled": true },
    "augment": { "enabled": true },
    "cursor":  { "enabled": false }
  }
}
```

CLI flags override config file values.

## Architecture

- **Backend**: Go, SQLite (WAL mode) with FTS5 full-text search, WebSocket for real-time updates
- **Frontend**: SolidJS with @solidjs/router, dark theme
- **Parsers**: Per-agent adapters (JSONL for Claude/Codex/Cursor, JSON for Augment) with incremental parsing

See [SPEC.md](SPEC.md) for the full architecture and build plan.

## Pages

- **Overview** — stats, running processes, recent sessions, timeline
- **Sessions** — browsable session list with event detail view
- **Timeline** — chronological feed of session and memory changes
- **Memory** — browser for agent memory/config files (CLAUDE.md, Codex memories)
- **Search** — full-text search across sessions, events, and memory

## Privacy

All data stays local. Binds to localhost only. Never uploads data. Never mutates agent-owned files.
