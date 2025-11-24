import { create } from 'zustand';
import { Player } from '../types';

interface PlayerState {
  players: Player[];
  loading: boolean;
  error: string | null;

  // Actions
  fetchPlayers: () => Promise<void>;
  banPlayer: (playerId: string, reason: string) => Promise<void>;
  unbanPlayer: (playerId: string) => Promise<void>;

  // Selectors
  getPlayerById: (id: string) => Player | undefined;
  getBannedPlayers: () => Player[];
}

// Mock data for initial state
const mockPlayers: Player[] = [
  {
    id: '1',
    username: 'flind_',
    serverId: '1',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '2',
    username: 'Player2',
    serverId: '2',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '3',
    username: 'PermaBannedPlayer',
    serverId: '1',
    lastSeen: new Date(),
    status: 'banned',
    banReason: 'Inappropriate behavior',
    bannedAt: new Date(),
    permanentBanned: true
  },
  {
    id: '4',
    username: 'Player3',
    serverId: '1',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '5',
    username: 'Player4',
    serverId: '3',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '6',
    username: 'ToxicGamer',
    serverId: '2',
    lastSeen: new Date(),
    status: 'banned',
    banReason: 'Toxic language',
    bannedAt: new Date(),
    bannedUntil: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000)
  },
  {
    id: '7',
    username: 'Player5',
    serverId: '1',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '8',
    username: 'Player6',
    serverId: '3',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '9',
    username: 'Cheater01',
    serverId: '2',
    lastSeen: new Date(),
    status: 'banned',
    banReason: 'Cheating',
    bannedAt: new Date()
  },
  {
    id: '10',
    username: 'Player7',
    serverId: '2',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '11',
    username: 'Player8',
    serverId: '1',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '12',
    username: 'Player9',
    serverId: '3',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '13',
    username: 'Spammer',
    serverId: '1',
    lastSeen: new Date(),
    status: 'banned',
    banReason: 'Spam messages',
    bannedAt: new Date()
  },
  {
    id: '14',
    username: 'Player10',
    serverId: '2',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '15',
    username: 'Player11',
    serverId: '3',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '16',
    username: 'Player12',
    serverId: '1',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '17',
    username: 'HackerX',
    serverId: '3',
    lastSeen: new Date(),
    status: 'banned',
    banReason: 'Exploiting game bugs',
    bannedAt: new Date()
  },
  {
    id: '18',
    username: 'Player13',
    serverId: '2',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '19',
    username: 'Player14',
    serverId: '3',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '20',
    username: 'Player15',
    serverId: '1',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '21',
    username: 'Player16',
    serverId: '2',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '22',
    username: 'Griefer',
    serverId: '2',
    lastSeen: new Date(),
    status: 'banned',
    banReason: 'Griefing other players',
    bannedAt: new Date()
  },
  {
    id: '23',
    username: 'Player17',
    serverId: '3',
    lastSeen: new Date(),
    status: 'offline'
  },
  {
    id: '24',
    username: 'Player18',
    serverId: '1',
    lastSeen: new Date(),
    status: 'online'
  },
  {
    id: '25',
    username: 'Player19',
    serverId: '3',
    lastSeen: new Date(),
    status: 'offline'
  }
];

export const usePlayerStore = create<PlayerState>((set, get) => ({
  players: mockPlayers,
  loading: false,
  error: null,

  fetchPlayers: async () => {
    set({ loading: true, error: null });

    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      set({ players: mockPlayers, loading: false });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch players',
        loading: false
      });
    }
  },

  banPlayer: async (playerId: string, reason: string) => {
    set({ loading: true, error: null });

    try {
      await new Promise(resolve => setTimeout(resolve, 500));

      set(state => ({
        players: state.players.map(player =>
          player.id === playerId
            ? {
              ...player,
              status: 'banned',
              banReason: reason,
              bannedAt: new Date()
            }
            : player
        ),
        loading: false
      }));
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to ban player',
        loading: false
      });
    }
  },

  unbanPlayer: async (playerId: string) => {
    set({ loading: true, error: null });

    try {
      await new Promise(resolve => setTimeout(resolve, 500));

      set(state => ({
        players: state.players.map(player =>
          player.id === playerId
            ? {
              ...player,
              status: 'offline',
              banReason: undefined,
              bannedAt: undefined
            }
            : player
        ),
        loading: false
      }));
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to unban player',
        loading: false
      });
    }
  },

  // Selectors
  getPlayerById: (id: string) => {
    return get().players.find(player => player.id === id);
  },

  getBannedPlayers: () => {
    return get().players.filter(player => player.status === 'banned');
  }
}));