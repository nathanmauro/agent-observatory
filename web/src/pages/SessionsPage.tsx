import { createSignal, createEffect, For, Show, Switch, Match } from "solid-js";
import { useParams, useNavigate } from "@solidjs/router";
import { api, type Session, type SessionEvent } from "../lib/api";
import { createSessionStore, createSessionEvents, useAgentFilter, isAgentActive } from "../lib/stores";
import { timeAgo, shortPath, truncate } from "../lib/format";
import SearchBar from "../components/SearchBar";

export default function SessionsPage() {
  const params = useParams();
  const navigate = useNavigate();
  const { sessions, loading, reload } = createSessionStore();
  const [query, setQuery] = createSignal("");
  const { activeAgents } = useAgentFilter();

  createEffect(() => {
    activeAgents();
    reload(query() || undefined);
  });

  const filteredSessions = () => sessions().filter(s => isAgentActive(s.agent_id));

  const handleSearch = (q: string) => {
    setQuery(q);
    reload(q || undefined);
  };

  const selectSession = (id: string) => {
    navigate(`/sessions/${id}`);
  };

  return (
    <div class="sessions-page">
      <aside class="sessions-sidebar">
        <div class="session-list-header">
          <SearchBar onSearch={handleSearch} placeholder="Search sessions..." />
          <button class="btn-icon" onClick={() => reload()} title="Refresh">&#8635;</button>
        </div>
        <Show when={!loading()} fallback={<div class="loading">Loading...</div>}>
          <div class="session-items">
            <For each={filteredSessions()} fallback={<div class="empty">No sessions found</div>}>
              {(session: Session) => (
                <div
                  class={`session-item ${params.id === session.id ? "selected" : ""}`}
                  onClick={() => selectSession(session.id)}
                >
                  <div class="session-title-row">
                    <span class={`agent-dot agent-${agentType(session)}`} />
                    <span class="session-title">{session.title || "Untitled session"}</span>
                  </div>
                  <div class="session-meta">
                    <span class="session-msgs">{session.message_count} msgs</span>
                    <span class="session-project">{shortPath(session.project_path ?? "")}</span>
                    <span class="session-time">{timeAgo(session.updated_at)}</span>
                  </div>
                </div>
              )}
            </For>
          </div>
        </Show>
      </aside>
      <main class="session-main">
        <Show when={params.id} fallback={
          <div class="empty-state"><p>Select a session to view its events</p></div>
        }>
          <SessionDetailView sessionId={params.id!} />
        </Show>
      </main>
    </div>
  );
}

function SessionDetailView(props: { sessionId: string }) {
  const [data, setData] = createSignal<{ session: Session; events: SessionEvent[]; next_cursor?: string } | null>(null);
  const [loading, setLoading] = createSignal(true);
  const { events, setEvents } = createSessionEvents(() => props.sessionId);

  createEffect(() => {
    setLoading(true);
    api.session(props.sessionId).then((d) => {
      setData(d);
      setEvents(d.events);
      setLoading(false);
    });
  });

  return (
    <div class="session-detail">
      <Show when={!loading()} fallback={<div class="loading">Loading...</div>}>
        <Show when={data()?.session} fallback={<div class="empty">Session not found</div>}>
          {(session) => (
            <>
              <div class="detail-header">
                <h2>{session().title || "Untitled session"}</h2>
                <div class="detail-meta">
                  <span>{shortPath(session().project_path ?? "")}</span>
                  <span>{session().message_count} messages</span>
                  <span>{timeAgo(session().updated_at)}</span>
                  <span class={`status-badge status-${session().status}`}>{session().status}</span>
                </div>
              </div>
              <div class="event-list">
                <For each={events()}>
                  {(event: SessionEvent) => <EventRow event={event} />}
                </For>
                <Show when={data()?.next_cursor}>
                  <div class="more-events">More events available...</div>
                </Show>
              </div>
            </>
          )}
        </Show>
      </Show>
    </div>
  );
}

function EventRow(props: { event: SessionEvent }) {
  const e = () => props.event;
  return (
    <div class={`event-row event-${e().kind} role-${e().role ?? "system"}`}>
      <div class="event-header">
        <span class="event-kind">{e().kind}</span>
        <span class="event-role">{e().role}</span>
        <Show when={e().tool_name}>
          <span class="event-tool">{e().tool_name}</span>
        </Show>
        <span class="event-seq">#{e().sequence}</span>
      </div>
      <Switch>
        <Match when={e().kind === "tool_use"}>
          <div class="event-content tool-use">
            <div class="tool-name">{e().tool_name}</div>
            <Show when={e().tool_input}>
              <pre class="tool-input">{truncate(e().tool_input!, 2000)}</pre>
            </Show>
          </div>
        </Match>
        <Match when={e().kind === "tool_result"}>
          <div class="event-content tool-result">
            <Show when={e().tool_output}>
              <pre class="tool-output">{truncate(e().tool_output!, 2000)}</pre>
            </Show>
          </div>
        </Match>
        <Match when={e().kind === "thinking"}>
          <details class="event-content thinking">
            <summary>Thinking ({(e().content?.length ?? 0).toLocaleString()} chars)</summary>
            <pre>{truncate(e().content!, 5000)}</pre>
          </details>
        </Match>
        <Match when={e().kind === "message"}>
          <div class="event-content message">
            <pre>{e().content}</pre>
          </div>
        </Match>
        <Match when={e().kind === "system"}>
          <details class="event-content system-event">
            <summary>System</summary>
            <pre>{truncate(e().content!, 2000)}</pre>
          </details>
        </Match>
      </Switch>
    </div>
  );
}

function agentType(session: Session): string {
  const id = session.agent_id;
  if (id.includes("claude")) return "claude";
  if (id.includes("codex")) return "codex";
  if (id.includes("augment")) return "auggie";
  if (id.includes("cursor")) return "cursor";
  return "unknown";
}
