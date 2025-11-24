import { create } from 'zustand';
import { Server } from '../types';
import * as api from '../service/apiService';
import {
  mapDockerContainersToServers,
  mapContainersWithStatsToServers,
  DockerContainer,
  ContainerWithStats
} from '../utils/serverMapper';

interface ServerState {
  servers: Server[];
  selectedServerId: string | null;
  loading: boolean;
  error: string | null;
  eventSource: EventSource | null;

  // Actions
  fetchServers: () => Promise<void>;
  connectSSE: () => void;
  disconnectSSE: () => void;
  setSelectedServer: (serverId: string | null) => void;
  restartServer: (serverId: string) => Promise<void>;
  stopServer: (serverId: string) => Promise<void>;
  startServer: (serverId: string) => Promise<void>;
  addServer: (serverData: Omit<Server, 'id' | 'status' | 'uptime' | 'cpu' | 'memory' | 'players'>) => Promise<void>;
  updateServer: (serverId: string, data: { name?: string; env?: Record<string, string> }) => Promise<void>;
  deleteServer: (serverId: string, removeData?: boolean) => Promise<void>;
  addPluginToServer: (serverId: string, pluginId: string) => Promise<void>;
  removePluginFromServer: (serverId: string, pluginId: string) => Promise<void>;

  // Selectors
  getServerById: (id: string) => Server | undefined;
  getSelectedServer: () => Server | undefined;
}

const API_BASE_URL = 'http://localhost:3000/api/v1';

// Sort servers: Proxy first, Lobby second, then game servers by port
const sortServers = (servers: Server[]): Server[] => {
  return [...servers].sort((a, b) => {
    // Proxy always first
    if (a.location === 'Proxy') return -1;
    if (b.location === 'Proxy') return 1;

    // Lobby always second
    if (a.location === 'Lobby') return -1;
    if (b.location === 'Lobby') return 1;

    // Game servers sorted by port number
    return a.port - b.port;
  });
};

