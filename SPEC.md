# PROJECT: Agent Observatory

Build a production-quality local dashboard for monitoring AI coding agents on macOS in real time.

## Product Goal

Agent Observatory is a local-first, read-only monitoring dashboard for AI agent activity across Claude Code, Codex CLI, Augment/Auggie, Cursor, Copilot, and future agents. It must show live processes, unified sessions, memory files, search, and a cross-agent activity timeline from multiple storage formats.

The system must start fresh, but may borrow concepts from the existing `agentseq` / `clone-sessions` FastAPI + React + SQLite FTS5 Claude-only dashboard.

## Core Architecture

Use:

- Backend: Go single binary
- Frontend: SolidJS
- Storage: SQLite with WAL and FTS5
- Transport: REST for initial loads and search, WebSocket for live deltas
- File watching: `fsnotify` with recursive watch management
- Runtime: localhost-only by default, macOS optimized, no cloud dependency

The backend owns all parsing, indexing, process monitoring, and WebSocket fanout. The frontend is a thin reactive client with normalized state, virtualized lists, and fast filters.

## Data Flow

1. Source discovery finds enabled agent roots:
   - Claude Code: `~/.claude/`
   - Codex CLI: `~/.codex/`
   - Augment/Auggie: `~/.augment/`
   - Cursor: `~/.cursor/` and relevant `~/Library/Application Support/Cursor/` paths
   - Copilot/editor data where discoverable

2. Source adapters scan files and databases.

3. Parsers stream raw records into normalized internal events.

4. Indexer writes sessions, events, memory docs, and timeline rows into SQLite in batched transactions.

5. FTS5 indexes searchable session text, event text, tool calls, memory files, paths, and summaries.

6. File watcher detects changes, debounces writes, parses only changed data, updates SQLite, and emits domain events.

7. WebSocket broker sends ordered deltas to connected clients.

8. SolidJS frontend loads initial state through REST, then applies WebSocket deltas.

## Backend Package Shape

Use this Go layout:

```text
cmd/agent-observatory/
internal/config/
internal/db/
internal/migrations/
internal/sources/
internal/sources/claude/
internal/sources/codex/
internal/sources/augment/
internal/sources/cursor/
internal/processes/
internal/watch/
internal/indexer/
internal/search/
internal/timeline/
internal/api/
internal/ws/
web/
```

Define a source adapter interface:

```go
type Source interface {
    AgentType() AgentType
    Discover(ctx context.Context) ([]Root, error)
    Scan(ctx context.Context, root Root, sink Sink) error
    WatchPaths(root Root) ([]string, error)
    ParseChanged(ctx context.Context, path string, sink Sink) error
}
```

All adapters must be read-only against agent-owned directories.

## SQLite Model

Use stable internal IDs and preserve raw external IDs.

Core tables:

```sql
agents(
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  name TEXT NOT NULL,
  root_path TEXT NOT NULL,
  enabled INTEGER NOT NULL,
  metadata_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

sessions(
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  external_id TEXT,
  source_path TEXT,
  project_path TEXT,
  title TEXT,
  status TEXT,
  started_at TEXT,
  updated_at TEXT,
  message_count INTEGER NOT NULL DEFAULT 0,
  summary TEXT,
  metadata_json TEXT NOT NULL,
  UNIQUE(agent_id, external_id, source_path)
);

session_events(
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  timestamp TEXT,
  role TEXT,
  kind TEXT NOT NULL,
  content TEXT,
  tool_name TEXT,
  tool_input_json TEXT,
  tool_output TEXT,
  files_json TEXT NOT NULL,
  raw_json TEXT,
  content_hash TEXT NOT NULL,
  UNIQUE(session_id, sequence, content_hash)
);

memory_docs(
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  source_path TEXT NOT NULL,
  project_path TEXT,
  title TEXT,
  content TEXT,
  size_bytes INTEGER NOT NULL,
  mtime TEXT NOT NULL,
  checksum TEXT NOT NULL,
  metadata_json TEXT NOT NULL,
  UNIQUE(agent_id, source_path)
);

timeline_items(
  id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  session_id TEXT,
  memory_doc_id TEXT,
  process_id TEXT,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT,
  metadata_json TEXT NOT NULL
);

ingestion_state(
  source_path TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  parser_version TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  mtime TEXT NOT NULL,
  checksum TEXT,
  offset_bytes INTEGER,
  last_ingested_at TEXT NOT NULL,
  error TEXT
);
```

Add FTS5 tables for `sessions`, `session_events`, and `memory_docs`. Use external-content FTS where practical. Enable `journal_mode=WAL`, `synchronous=NORMAL`, and prepared statements.

## Agent Parsing Strategy

First implementation step: verify real local formats with `find`, `jq`, `sqlite3`, `file`, and small sampled fixtures. Do not assume undocumented schemas are stable.

Claude Code:
- Parse `~/.claude/projects/**/*.jsonl` as append-only JSONL sessions.
- Stream line by line.
- Tolerate partial final lines.
- Normalize user, assistant, tool use, tool result, system, and summary records.
- Treat project path encoded in the Claude project directory as metadata.
- Index `CLAUDE.md`, project memory files, commands, settings, and relevant markdown knowledge files.

