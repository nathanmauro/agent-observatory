import { createSignal, Show } from "solid-js";
import SessionList from "./components/SessionList";
import SessionDetail from "./components/SessionDetail";
import { api } from "./lib/api";

export default function App() {
  const [selectedId, setSelectedId] = createSignal<string | undefined>();
  const [reindexing, setReindexing] = createSignal(false);

  const handleReindex = async () => {
    setReindexing(true);
    try {
      await api.reindex();
    } finally {
      setTimeout(() => setReindexing(false), 2000);
    }
  };

  return (
    <div class="app">
      <header class="app-header">
        <h1>Agent Observatory</h1>
        <div class="header-actions">
          <button onClick={handleReindex} disabled={reindexing()}>
            {reindexing() ? "Reindexing…" : "Reindex"}
          </button>
        </div>
      </header>
      <div class="app-body">
        <aside class="sidebar">
          <SessionList onSelect={setSelectedId} selectedId={selectedId()} />
        </aside>
        <main class="main-content">
          <Show when={selectedId()} fallback={
            <div class="empty-state">
              <p>Select a session to view its events</p>
            </div>
          }>
            {(id) => <SessionDetail sessionId={id()} />}
          </Show>
        </main>
      </div>
    </div>
  );
}
