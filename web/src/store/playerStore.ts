import { create } from 'zustand';
import { Player } from '../types';
import * as api from '../service/apiService';

interface PlayerState {
  players: Player[];
  loading: boolean;
  error: string | null;
  eventSource: EventSource | null;
  actionInProgressByPlayer: Record<string, boolean>;

  fetchPlayers: () => Promise<void>;
  connectSSE: () => void;
  disconnectSSE: () => void;
  sendMessage: (playerName: string, message: string) => Promise<void>;
  kickPlayer: (playerName: string, reason: string) => Promise<void>;
  banPlayer: (playerName: string, reason: string) => Promise<void>;

  getPlayerById: (id: string) => Player | undefined;
  getBannedPlayers: () => Player[];
}

const API_BASE_URL = 'http://localhost:3000/api/v1';

const mapPlayer = (dto: api.PlayerDTO): Player => ({
  id: dto.name,
  username: dto.name,
  avatarDataUrl: dto.avatarDataUrl,
  currentServer: dto.currentServer,
  lastLoggedInAt: dto.lastLoggedInAt,
  lastLoggedOutAt: dto.lastLoggedOutAt,
  status: dto.status === 'banned' ? 'banned' : dto.status === 'online' ? 'online' : 'offline',
  banned: dto.banned,
  banReason: dto.banReason
});

const mapPlayers = (dtos: api.PlayerDTO[]): Player[] => dtos.map(mapPlayer);

export const usePlayerStore = create<PlayerState>((set, get) => ({
  players: [],
  loading: false,
  error: null,
  eventSource: null,
  actionInProgressByPlayer: {},

  fetchPlayers: async () => {
    set({ loading: true, error: null });
    try {
      const response = await api.getPlayers();
      set({ players: mapPlayers(response.data), loading: false });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch players',
        loading: false
      });
    }
  },

  connectSSE: () => {
    const current = get().eventSource;
    if (current) {
      current.close();
    }

    try {
      const eventSource = new EventSource(`${API_BASE_URL}/player/stream`);
      eventSource.onopen = () => set({ error: null });
      eventSource.onmessage = event => {
        try {
          const data = JSON.parse(event.data);
          const payload = data.Data ? (typeof data.Data === 'string' ? JSON.parse(data.Data) : data.Data) : data;
          const players = Array.isArray(payload) ? mapPlayers(payload) : [];
          set({ players, loading: false });
        } catch (err) {
          console.error('Failed to parse players SSE payload', err);
        }
      };
      eventSource.onerror = () => {
        eventSource.close();
        setTimeout(() => {
          const state = get();
          if (!state.eventSource || state.eventSource.readyState === EventSource.CLOSED) {
            get().connectSSE();
          }
        }, 5000);
      };

      set({ eventSource });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to establish players SSE connection' });
    }
  },

  disconnectSSE: () => {
    const eventSource = get().eventSource;
    if (eventSource) {
      eventSource.close();
      set({ eventSource: null });
    }
  },

  sendMessage: async (playerName: string, message: string) => {
    set(state => ({
      actionInProgressByPlayer: {
        ...state.actionInProgressByPlayer,
        [playerName]: true
      }
    }));
    try {
      await api.sendPlayerMessage(playerName, message);
    } finally {
      set(state => ({
        actionInProgressByPlayer: {
          ...state.actionInProgressByPlayer,
          [playerName]: false
        }
      }));
    }
  },

  kickPlayer: async (playerName: string, reason: string) => {
    set(state => ({
      actionInProgressByPlayer: {
        ...state.actionInProgressByPlayer,
        [playerName]: true
      }
    }));
    try {
      await api.kickPlayer(playerName, reason);
    } finally {
      set(state => ({
        actionInProgressByPlayer: {
          ...state.actionInProgressByPlayer,
          [playerName]: false
        }
      }));
    }
  },

  banPlayer: async (playerName: string, reason: string) => {
    set(state => ({
      actionInProgressByPlayer: {
        ...state.actionInProgressByPlayer,
        [playerName]: true
      }
    }));
    try {
      await api.banPlayer(playerName, reason);
      set(state => ({
        players: state.players.map(player =>
          player.username === playerName
            ? { ...player, status: 'banned', banned: true, banReason: reason, currentServer: '' }
            : player
        )
      }));
    } finally {
      set(state => ({
        actionInProgressByPlayer: {
          ...state.actionInProgressByPlayer,
          [playerName]: false
        }
      }));
    }
  },

  getPlayerById: (id: string) => get().players.find(player => player.id === id),
  getBannedPlayers: () => get().players.filter(player => player.status === 'banned' || player.banned)
}));