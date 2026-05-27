# Handoff — Agent Observatory

## For the next session

You are picking up a greenfield Go + SolidJS project. Everything is planned, nothing is built yet.

### Start here

1. `cd ~/Developer/proj/agent-observatory`
2. `git checkout -b feature/phase-1-mvp`
3. Read `PLAN.md` for the checklist, `SPEC.md` for the full architecture

### What exists

- `SPEC.md` — full architecture spec (reviewed and corrected by 4-agent orchestrated review)
- `PLAN.md` — current phase, checklist, session log
- `prompts.md` — prompt history (original → Codex-enhanced)
- Claude Code skill at `~/.claude/skills/agent-observatory/` with references for architecture and source formats

### What to build (Phase 1 MVP)

A Go server that parses Claude Code JSONL sessions into SQLite FTS5, serves a SolidJS frontend with session list + search. REST only, no WebSocket yet. Claude-only, no other agents yet.

Target: ~20 files, ~830 lines.

**Backend first order:**
1. `go mod init github.com/nathanmauro/agent-observatory`
2. SQLite migrations (schema in SPEC.md § SQLite Model)
3. Claude Code parser (`~/.claude/projects/**/*.jsonl` — line-by-line, byte-offset incremental)
4. REST handlers (chi router): health, agents, sessions, sessions/{id}, reindex
5. Static embed of SolidJS build

**Frontend second order:**
1. `cd web && npm create vite@latest . -- --template solid-ts`
2. SessionList, SessionDetail, SearchBar, API client
3. Build → `web/dist/` → Go embeds it

### Key decisions already made

- **Go + SolidJS**: intentional learning project, not reuse of agentseq Python/React
- **modernc.org/sqlite** over mattn/go-sqlite3: pure Go, no CGo, true single binary
- **chi router**: lightweight, stdlib-compatible
- **No WebSocket in Phase 1**: polling or manual reindex button
- **Claude-only in Phase 1**: best-understood format, simplest adapter
- **Localhost only**: bind 127.0.0.1, never upload data, never mutate agent dirs

### Verified local formats (from review agents)

Claude Code JSONL records have mixed types:
- Message records: `{"parentUuid":"...","isSidechain":false,"message":{"model":"claude-opus-4-7","role":"assistant","content":[...]}}`
- Metadata records: `{"type":"last-prompt",...}`, `{"type":"ai-title",...}`
- Fields to extract: `message.role`, `message.content`, `message.usage`, `isSidechain`, `parentUuid`
- Parser must handle both message containers and metadata events

Existing agentseq parser is 118 lines of Python — the Go port should be ~50-80 lines.

### What NOT to do

- Don't touch `~/Developer/proj/claude-sessions/` (agentseq) — this is independent
- Don't add WebSocket, process monitoring, or multi-agent support yet
- Don't use CGo (kills the single-binary advantage)
- Don't write agent files — read-only access to `~/.claude/`, `~/.codex/`, etc.

### After Phase 1

Update `PLAN.md` session log, merge branch, then `git checkout -b feature/phase-2-live` for WebSocket + process monitoring.
