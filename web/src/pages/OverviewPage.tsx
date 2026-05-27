import { createResource, For, Show } from "solid-js";
import { A } from "@solidjs/router";
import { api, type Session, type TimelineItem } from "../lib/api";
import { createProcessStore, type Process } from "../lib/stores";
import { timeAgo, shortPath } from "../lib/format";

export default function OverviewPage() {
  const [stats] = createResource(() => api.stats());
  const [recent] = createResource(() => api.sessions({ limit: 8 }).then(r => r.data));
  const [timeline] = createResource(() => api.timeline({ limit: 10 }).then(r => r.data));
  const { procs } = createProcessStore();

  return (
    <div class="overview-page">
      <div class="overview-grid">
        <div class="stat-cards">
          <Show when={stats()}>
            {(s) => (
              <>
                <StatCard label="Sessions" value={s().total_sessions} href="/sessions" />
                <StatCard label="Events" value={s().total_events} />
                <StatCard label="Memory Files" value={s().total_memory_docs} href="/memory" />
                <StatCard label="Agents" value={Object.keys(s().agent_counts).length} />
              </>
            )}
          </Show>
        </div>

        <div class="overview-section">
          <div class="section-header">
            <h3>Running Agents</h3>
          </div>
          <Show when={procs().length > 0} fallback={<div class="panel-empty">No agent processes detected</div>}>
            <table class="process-table">
              <thead>
                <tr><th>Agent</th><th>PID</th><th>CPU</th><th>RSS</th><th>Status</th></tr>
              </thead>
              <tbody>
                <For each={procs()}>
                  {(p: Process) => (
                    <tr>
                      <td class="proc-name">{p.name}</td>
                      <td class="proc-pid">{p.pid}</td>
                      <td class="proc-cpu">{p.cpu_percent.toFixed(1)}%</td>
                      <td class="proc-rss">{formatRSS(p.rss_bytes)}</td>
                      <td class="proc-status">{p.status}</td>
                    </tr>
                  )}
                </For>
              </tbody>
            </table>
          </Show>
        </div>

        <div class="overview-section">
          <div class="section-header">
            <h3>Recent Sessions</h3>
            <A href="/sessions" class="section-link">View all</A>
          </div>
          <Show when={recent()} fallback={<div class="loading-sm">Loading...</div>}>
            <div class="mini-session-list">
              <For each={recent()}>
                {(s: Session) => (
                  <A href={`/sessions/${s.id}`} class="mini-session-item">
                    <span class="mini-session-title">{s.title || "Untitled"}</span>
                    <span class="mini-session-meta">
                      {s.message_count} msgs · {shortPath(s.project_path ?? "")} · {timeAgo(s.updated_at)}
                    </span>
                  </A>
                )}
              </For>
            </div>
          </Show>
        </div>

        <div class="overview-section">
          <div class="section-header">
            <h3>Timeline</h3>
            <A href="/timeline" class="section-link">View all</A>
          </div>
          <Show when={timeline()} fallback={<div class="loading-sm">Loading...</div>}>
            <div class="mini-timeline">
              <For each={timeline()}>
                {(t: TimelineItem) => (
                  <div class="mini-timeline-item">
                    <span class={`tl-kind tl-${t.kind.split(".")[0]}`}>{t.kind}</span>
                    <span class="tl-title">{t.title || "Untitled"}</span>
                    <span class="tl-time">{timeAgo(t.timestamp)}</span>
                  </div>
                )}
              </For>
            </div>
          </Show>
        </div>
      </div>
    </div>
  );
}

function StatCard(props: { label: string; value: number; href?: string }) {
  const inner = (
    <div class="stat-card">
      <div class="stat-value">{(props.value ?? 0).toLocaleString()}</div>
      <div class="stat-label">{props.label}</div>
    </div>
  );
  return props.href ? <A href={props.href} class="stat-card-link">{inner}</A> : inner;
}

function formatRSS(bytes: number) {
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
