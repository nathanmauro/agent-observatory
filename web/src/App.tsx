import { createSignal, onMount } from "solid-js";
import { A, useLocation } from "@solidjs/router";
import AgentChips from "./components/AgentChips";
import { connectionStatus, loadAgents } from "./lib/stores";
import { api } from "./lib/api";

export default function App(props: { children?: any }) {
  const location = useLocation();
  const [reindexing, setReindexing] = createSignal(false);

  onMount(() => {
    loadAgents();
  });

  const handleReindex = async () => {
    setReindexing(true);
    try {
      await api.reindex();
    } finally {
      setTimeout(() => setReindexing(false), 2000);
    }
  };

  const statusColor = () => {
    switch (connectionStatus()) {
      case "connected": return "var(--green)";
      case "connecting": return "var(--yellow)";
      default: return "var(--red)";
    }
  };

  const isActive = (path: string) => {
    if (path === "/") return location.pathname === "/";
    return location.pathname.startsWith(path);
  };

  return (
    <div class="app">
      <header class="app-header">
        <div class="header-left">
          <A href="/" class="logo">Agent Observatory</A>
          <nav class="header-nav">
            <A href="/" class={`nav-link ${isActive("/") ? "active" : ""}`}>Overview</A>
            <A href="/sessions" class={`nav-link ${isActive("/sessions") ? "active" : ""}`}>Sessions</A>
            <A href="/timeline" class={`nav-link ${isActive("/timeline") ? "active" : ""}`}>Timeline</A>
            <A href="/memory" class={`nav-link ${isActive("/memory") ? "active" : ""}`}>Memory</A>
            <A href="/search" class={`nav-link ${isActive("/search") ? "active" : ""}`}>Search</A>
          </nav>
        </div>
        <div class="header-right">
          <AgentChips />
          <span class="conn-status" title={connectionStatus()}>
            <span class="conn-dot" style={{ background: statusColor() }} />
          </span>
          <button class="btn-sm" onClick={handleReindex} disabled={reindexing()}>
            {reindexing() ? "Reindexing..." : "Reindex"}
          </button>
        </div>
      </header>
      <main class="app-body">
        {props.children}
      </main>
    </div>
  );
}
