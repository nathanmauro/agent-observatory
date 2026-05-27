import { createSignal, For, Show } from "solid-js";
import { A, useSearchParams } from "@solidjs/router";
import { api, type SearchResult } from "../lib/api";
import { timeAgo, shortPath } from "../lib/format";

const TYPE_LABELS: Record<string, string> = {
  session: "Session",
  event: "Event",
  memory: "Memory",
};

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [results, setResults] = createSignal<SearchResult[]>([]);
  const [loading, setLoading] = createSignal(false);
  const initParam = typeof searchParams.q === "string" ? searchParams.q : "";
  const [query, setQuery] = createSignal(initParam);
  const [searched, setSearched] = createSignal(false);

  const doSearch = async (q: string) => {
    if (!q.trim()) return;
    setQuery(q);
    setSearchParams({ q });
    setLoading(true);
    setSearched(true);
    try {
      const res = await api.search(q, { limit: 100 });
      setResults(res);
    } finally {
      setLoading(false);
    }
  };

  const initQ = searchParams.q;
  if (typeof initQ === "string" && initQ) {
    doSearch(initQ);
  }

  const resultsByType = () => {
    const grouped: Record<string, SearchResult[]> = {};
    for (const r of results()) {
      if (!grouped[r.type]) grouped[r.type] = [];
      grouped[r.type].push(r);
    }
    return grouped;
  };

  return (
    <div class="search-page">
      <div class="page-header">
        <h2>Search</h2>
      </div>
      <form class="search-form" onSubmit={(e) => { e.preventDefault(); doSearch(query()); }}>
        <input
          type="text"
          class="search-input-lg"
          value={query()}
          onInput={(e) => setQuery(e.currentTarget.value)}
          placeholder="Search across sessions, events, and memory..."
          autofocus
        />
        <button type="submit" class="btn-primary">Search</button>
      </form>

      <Show when={loading()}>
        <div class="loading">Searching...</div>
      </Show>

      <Show when={!loading() && searched()}>
        <Show when={results().length > 0} fallback={<div class="empty">No results found for "{query()}"</div>}>
          <div class="search-results">
            <div class="result-summary">{results().length} results for "{query()}"</div>
            <For each={Object.entries(resultsByType())}>
              {([type, items]) => (
                <div class="result-group">
                  <h3 class="result-group-label">{TYPE_LABELS[type] || type}s ({items.length})</h3>
                  <For each={items}>
                    {(r) => <ResultRow result={r} />}
                  </For>
                </div>
              )}
            </For>
          </div>
        </Show>
      </Show>
    </div>
  );
}

function ResultRow(props: { result: SearchResult }) {
  const r = () => props.result;

  const href = () => {
    switch (r().type) {
      case "session": return `/sessions/${r().id}`;
      case "memory": return `/memory/${r().id}`;
      case "event": return `/sessions/${r().id}`;
      default: return undefined;
    }
  };

  return (
    <A href={href() || "#"} class="result-row">
      <div class="result-header">
        <span class={`result-type result-type-${r().type}`}>{r().type}</span>
        <Show when={r().agent_type}>
          <span class={`agent-badge agent-${r().agent_type}`}>{r().agent_type}</span>
        </Show>
        <span class="result-title">{r().title || "Untitled"}</span>
        <Show when={r().timestamp}>
          <span class="result-time">{timeAgo(r().timestamp!)}</span>
        </Show>
      </div>
      <Show when={r().snippet}>
        <div class="result-snippet">{r().snippet}</div>
      </Show>
      <Show when={r().project || r().path}>
        <div class="result-path">{shortPath(r().project || r().path || "")}</div>
      </Show>
    </A>
  );
}
