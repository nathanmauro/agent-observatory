import { For, Show } from "solid-js";
import { createProcessStore, type Process } from "../lib/stores";

export default function ProcessPanel() {
  const { procs } = createProcessStore();

  const formatRSS = (bytes: number) => {
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div class="process-panel">
      <h3 class="panel-title">Running Agents</h3>
      <Show when={procs().length > 0} fallback={<div class="panel-empty">No agent processes detected</div>}>
        <table class="process-table">
          <thead>
            <tr>
              <th>Agent</th>
              <th>PID</th>
              <th>CPU</th>
              <th>RSS</th>
              <th>Status</th>
            </tr>
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
  );
}
