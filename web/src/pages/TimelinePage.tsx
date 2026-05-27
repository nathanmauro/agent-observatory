import { createEffect, For, Show } from "solid-js";
import { A } from "@solidjs/router";
import { createTimelineStore, useAgentFilter, isAgentActive } from "../lib/stores";
import { timeAgo } from "../lib/format";
import type { TimelineItem } from "../lib/api";

const KIND_LABELS: Record<string, string> = {
  "session.created": "New Session",
  "session.updated": "Session Updated",
  "memory.created": "New Memory",
  "memory.updated": "Memory Updated",
};

export default function TimelinePage() {
  const { items, loading, reload } = createTimelineStore();
  const { activeAgents } = useAgentFilter();

  createEffect(() => {
    activeAgents();
    reload();
  });

  const filtered = () => items().filter(t => isAgentActive(t.agent_id));

  const grouped = () => {
    const groups: { date: string; items: TimelineItem[] }[] = [];
    let currentDate = "";
    for (const item of filtered()) {
      const d = item.timestamp.slice(0, 10);
      if (d !== currentDate) {
        currentDate = d;
        groups.push({ date: d, items: [] });
      }
      groups[groups.length - 1].items.push(item);
    }
    return groups;
  };

  return (
    <div class="timeline-page">
      <div class="page-header">
        <h2>Timeline</h2>
        <button class="btn-sm" onClick={() => reload()}>Refresh</button>
      </div>
      <Show when={!loading()} fallback={<div class="loading">Loading...</div>}>
        <Show when={filtered().length > 0} fallback={<div class="empty">No timeline events</div>}>
          <div class="timeline-groups">
            <For each={grouped()}>
              {(group) => (
                <div class="timeline-group">
                  <div class="timeline-date">{formatDate(group.date)}</div>
                  <div class="timeline-items">
                    <For each={group.items}>
                      {(item) => <TimelineRow item={item} />}
                    </For>
                  </div>
                </div>
              )}
            </For>
          </div>
        </Show>
      </Show>
    </div>
  );
}

function TimelineRow(props: { item: TimelineItem }) {
  const t = () => props.item;
  const kindBase = () => t().kind.split(".")[0];
  const label = () => KIND_LABELS[t().kind] || t().kind;

  const link = () => {
    if (t().session_id) return `/sessions/${t().session_id}`;
    if (t().memory_doc_id) return `/memory/${t().memory_doc_id}`;
    return undefined;
  };

  const inner = (
    <div class={`timeline-row tl-${kindBase()}`}>
      <div class="tl-marker" />
      <div class="tl-content">
        <div class="tl-header">
          <span class={`tl-badge tl-badge-${kindBase()}`}>{label()}</span>
          <Show when={t().agent_type}>
            <span class={`agent-badge agent-${t().agent_type}`}>{t().agent_type}</span>
          </Show>
          <span class="tl-time">{timeAgo(t().timestamp)}</span>
        </div>
        <div class="tl-title">{t().title || "Untitled"}</div>
        <Show when={t().body}>
          <div class="tl-body">{t().body}</div>
        </Show>
      </div>
    </div>
  );

  const href = link();
  return href ? <A href={href} class="tl-link">{inner}</A> : inner;
}

function formatDate(iso: string): string {
  const d = new Date(iso + "T00:00:00");
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (iso === today.toISOString().slice(0, 10)) return "Today";
  if (iso === yesterday.toISOString().slice(0, 10)) return "Yesterday";
  return d.toLocaleDateString("en-US", { weekday: "long", month: "short", day: "numeric" });
}
