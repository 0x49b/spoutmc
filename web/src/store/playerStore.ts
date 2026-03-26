import {create} from 'zustand';
import {Player} from '../types';
import * as api from '../service/apiService';

interface PlayerState {
  players: Player[];
  loading: boolean;
  error: string | null;
  eventSource: EventSource | null;
  sseShouldReconnect: boolean;
  actionInProgressByPlayer: Record<string, boolean>;

  fetchPlayers: () => Promise<void>;
  connectSSE: () => void;
  disconnectSSE: () => void;
  sendMessage: (playerName: string, message: string, sender?: string, role?: string) => Promise<void>;
  kickPlayer: (playerName: string, reason: string) => Promise<void>;
  banPlayer: (playerName: string, reason: string) => Promise<void>;
  unbanPlayer: (playerName: string) => Promise<void>;
  getPlayerChat: (playerName: string) => Promise<api.PlayerChatMessageDTO[]>;

  getPlayerById: (id: string) => Player | undefined;
  getBannedPlayers: () => Player[];

  // Player detail (UUID-keyed) helpers
  getPlayerSummary: (playerUuid: string) => Promise<api.PlayerSummaryDTO>;
  getPlayerConversations: (playerUuid: string) => Promise<api.PlayerConversationListDTO>;
  getConversationMessages: (playerUuid: string, staffUserId: number) => Promise<api.PlayerChatMessageDTO[]>;
  getPlayerBans: (playerUuid: string) => Promise<api.PlayerBanHistoryDTO[]>;
  getPlayerKicks: (playerUuid: string) => Promise<api.PlayerKickHistoryDTO[]>;
  getBanDurations: () => Promise<api.BanDurationOptionDTO[]>;
}

const API_BASE_URL = 'http://localhost:3000/api/v1';

const mapPlayer = (dto: api.PlayerDTO): Player => ({
  id: dto.uuid ?? dto.name,
  username: dto.name,
  avatarDataUrl: dto.avatarDataUrl,
  currentServer: dto.currentServer,
  lastLoggedInAt: dto.lastLoggedInAt,
  lastLoggedOutAt: dto.lastLoggedOutAt,
  clientBrand: dto.clientBrand,
  clientMods: dto.clientMods,
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
  sseShouldReconnect: false,
  actionInProgressByPlayer: {},

  fetchPlayers: async () => {
    set({ loading: true, error: null });
    try {
      const response = await api.getPlayers();
      const mapped = mapPlayers(response.data);

      set({ players: mapped, loading: false });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch players',
        loading: false
      });
    }
  },

  connectSSE: () => {
    set({ sseShouldReconnect: true });

    const current = get().eventSource;
    if (current) {
      current.close();
    }

    try {
      const eventSource = new EventSource(api.withSSEAuth(`${API_BASE_URL}/player/stream`));
      eventSource.onopen = () => {
        if (get().eventSource !== eventSource) return;
        set({ error: null });
      };
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
        if (get().eventSource !== eventSource) return;
        eventSource.close();
        set({ eventSource: null });

        setTimeout(() => {
          const state = get();
          if (
            state.sseShouldReconnect &&
            (!state.eventSource || state.eventSource.readyState === EventSource.CLOSED)
          ) {
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
    set({ sseShouldReconnect: false });
    const eventSource = get().eventSource;
    if (eventSource) {
      eventSource.close();
      set({ eventSource: null });
    }
  },

  sendMessage: async (playerName: string, message: string, sender?: string, role?: string) => {
    set(state => ({
      actionInProgressByPlayer: {
        ...state.actionInProgressByPlayer,
        [playerName]: true
      }
    }));
    try {
      await api.sendPlayerMessage(playerName, message, sender, role);
    } finally {
      set(state => ({
        actionInProgressByPlayer: {
          ...state.actionInProgressByPlayer,
          [playerName]: false
        }
      }));
    }
  },

  getPlayerChat: async (playerName: string) => {
    const response = await api.getPlayerChat(playerName);
    return response.data;
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
          player.id === playerName
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

  unbanPlayer: async (playerName: string) => {
    set(state => ({
      actionInProgressByPlayer: {
        ...state.actionInProgressByPlayer,
        [playerName]: true
      }
    }));
    try {
      await api.unbanPlayer(playerName);
      set(state => ({
        players: state.players.map(player =>
          player.id === playerName
            ? {
              ...player,
              status: player.currentServer ? 'online' : 'offline',
              banned: false,
              banReason: undefined
            }
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

  getPlayerSummary: async (playerUuid: string) => {
    const res = await api.getPlayerSummary(playerUuid);
    return res.data;
  },

  getPlayerConversations: async (playerUuid: string) => {
    const res = await api.getPlayerConversations(playerUuid);
    return res.data;
  },

  getConversationMessages: async (playerUuid: string, staffUserId: number) => {
    const res = await api.getConversationMessages(playerUuid, staffUserId);
    return res.data;
  },

  getPlayerBans: async (playerUuid: string) => {
    const res = await api.getPlayerBans(playerUuid);
    return res.data;
  },

  getPlayerKicks: async (playerUuid: string) => {
    const res = await api.getPlayerKicks(playerUuid);
    return res.data;
  },

  getBanDurations: async () => {
    const res = await api.getBanDurations();
    return res.data.options;
  },

  getPlayerById: (id: string) => get().players.find(player => player.id === id),
  getBannedPlayers: () => get().players.filter(player => player.status === 'banned' || player.banned)
}));