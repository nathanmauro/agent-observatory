import { createResource, createEffect, For, Show, Switch, Match } from "solid-js";
import { api, type SessionEvent } from "../lib/api";
import { timeAgo, shortPath, truncate } from "../lib/format";
import { createSessionEvents } from "../lib/stores";

interface Props {
  sessionId: string;
}

export default function SessionDetail(props: Props) {
  const [data] = createResource(() => props.sessionId, (id) => api.session(id));
  const { events, setEvents } = createSessionEvents(() => props.sessionId);

  createEffect(() => {
    const d = data();
    if (d?.events) setEvents(d.events);
  });

  return (
    <div class="session-detail">
      <Show when={!data.loading} fallback={<div class="loading">Loading…</div>}>
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
                  <div class="more-events">More events available…</div>
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
