import { createSignal, onCleanup } from "solid-js";
import { api, type Agent, type Session, type SessionEvent, type MemoryDoc, type TimelineItem } from "./api";
import { createWebSocket, type WSMessage } from "./ws";

export interface Process {
  pid: number;
  ppid: number;
  name: string;
  agent_type: string;
  command: string;
  cpu_percent: number;
  rss_bytes: number;
  status: string;
}

const wsConn = createWebSocket();
export const connectionStatus = wsConn.status;

// Global agent filter — shared across all pages
const [activeAgents, setActiveAgents] = createSignal<Set<string>>(new Set());
const [allAgents, setAllAgents] = createSignal<Agent[]>([]);

export function useAgentFilter() {
  return { activeAgents, setActiveAgents, allAgents, setAllAgents };
}

export function loadAgents() {
  api.agents().then((agents) => {
    setAllAgents(agents);
    setActiveAgents(new Set(agents.map(a => a.id)));
  });
}

export function agentFilterParam(): string | undefined {
  const all = allAgents();
  const active = activeAgents();
  if (active.size === 0 || active.size === all.length) return undefined;
  if (active.size === 1) return [...active][0];
  return undefined;
}

export function isAgentActive(agentId: string): boolean {
  const active = activeAgents();
  return active.size === 0 || active.has(agentId);
}

export function createSessionStore() {
  const [sessions, setSessions] = createSignal<Session[]>([]);
  const [loading, setLoading] = createSignal(true);

  async function load(q?: string, agentId?: string) {
    setLoading(true);
    try {
      const resp = await api.sessions({ q, agent: agentId, limit: 100 });
      setSessions(resp.data);
    } finally {
      setLoading(false);
    }
  }

  const off = wsConn.on("sessions", (msg: WSMessage) => {
    if (msg.type === "session.updated" || msg.type === "session.created") {
      const updated = msg.data as Session;
      setSessions((prev) => {
        const idx = prev.findIndex((s) => s.id === updated.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = updated;
          return next;
        }
        return [updated, ...prev];
      });
    }
  });

  onCleanup(off);

  return { sessions, loading, reload: load };
}

export function createSessionEvents(getId: () => string | undefined) {
  const [events, setEvents] = createSignal<SessionEvent[]>([]);

  const off = wsConn.on("sessions", (msg: WSMessage) => {
    if (msg.type === "event.appended") {
      const data = msg.data as { session_id: string };
      const id = getId();
      if (id && data.session_id === id) {
        api.session(id).then((detail) => {
          setEvents(detail.events);
        });
      }
    }
  });

  onCleanup(off);

  return { events, setEvents };
}

export function createProcessStore() {
  const [procs, setProcs] = createSignal<Process[]>([]);

  api.processes().then(setProcs);

  const off = wsConn.on("processes", (msg: WSMessage) => {
    if (msg.type === "process.snapshot" || msg.type === "process.diff") {
      setProcs(msg.data as Process[]);
    }
  });

  onCleanup(off);

  return { procs };
}

export function createMemoryStore() {
  const [docs, setDocs] = createSignal<MemoryDoc[]>([]);
  const [loading, setLoading] = createSignal(true);

  async function load(q?: string, agentId?: string) {
    setLoading(true);
    try {
      const resp = await api.memory({ q, agent: agentId, limit: 100 });
      setDocs(resp.data);
    } finally {
      setLoading(false);
    }
  }

  const off = wsConn.on("memory", (msg: WSMessage) => {
    if (msg.type === "memory.updated" || msg.type === "memory.created") {
      const updated = msg.data as MemoryDoc;
      setDocs((prev) => {
        const idx = prev.findIndex((d) => d.id === updated.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = updated;
          return next;
        }
        return [updated, ...prev];
      });
    }
  });

  onCleanup(off);

  return { docs, loading, reload: load };
}

export function createTimelineStore() {
  const [items, setItems] = createSignal<TimelineItem[]>([]);
  const [loading, setLoading] = createSignal(true);

  async function load(agentId?: string, kind?: string) {
    setLoading(true);
    try {
      const resp = await api.timeline({ agent: agentId, kind, limit: 100 });
      setItems(resp.data);
    } finally {
      setLoading(false);
    }
  }

  const off = wsConn.on("timeline", (msg: WSMessage) => {
    if (msg.type === "timeline.appended") {
      const item = msg.data as TimelineItem;
      setItems((prev) => [item, ...prev]);
    }
  });

  onCleanup(off);

  return { items, loading, reload: load };
}
