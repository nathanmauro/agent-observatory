import { createSignal, onCleanup } from "solid-js";
import { api, type Session, type SessionEvent } from "./api";
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

export function createSessionStore() {
  const [sessions, setSessions] = createSignal<Session[]>([]);
  const [loading, setLoading] = createSignal(true);

  async function load(q?: string) {
    setLoading(true);
    try {
      const resp = await api.sessions({ q, limit: 50 });
      setSessions(resp.data);
    } finally {
      setLoading(false);
    }
  }

  load();

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