Codex CLI:
- Discover JSONL sessions, SQLite logs, config, and memory folders under `~/.codex/`.
- Parse SQLite with schema probing, not hardcoded assumptions only.
- Index git-versioned memory files under `~/.codex/memories/`.
- Preserve rollout/session IDs where present.
- Treat database rows and JSONL rows as possible duplicate sources; deduplicate by content hash plus timestamp plus source path.

Augment/Auggie:
- Parse `~/.augment/sessions/*.json` style session files and checkpoint files.
- Extract messages, sub-agent metadata, tool activity, project paths, credit fields when present, and checkpoints as timeline events.
- Expect single JSON files, not JSONL.
- Handle schema drift by keeping unknown fields in `metadata_json`.

Cursor:
- Discover project-based data under `.cursor` and Cursor application support storage.
- Probe SQLite databases such as workspace state DBs before parsing.
- Normalize chats, composer sessions, project roots, and editor agent metadata when present.
- Cursor storage is the least stable source; build the adapter defensively and flag unsupported schemas in the UI.

Processes:
- Use `gopsutil` or native macOS process inspection.
- Poll every 1 second by default.
- Detect `claude`, `codex`, `auggie`, `cursor`, `Cursor`, `copilot`, editor extension hosts, and child processes.
- Capture PID, parent PID, command, args, cwd when available, CPU, RSS memory, start time, and status.
- Emit diffs only when values materially change.

## File Watching

`fsnotify` is not recursive. Build a recursive watch registry:

- Walk enabled roots at startup.
- Add watchers for directories.
- Add new watchers when directories appear.
- Debounce per path for 150-500ms.
- Coalesce rapid writes into one parse job.
- Use a bounded worker pool for parsing.
- Use backpressure: if the queue grows, collapse multiple file events into a source-level rescan.
- Handle rename, atomic replace, delete, partial writes, and permission errors.
- Never block the watcher goroutine on parsing or SQLite writes.

## REST API

Expose versioned routes under `/api/v1`.

Required endpoints:

```text
GET  /health
GET  /api/v1/agents
GET  /api/v1/processes
GET  /api/v1/sessions?agent=&project=&q=&from=&to=&limit=&cursor=
GET  /api/v1/sessions/{id}
GET  /api/v1/sessions/{id}/events?limit=&cursor=
GET  /api/v1/memory?agent=&project=&q=&limit=&cursor=
GET  /api/v1/memory/{id}
GET  /api/v1/search?q=&type=&agent=&from=&to=&limit=&cursor=
GET  /api/v1/timeline?agent=&kind=&from=&to=&limit=&cursor=
GET  /api/v1/stats
POST /api/v1/reindex
```

Search must return matched entity type, title, path/project, timestamp, snippet, rank, and agent.

Use cursor pagination everywhere. Do not use offset pagination for large tables.

## WebSocket Protocol

Endpoint:

```text
GET /api/v1/ws
```

Client messages:

```json
{"type":"hello","last_seq":123}
{"type":"subscribe","topics":["processes","sessions","memory","timeline","indexer"]}
{"type":"unsubscribe","topics":["memory"]}
{"type":"ping","client_time":"2026-05-27T10:00:00-04:00"}
```

Server envelope:

```json
{
  "seq": 124,
  "schema_version": 1,
  "type": "session.updated",
  "topic": "sessions",
  "sent_at": "2026-05-27T14:00:00Z",
  "data": {}
}
```

Required server message types:

```text
hello
heartbeat
process.snapshot
process.diff
session.created
session.updated
session.deleted
event.appended
memory.created
memory.updated
memory.deleted
timeline.appended
indexer.progress
indexer.error
error
```

WebSocket rules:

- Initial data comes from REST.
- WebSocket sends deltas only.
- Every message has a monotonic `seq`.
- Client reconnects with `last_seq`.
- Server keeps a short replay buffer.
- If replay is unavailable, server sends `resync_required`.

## SolidJS Frontend

Use a dense dashboard layout, not a marketing page.

Component structure:

```text
App
  AppShell
    SidebarAgentFilter
    TopSearchBar
    ConnectionStatus
    MainRoutes
      OverviewPage
        LiveProcessPanel
        RecentSessionsPanel
        IndexerStatusPanel
      SessionsPage
        SessionList
        SessionDetail
          EventList
          EventRenderer
          ToolCallBlock
      TimelinePage
        TimelineFilters
        VirtualTimeline
      MemoryPage
        MemoryTree
        MemorySearchResults
        MemoryDocumentViewer
      SearchPage
        UnifiedSearchResults
      SettingsPage
```

State strategy:

- Keep normalized maps by ID.
- Use Solid stores for entity maps.
- Use resources for REST initial loads.
- Apply WebSocket deltas to normalized state.
- Use keyed rendering and virtualized lists for events, sessions, memory results, and timeline rows.
- Avoid global rerenders from high-frequency process updates.
- Show stale/disconnected state clearly.

