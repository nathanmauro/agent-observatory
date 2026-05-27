import { For, Show } from "solid-js";
import { type Session } from "../lib/api";
import { timeAgo, shortPath } from "../lib/format";
import { createSessionStore } from "../lib/stores";
import SearchBar from "./SearchBar";

interface Props {
  onSelect: (id: string) => void;
  selectedId?: string;
}

export default function SessionList(props: Props) {
  const { sessions, loading, reload } = createSessionStore();

  const handleSearch = (q: string) => {
    reload(q || undefined);
  };

  return (
    <div class="session-list">
      <div class="session-list-header">
        <SearchBar onSearch={handleSearch} />
        <button class="btn-icon" onClick={() => reload()} title="Refresh">
          ↻
        </button>
      </div>
      <Show when={!loading()} fallback={<div class="loading">Loading…</div>}>
        <div class="session-items">
          <For each={sessions()} fallback={<div class="empty">No sessions found</div>}>
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
      </Show>
    </div>
  );
}
