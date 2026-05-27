import { createSignal } from "solid-js";

export type ConnectionStatus = "connected" | "connecting" | "disconnected";
export type WSHandler = (msg: WSMessage) => void;

export interface WSMessage {
  seq: number;
  schema_version: number;
  type: string;
  topic: string;
  sent_at: string;
  data: any;
}

const WS_BASE = import.meta.env.DEV ? "ws://127.0.0.1:3284" : `ws://${location.host}`;

export function createWebSocket() {
  const [status, setStatus] = createSignal<ConnectionStatus>("disconnected");
  let ws: WebSocket | null = null;
  let lastSeq = 0;
  let reconnectDelay = 1000;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  const handlers = new Map<string, Set<WSHandler>>();

  function connect() {
    setStatus("connecting");
    ws = new WebSocket(`${WS_BASE}/api/v1/ws`);

    ws.onopen = () => {
      setStatus("connected");
      reconnectDelay = 1000;
      ws!.send(JSON.stringify({ type: "hello", last_seq: lastSeq }));
      ws!.send(JSON.stringify({ type: "subscribe", topics: ["sessions", "processes"] }));
    };

    ws.onmessage = (ev) => {
      let msg: WSMessage;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (msg.seq > lastSeq) lastSeq = msg.seq;
      dispatch(msg);
    };

    ws.onclose = () => {
      setStatus("disconnected");
      scheduleReconnect();
    };

    ws.onerror = () => {
      ws?.close();
    };
  }

  function scheduleReconnect() {
    if (reconnectTimer) return;
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      reconnectDelay = Math.min(reconnectDelay * 2, 30000);
      connect();
    }, reconnectDelay);
  }

  function dispatch(msg: WSMessage) {
    const topicHandlers = handlers.get(msg.topic);
    if (topicHandlers) {
      for (const h of topicHandlers) h(msg);
    }
    const allHandlers = handlers.get("*");
    if (allHandlers) {
      for (const h of allHandlers) h(msg);
    }
  }

  function on(topic: string, handler: WSHandler) {
    if (!handlers.has(topic)) handlers.set(topic, new Set());
    handlers.get(topic)!.add(handler);
    return () => handlers.get(topic)?.delete(handler);
  }

  function close() {
    if (reconnectTimer) clearTimeout(reconnectTimer);
    ws?.close();
  }

  connect();

  return { status, on, close };
}