UI must support:

- Agent filter chips
- Live process table with CPU/RSS
- Session list grouped by agent and project
- Session detail with message/tool rendering
- Memory file browser and full-text search
- Cross-agent chronological timeline
- Unified search across sessions, events, and memory
- Indexing progress and parser errors

## Performance Budgets

Hard targets:

- UI reaction to WebSocket delta: under 100ms p95
- Process refresh cadence: 1s default, configurable
- REST list endpoints: under 100ms p95 on warm DB
- Search endpoint: under 150ms p95 for common queries
- Memory document open: under 200ms for normal markdown files
- Idle backend CPU: under 2%
- Idle backend memory: under 200MB
- Initial index of 2.2GB local data: bounded, resumable, visible progress
- Warm startup after index exists: under 2s to usable UI

Optimization requirements:

- Stream parse large files.
- Incrementally parse append-only JSONL by byte offset.
- Batch SQLite writes.
- Use FTS indexes and covering indexes.
- Use virtualized frontend lists.
- Send WebSocket diffs, not full snapshots.
- Deduplicate aggressively by stable hashes.
- Keep raw payloads optional and compressed only if needed.

## Testing

Required test layers:

- Parser golden tests per agent with fixture files.
- SQLite migration and query tests.
- File watcher debounce tests with temp directories.
- WebSocket protocol tests with reconnect and replay.
- Process detector tests with fake process snapshots.
- Frontend component tests for stores and rendering.
- Playwright smoke test for the running dashboard.
- Benchmarks for parser throughput and search latency.

Acceptance criteria:

- Running app discovers all configured roots.
- Live process panel updates without page refresh.
- New or changed session files appear in the UI through WebSocket deltas.
- Search finds content across sessions and memory.
- Timeline mixes all agents in chronological order.
- Parser errors are visible but do not crash indexing.
- App remains responsive during initial indexing.

## Security and Privacy

- Bind to `127.0.0.1` by default.
- Never upload data.
- Never mutate agent-owned files.
- Redact obvious secrets from previews and logs.
- Do not expose environment variables unless explicitly enabled.
- Require an explicit config change before binding to non-localhost.
- Keep raw transcript access local and auditable.

## Orchestration Guidance for an AI Coding Agent

Break the build into phases.

Phase 0: discovery and fixtures
- Inspect real local directories.
- Document exact observed formats.
- Create small redacted fixtures.
- Write the source adapter contracts and parser expectations.

Phase 1: skeleton
- Create Go service, SQLite migrations, REST health route, static Solid build serving, and dev scripts.
- Add a minimal Solid dashboard shell.

Phase 2: indexing foundation
- Implement DB layer, source registry, ingestion state, FTS, and one parser end to end.
- Start with Claude Code because it has known JSONL session structure.

Phase 3: live runtime
- Add process monitor, recursive watcher, WebSocket broker, and frontend live stores.

Phase 4: multi-agent adapters
- Add Codex, Augment/Auggie, and Cursor adapters.
- Each adapter must ship with golden fixtures and parser tests.

Phase 5: product UI
- Build sessions, memory browser, unified search, and timeline.
- Add error surfaces and indexing progress.

Phase 6: hardening
- Add benchmarks, Playwright smoke tests, packaging, config file support, and launch instructions.

If using multiple agents, split by ownership:
- Backend/indexing agent owns Go DB, adapters, watchers, WebSocket.
- Frontend agent owns Solid routes, stores, rendering, performance.
- Format research agent owns local storage discovery and fixtures.
- QA agent owns tests, benchmarks, and acceptance checks.

No two agents should edit the same files at the same time. The coordinator merges in this order: contracts, backend foundation, frontend shell, adapters, UI polish, performance.

## Unique Challenges

This project is hard because agent vendors do not share a session format, many formats are undocumented, files may be written while being parsed, Cursor and Electron storage can be opaque, and local transcript data is sensitive.

Design for format drift. Keep raw metadata, version parsers, surface unsupported schemas, and make reindexing cheap. Treat the database as a derived cache, not the source of truth.

## Claude Code Custom Skill Plan

Create a Claude Code skill at:

```text
~/.claude/skills/agent-observatory/SKILL.md
```

Skill purpose: help Claude Code build, run, debug, and extend Agent Observatory.

Keep `SKILL.md` concise. Put details in references only when needed.

Suggested skill structure:

```text
agent-observatory/
  SKILL.md
  references/
    architecture.md
    source-formats.md
    api-contract.md
    parser-testing.md
  scripts/
    dev-launch.sh
    collect-redacted-fixtures.sh
    smoke-test.sh
```

`SKILL.md` should include:
- When to use the skill
- Repo setup and dev commands
- Phase order
- Source adapter contract
- Validation checklist
- Rule to verify real local formats before coding parsers
- Rule to keep raw agent directories read-only

After installation:
- Restart Claude Code or start a fresh session.
- Verify the skill appears in the skills list.
- Test with: “Use the Agent Observatory skill to inspect current source formats and plan the next adapter.”
```
