import { create } from 'zustand';
import { UserProfile, Role, Permission } from '../types';
import * as api from '../service/apiService';

interface AuthState {
  user: UserProfile | null;
  users: UserProfile[];
  roles: api.RoleDTO[];
  loading: boolean;
  error: string | null;

  // Actions
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
  fetchUsers: () => Promise<void>;
  fetchRoles: () => Promise<void>;
  addUser: (userData: {
    email: string;
    password: string;
    displayName: string;
    minecraftName?: string;
    roleIds?: number[];
  }) => Promise<void>;
  updateUser: (
    userId: string,
    userData: {
      email?: string;
      displayName?: string;
      minecraftName?: string;
      roleIds?: number[];
    }
  ) => Promise<void>;
  deleteUser: (userId: string) => Promise<void>;
  updateProfile: (data: {
    email?: string;
    password?: string;
    displayName?: string;
    minecraftName?: string;
  }) => Promise<void>;

  // Permissions
  hasPermission: (action: string, subject: string) => boolean;
}

const roleHierarchy: Record<string, number> = {
  admin: 5,
  manager: 4,
  editor: 3,
  mod: 2,
  support: 1
};

const permissions: Record<string, Permission[]> = {
  admin: [{ action: 'manage', subject: 'all' }],
  manager: [
    { action: 'read', subject: 'all' },
    { action: 'manage', subject: 'servers' },
    { action: 'manage', subject: 'players' },
    { action: 'manage', subject: 'users' }
  ],
  editor: [
    { action: 'read', subject: 'all' },
    { action: 'manage', subject: 'servers' }
  ],
  mod: [
    { action: 'read', subject: 'all' },
    { action: 'manage', subject: 'players' }
  ],
  support: [{ action: 'read', subject: 'all' }]
};

function userDtoToProfile(dto: api.UserDTO): UserProfile {
  return {
    id: String(dto.id),
    email: dto.email,
    roles: dto.roles.map((r) => r.name as Role),
    displayName: dto.displayName,
    minecraftName: dto.minecraftName,
    created_at: dto.createdAt,
    aud: 'authenticated',
    app_metadata: {},
    user_metadata: {},
    identities: []
  };
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  users: [],
  roles: [],
  loading: false,
  error: null,

  login: async (email: string, password: string) => {
    set({ loading: true, error: null });
    try {
      const { data } = await api.login(email, password);
      localStorage.setItem('auth_token', data.token);
      set({ user: userDtoToProfile(data.user), loading: false });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to login',
        loading: false
      });
      throw error;
    }
  },

  logout: async () => {
    set({ loading: true, error: null });
    try {
      localStorage.removeItem('auth_token');
      set({ user: null, loading: false });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to logout',
        loading: false
      });
      throw error;
    }
  },

  checkAuth: async () => {
    set({ loading: true, error: null });
    try {
      const token = localStorage.getItem('auth_token');
      if (!token) {
        set({ user: null, loading: false });
        return;
      }
      const { data } = await api.verifyToken();
      set({ user: userDtoToProfile(data), loading: false });
    } catch (error) {
      localStorage.removeItem('auth_token');
      set({ user: null, loading: false });
    }
  },

  fetchUsers: async () => {
    // Do NOT set loading: true - auth store loading is used by ProtectedRoute.
    // Setting it would unmount the page during fetch, then remount triggers fetch again = infinite loop.
    try {
      const { data } = await api.getUsers();
      set({ users: data.map((u) => userDtoToProfile(u)) });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch users'
      });
    }
  },

  fetchRoles: async () => {
    // Do NOT set loading: true - same reason as fetchUsers.
    try {
      const { data } = await api.getRoles();
      set({ roles: data });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch roles'
      });
    }
  },

  addUser: async (userData) => {
    set({ error: null });
    try {
      const { data } = await api.createUser(userData);
      const newUser = userDtoToProfile(data);
      set((state) => ({
        users: [...state.users, newUser]
      }));
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to add user'
      });
      throw error;
    }
  },

  updateUser: async (userId: string, userData) => {
    set({ error: null });
    try {
      const { data } = await api.updateUser(userId, userData);
      const updated = userDtoToProfile(data);
      set((state) => ({
        users: state.users.map((u) => (u.id === userId ? updated : u)),
        user: state.user?.id === userId ? updated : state.user
      }));
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to update user'
      });
      throw error;
    }
  },

  deleteUser: async (userId: string) => {
    set({ error: null });
    try {
      await api.deleteUser(userId);
      set((state) => ({
        users: state.users.filter((u) => u.id !== userId)
      }));
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to delete user'
      });
      throw error;
    }
  },

  updateProfile: async (data) => {
    set({ error: null });
    try {
      const { data: userData } = await api.updateProfile(data);
      set({ user: userDtoToProfile(userData) });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to update profile'
      });
      throw error;
    }
  },

  hasPermission: (action: string, subject: string) => {
    const { user } = get();
    if (!user || !user.roles || !Array.isArray(user.roles)) return false;

    return user.roles.some((role) => {
      if (role === 'admin') return true;
      if (!permissions[role]) return false;
      const rolePermissions = permissions[role];
      return rolePermissions.some((p) => {
        if (p.action === action && p.subject === subject) return true;
        if (p.action === action && p.subject === 'all') return true;
        if (p.action === 'manage' && p.subject === subject) return true;
        if (p.action === 'manage' && p.subject === 'all') return true;
        return false;
      });
    });
  }
}));
