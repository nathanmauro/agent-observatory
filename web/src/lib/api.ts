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

export interface MemoryDoc {
  id: string;
  agent_id: string;
  source_path: string;
  project_path?: string;
  title: string;
  content?: string;
  size_bytes: number;
  mtime: string;
}

export interface TimelineItem {
  id: string;
  timestamp: string;
  agent_id: string;
  agent_type?: string;
  session_id?: string;
  memory_doc_id?: string;
  kind: string;
  title: string;
  body?: string;
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

export interface Stats {
  total_sessions: number;
  total_events: number;
  total_memory_docs: number;
  agent_counts: Record<string, number>;
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

function buildParams(params: Record<string, string | number | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") sp.set(k, String(v));
  }
  const qs = sp.toString();
  return qs ? `?${qs}` : "";
}

export const api = {
  health: () => fetchJSON<{ status: string; version: string }>("/health"),
  agents: () => fetchJSON<Agent[]>("/api/v1/agents"),

  sessions: (params?: { agent?: string; project?: string; q?: string; limit?: number; cursor?: string }) =>
    fetchJSON<PagedResponse<Session>>(`/api/v1/sessions${buildParams(params ?? {})}`),

  session: (id: string) => fetchJSON<SessionDetail>(`/api/v1/sessions/${id}`),

  search: (q: string, params?: { type?: string; agent?: string; limit?: number }) =>
    fetchJSON<SearchResult[]>(`/api/v1/search${buildParams({ q, ...params })}`),

  memory: (params?: { agent?: string; project?: string; q?: string; limit?: number; cursor?: string }) =>
    fetchJSON<PagedResponse<MemoryDoc>>(`/api/v1/memory${buildParams(params ?? {})}`),

  memoryDoc: (id: string) => fetchJSON<MemoryDoc>(`/api/v1/memory/${id}`),

  timeline: (params?: { agent?: string; kind?: string; limit?: number; cursor?: string }) =>
    fetchJSON<PagedResponse<TimelineItem>>(`/api/v1/timeline${buildParams(params ?? {})}`),

  stats: () => fetchJSON<Stats>("/api/v1/stats"),

  processes: () => fetchJSON<any[]>("/api/v1/processes"),
  reindex: () => fetch(`${BASE}/api/v1/reindex`, { method: "POST" }).then(r => r.json()),
};
