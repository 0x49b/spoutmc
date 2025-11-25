import { create } from 'zustand';
import { InfrastructureContainer } from '../types';
import axios from 'axios';

interface InfrastructureState {
  containers: InfrastructureContainer[];
  loading: boolean;
  error: string | null;
  eventSource: EventSource | null;

  // Actions
  fetchInfrastructure: () => Promise<void>;
  connectSSE: () => void;
  disconnectSSE: () => void;
  restartContainer: (containerId: string) => Promise<void>;
  stopContainer: (containerId: string) => Promise<void>;

  // Selectors
  getContainerById: (id: string) => InfrastructureContainer | undefined;
}

const API_BASE_URL = 'http://localhost:3000/api/v1';

export const useInfrastructureStore = create<InfrastructureState>((set, get) => ({
  containers: [],
  loading: false,
  error: null,
  eventSource: null,

  fetchInfrastructure: async () => {
    set({ loading: true, error: null });

    try {
      const response = await axios.get(`${API_BASE_URL}/infrastructure`);
      const containers: InfrastructureContainer[] = Array.isArray(response.data) ? response.data : [];

      set({ containers, loading: false });
    } catch (error) {
      console.error('Failed to fetch infrastructure:', error);
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch infrastructure',
        loading: false,
        containers: []
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
      const eventSource = new EventSource(`${API_BASE_URL}/infrastructure/stream`);

      eventSource.onopen = () => {
        console.log('SSE connection established for infrastructure');
        set({ error: null });
      };

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);

          // Check if data is wrapped in an SSE event structure
          let containersData: InfrastructureContainer[];
          if (data.Data) {
            // Data is wrapped in SSE event structure
            containersData = typeof data.Data === 'string'
              ? JSON.parse(data.Data)
              : data.Data;
          } else {
            // Data is the container array directly
            containersData = data;
          }

          // Ensure it's an array
          const containers = Array.isArray(containersData) ? containersData : [];

          set({ containers, loading: false });
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
            console.log('Attempting to reconnect infrastructure SSE...');
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
      console.log('Infrastructure SSE connection closed');
    }
  },

  restartContainer: async (containerId: string) => {
    try {
      await axios.post(`${API_BASE_URL}/infrastructure/${containerId}/restart`);

      // Optimistically update status
      set(state => ({
        containers: state.containers.map(container =>
          container.summary.ID === containerId
            ? { ...container, summary: { ...container.summary, State: 'restarting' } }
            : container
        )
      }));
    } catch (error) {
      console.error('Failed to restart infrastructure container:', error);
      throw error;
    }
  },

  stopContainer: async (containerId: string) => {
    try {
      await axios.post(`${API_BASE_URL}/infrastructure/${containerId}/stop`);

      // Optimistically update status
      set(state => ({
        containers: state.containers.map(container =>
          container.summary.ID === containerId
            ? { ...container, summary: { ...container.summary, State: 'exited' } }
            : container
        )
      }));
    } catch (error) {
      console.error('Failed to stop infrastructure container:', error);
      throw error;
    }
  },

  // Selectors
  getContainerById: (id: string) => {
    return get().containers.find(container => container.summary.ID === id);
  }
}));
