import {withSSEAuth} from './apiService';

export type RealtimeMessageType = 'connected' | 'stats' | 'log' | 'command_ack' | 'error' | 'pong';

export interface RealtimeMessage {
  type: RealtimeMessageType | string;
  channel?: string;
  timestamp?: number;
  payload?: unknown;
  error?: string;
}

type RealtimeCallbacks = {
  id: string;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (event: Event) => void;
  onMessage?: (message: RealtimeMessage) => void;
};

export class ServerRealtimeWsClient {
  private socket: WebSocket | null = null;
  private readonly url: string;
  private listeners = new Map<string, RealtimeCallbacks>();

  constructor(url: string) {
    this.url = withSSEAuth(url);
  }

  connect() {
    if (this.socket && (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)) {
      return;
    }

    this.socket = new WebSocket(this.url);

    this.socket.onopen = () => {
      this.listeners.forEach((listener) => listener.onOpen?.());
    };

    this.socket.onclose = () => {
      this.listeners.forEach((listener) => listener.onClose?.());
    };

    this.socket.onerror = (event) => {
      this.listeners.forEach((listener) => listener.onError?.(event));
    };

    this.socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as RealtimeMessage;
        this.listeners.forEach((listener) => listener.onMessage?.(message));
      } catch (error) {
        console.error('Failed to parse realtime WS message:', error);
      }
    };
  }

  disconnect() {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }

  subscribe(channel: 'stats' | 'logs') {
    this.send({ type: 'subscribe', channel });
  }

  unsubscribe(channel: 'stats' | 'logs') {
    this.send({ type: 'unsubscribe', channel });
  }

  sendCommand(command: string) {
    this.send({ type: 'command', command });
  }

  isConnected(): boolean {
    return this.socket?.readyState === WebSocket.OPEN;
  }

  addListener(callbacks: RealtimeCallbacks) {
    this.listeners.set(callbacks.id, callbacks);
  }

  removeListener(id: string) {
    this.listeners.delete(id);
  }

  private send(payload: object) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return;
    }
    this.socket.send(JSON.stringify(payload));
  }
}

export function useServerDetailWsTransport(): boolean {
  return import.meta.env.VITE_USE_SERVER_DETAIL_WS !== 'false';
}

type ManagedClient = {
  client: ServerRealtimeWsClient;
  refs: number;
  closeTimer: ReturnType<typeof setTimeout> | null;
};

const managedClients = new Map<string, ManagedClient>();

export function acquireServerRealtimeWsClient(serverId: string, url: string): ServerRealtimeWsClient {
  const existing = managedClients.get(serverId);
  if (existing) {
    existing.refs += 1;
    if (existing.closeTimer) {
      clearTimeout(existing.closeTimer);
      existing.closeTimer = null;
    }
    return existing.client;
  }

  const managed: ManagedClient = {
    client: new ServerRealtimeWsClient(url),
    refs: 1,
    closeTimer: null
  };
  managedClients.set(serverId, managed);
  return managed.client;
}

export function releaseServerRealtimeWsClient(serverId: string) {
  const existing = managedClients.get(serverId);
  if (!existing) {
    return;
  }

  existing.refs -= 1;
  if (existing.refs > 0) {
    return;
  }

  // Grace period prevents noisy close/reopen cycles during fast remounts.
  existing.closeTimer = setTimeout(() => {
    const current = managedClients.get(serverId);
    if (!current || current.refs > 0) {
      return;
    }
    current.client.disconnect();
    managedClients.delete(serverId);
  }, 1000);
}
