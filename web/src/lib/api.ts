const BASE = import.meta.env.DEV ? "http://127.0.0.1:3284" : "";

export interface Agent {
  id: string;
  type: string;
  name: string;
  root_path: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Session {
  id: string;
  agent_id: string;
  external_id?: string;
  source_path?: string;
  project_path?: string;
  title?: string;
  status?: string;
  started_at: string;
  updated_at: string;
  message_count: number;
  summary?: string;
}

export interface SessionEvent {
  id: string;
  session_id: string;
  agent_id: string;
  sequence: number;
  timestamp: string;
  role?: string;
  kind: string;
  content?: string;
  tool_name?: string;
  tool_input?: string;
  tool_output?: string;
}

export interface SearchResult {
  type: string;
  id: string;
  agent_type?: string;
  title: string;
  path?: string;
  project?: string;
  snippet?: string;
  rank?: number;
  timestamp?: string;
}

export interface PagedResponse<T> {
  data: T[];
  next_cursor?: string;
  total?: number;
}

export interface SessionDetail {
  session: Session;
  events: SessionEvent[];
  next_cursor?: string;
}

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) throw new Error(`${res.status}: ${await res.text()}`);
  return res.json();
}

export const api = {
  health: () => fetchJSON<{ status: string; version: string }>("/health"),
  agents: () => fetchJSON<Agent[]>("/api/v1/agents"),
  sessions: (params?: { agent?: string; project?: string; q?: string; limit?: number; cursor?: string }) => {
    const sp = new URLSearchParams();
    if (params?.agent) sp.set("agent", params.agent);
    if (params?.project) sp.set("project", params.project);
    if (params?.q) sp.set("q", params.q);
    if (params?.limit) sp.set("limit", String(params.limit));
    if (params?.cursor) sp.set("cursor", params.cursor);
    const qs = sp.toString();
    return fetchJSON<PagedResponse<Session>>(`/api/v1/sessions${qs ? `?${qs}` : ""}`);
  },
  session: (id: string) => fetchJSON<SessionDetail>(`/api/v1/sessions/${id}`),
  search: (q: string, limit = 50) => fetchJSON<SearchResult[]>(`/api/v1/search?q=${encodeURIComponent(q)}&limit=${limit}`),
  reindex: () => fetch(`${BASE}/api/v1/reindex`, { method: "POST" }).then(r => r.json()),
};
