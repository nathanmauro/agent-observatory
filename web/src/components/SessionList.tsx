import { createResource, createSignal, For, Show } from "solid-js";
import { api, type Session } from "../lib/api";
import { timeAgo, shortPath } from "../lib/format";
import SearchBar from "./SearchBar";

interface Props {
  onSelect: (id: string) => void;
  selectedId?: string;
}

export default function SessionList(props: Props) {
  const [query, setQuery] = createSignal("");
  const [cursor, setCursor] = createSignal<string | undefined>();

  const [data, { refetch }] = createResource(
    () => ({ q: query(), cursor: cursor() }),
    (params) => api.sessions({ q: params.q || undefined, limit: 50, cursor: params.cursor }),
  );

  const handleSearch = (q: string) => {
    setCursor(undefined);
    setQuery(q);
  };

  const loadMore = () => {
    const next = data()?.next_cursor;
    if (next) setCursor(next);
  };

  return (
    <div class="session-list">
      <div class="session-list-header">
        <SearchBar onSearch={handleSearch} />
        <button class="btn-icon" onClick={() => { setCursor(undefined); refetch(); }} title="Refresh">
          ↻
        </button>
      </div>
      <Show when={!data.loading} fallback={<div class="loading">Loading…</div>}>
        <div class="session-items">
          <For each={data()?.data ?? []} fallback={<div class="empty">No sessions found</div>}>
            {(session: Session) => (
              <div
                class={`session-item ${props.selectedId === session.id ? "selected" : ""}`}
                onClick={() => props.onSelect(session.id)}
              >
                <div class="session-title">
                  {session.title || "Untitled session"}
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
        <Show when={data()?.next_cursor}>
          <button class="load-more" onClick={loadMore}>Load more</button>
        </Show>
      </Show>
    </div>
  );
}
