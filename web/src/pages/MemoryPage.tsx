import { createSignal, createEffect, createResource, For, Show } from "solid-js";
import { useParams, useNavigate } from "@solidjs/router";
import { api, type MemoryDoc } from "../lib/api";
import { createMemoryStore, useAgentFilter, isAgentActive } from "../lib/stores";
import { timeAgo, shortPath } from "../lib/format";
import SearchBar from "../components/SearchBar";

export default function MemoryPage() {
  const params = useParams();
  const navigate = useNavigate();
  const { docs, loading, reload } = createMemoryStore();
  const [query, setQuery] = createSignal("");
  const { activeAgents } = useAgentFilter();

  createEffect(() => {
    activeAgents();
    reload(query() || undefined);
  });

  const filtered = () => docs().filter(d => isAgentActive(d.agent_id));

  const handleSearch = (q: string) => {
    setQuery(q);
    reload(q || undefined);
  };

  return (
    <div class="memory-page">
      <aside class="memory-sidebar">
        <div class="memory-list-header">
          <SearchBar onSearch={handleSearch} placeholder="Search memory files..." />
        </div>
        <Show when={!loading()} fallback={<div class="loading">Loading...</div>}>
          <div class="memory-items">
            <For each={filtered()} fallback={<div class="empty">No memory files found</div>}>
              {(doc: MemoryDoc) => (
                <div
                  class={`memory-item ${params.id === doc.id ? "selected" : ""}`}
                  onClick={() => navigate(`/memory/${doc.id}`)}
                >
                  <div class="memory-title-row">
                    <span class={`agent-dot agent-${agentType(doc)}`} />
                    <span class="memory-title">{doc.title}</span>
                  </div>
                  <div class="memory-meta">
                    <span>{shortPath(doc.source_path)}</span>
                    <span>{formatSize(doc.size_bytes)}</span>
                    <span>{timeAgo(doc.mtime)}</span>
                  </div>
                </div>
              )}
            </For>
          </div>
        </Show>
      </aside>
      <main class="memory-main">
        <Show when={params.id} fallback={
          <div class="empty-state"><p>Select a memory file to view its contents</p></div>
        }>
          <MemoryDocView docId={params.id!} />
        </Show>
      </main>
    </div>
  );
}

function MemoryDocView(props: { docId: string }) {
  const [doc] = createResource(() => props.docId, (id) => api.memoryDoc(id));

  return (
    <div class="memory-detail">
      <Show when={!doc.loading} fallback={<div class="loading">Loading...</div>}>
        <Show when={doc()} fallback={<div class="empty">Document not found</div>}>
          {(d) => (
            <>
              <div class="detail-header">
                <h2>{d().title}</h2>
                <div class="detail-meta">
                  <span>{shortPath(d().source_path)}</span>
                  <span>{formatSize(d().size_bytes)}</span>
                  <span>{timeAgo(d().mtime)}</span>
                </div>
              </div>
              <pre class="memory-content">{d().content}</pre>
            </>
          )}
        </Show>
      </Show>
    </div>
  );
}

function agentType(doc: MemoryDoc): string {
  if (doc.agent_id.includes("claude")) return "claude";
  if (doc.agent_id.includes("codex")) return "codex";
  if (doc.agent_id.includes("augment")) return "auggie";
  if (doc.agent_id.includes("cursor")) return "cursor";
  return "unknown";
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