export const useServerStore = create<ServerState>((set, get) => ({
  servers: [],
  selectedServerId: null,
  loading: false,
  error: null,
  eventSource: null,

  fetchServers: async () => {
    set({ loading: true, error: null });

    try {
      const response = await api.getServers();
      const dockerContainers: DockerContainer[] = response.data;
      const servers = sortServers(mapDockerContainersToServers(dockerContainers));

      set({ servers, loading: false });
    } catch (error) {
      console.error('Failed to fetch servers:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch servers',
        loading: false
      });
    }
  },

  connectSSE: () => {
    // Clean up existing connection if any
    const currentEventSource = get().eventSource;
    if (currentEventSource) {
      currentEventSource.close();
    }

    try {
      const eventSource = new EventSource(`${API_BASE_URL}/server/stream`);

      eventSource.onopen = () => {
        console.log('SSE connection established for server list');
        set({ error: null });
      };

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);

          // Check if data is wrapped in an SSE event structure
          let containersData: ContainerWithStats[] | DockerContainer[];
          if (data.Data) {
            // Data is wrapped in SSE event structure
            containersData = typeof data.Data === 'string'
              ? JSON.parse(data.Data)
              : data.Data;
          } else {
            // Data is the container array directly
            containersData = data;
          }

          // Check if data has the new format with stats
          let servers: Server[];
          if (containersData.length > 0 && 'container' in containersData[0]) {
            // New format: array of {container, stats}
            servers = sortServers(mapContainersWithStatsToServers(containersData as ContainerWithStats[]));
          } else {
            // Old format: array of containers
            servers = sortServers(mapDockerContainersToServers(containersData as DockerContainer[]));
          }

          set({ servers, loading: false });
        } catch (err) {
          console.error('Error processing SSE message:', err);
        }
      };

      eventSource.onerror = (error) => {
        console.error('SSE connection error:', error);
        eventSource.close();

        // Try to reconnect after 5 seconds
        setTimeout(() => {
          const currentState = get();
          if (!currentState.eventSource || currentState.eventSource.readyState === EventSource.CLOSED) {
            console.log('Attempting to reconnect SSE...');
            get().connectSSE();
          }
        }, 5000);
      };

      set({ eventSource });
    } catch (error) {
      console.error('Failed to establish SSE connection:', error);
      set({ error: 'Failed to establish real-time connection' });
    }
  },

  disconnectSSE: () => {
    const eventSource = get().eventSource;
    if (eventSource) {
      eventSource.close();
      set({ eventSource: null });
      console.log('SSE connection closed');
    }
  },

  setSelectedServer: (serverId: string | null) => {
    set({ selectedServerId: serverId });
  },

  restartServer: async (serverId: string) => {
    set({ loading: true, error: null });

    try {
      await api.restartServer(serverId);

      // Optimistically update status
      set(state => ({
        servers: state.servers.map(server =>
          server.id === serverId
            ? { ...server, status: 'restarting' }
            : server
        ),
        loading: false
      }));
    } catch (error) {
      console.error('Failed to restart server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to restart server',
        loading: false
      });
    }
  },

  stopServer: async (serverId: string) => {
    set({ loading: true, error: null });

    try {
      await api.stopServer(serverId);

      // Optimistically update status
      set(state => ({
        servers: state.servers.map(server =>
          server.id === serverId
            ? { ...server, status: 'offline' }
            : server
        ),
        loading: false
      }));
    } catch (error) {
      console.error('Failed to stop server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to stop server',
        loading: false
      });
    }
  },

  startServer: async (serverId: string) => {
    set({ loading: true, error: null });

    try {
      await api.startServer(serverId);

      // Optimistically update status
      set(state => ({
        servers: state.servers.map(server =>
          server.id === serverId
            ? { ...server, status: 'online' }
            : server
        ),
        loading: false
      }));
    } catch (error) {
      console.error('Failed to start server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to start server',
        loading: false
      });
    }
  },

  addServer: async (serverData) => {
    set({ loading: true, error: null });

    try {
      await api.addServer(serverData as any);

      // Refresh the server list after adding
      await get().fetchServers();

      set({ loading: false });
    } catch (error) {
      console.error('Failed to add server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to add server',
        loading: false
      });
      throw error;
    }
  },

  updateServer: async (serverId: string, data: { name?: string; env?: Record<string, string> }) => {
    set({ loading: true, error: null });

    try {
      await api.updateServer(serverId, data);

      // Refresh the server list after updating
      await get().fetchServers();

      set({ loading: false });
    } catch (error) {
      console.error('Failed to update server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to update server',
        loading: false
      });
      throw error;
    }
  },

  deleteServer: async (serverId: string, removeData: boolean = true) => {
    try {
      await api.deleteServer(serverId, removeData);

      // Remove server from local state immediately (optimistic update)
      // The SSE connection will sync the real state shortly after
      set(state => ({
        servers: state.servers.filter(server => server.id !== serverId)
      }));
    } catch (error) {
      console.error('Failed to delete server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to delete server'
      });
      throw error;
    }
  },

  addPluginToServer: async (serverId: string, pluginId: string) => {
    set({ loading: true, error: null });

    try {
      // Note: This would need a backend endpoint for plugin management
      // For now, this is a placeholder
      console.warn('addPluginToServer: Backend endpoint not yet implemented');

      set({ loading: false });
    } catch (error) {
      console.error('Failed to add plugin to server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to add plugin to server',
        loading: false
      });
      throw error;
    }
  },

  removePluginFromServer: async (serverId: string, pluginId: string) => {
    set({ loading: true, error: null });

    try {
      // Note: This would need a backend endpoint for plugin management
      // For now, this is a placeholder
      console.warn('removePluginFromServer: Backend endpoint not yet implemented');

      set({ loading: false });
    } catch (error) {
      console.error('Failed to remove plugin from server:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to remove plugin from server',
        loading: false
      });
      throw error;
    }
  },

  // Selectors
  getServerById: (id: string) => {
    return get().servers.find(server => server.id === id);
  },

  getSelectedServer: () => {
    const { servers, selectedServerId } = get();
    if (!selectedServerId) return undefined;
    return servers.find(server => server.id === selectedServerId);
  }
}));
