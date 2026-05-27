import { For } from "solid-js";
import { useAgentFilter } from "../lib/stores";

const AGENT_COLORS: Record<string, string> = {
  claude: "#7c5cfc",
  codex: "#34d399",
  auggie: "#f59e0b",
  cursor: "#60a5fa",
};

export default function AgentChips() {
  const { allAgents, activeAgents, setActiveAgents } = useAgentFilter();

  const toggle = (id: string) => {
    setActiveAgents((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const allActive = () => activeAgents().size === allAgents().length;

  const toggleAll = () => {
    if (allActive()) {
      setActiveAgents(new Set<string>());
    } else {
      setActiveAgents(new Set<string>(allAgents().map(a => a.id)));
    }
  };

  return (
    <div class="agent-chips">
      <button
        class={`agent-chip ${allActive() ? "active" : ""}`}
        onClick={toggleAll}
        style={{ "--chip-color": "var(--text-dim)" }}
      >
        All
      </button>
      <For each={allAgents()}>
        {(agent) => (
          <button
            class={`agent-chip ${activeAgents().has(agent.id) ? "active" : ""}`}
            onClick={() => toggle(agent.id)}
            style={{ "--chip-color": AGENT_COLORS[agent.type] || "var(--text-dim)" }}
          >
            <span class="chip-dot" />
            {agent.name}
          </button>
        )}
      </For>
    </div>
  );
}
