import { create } from 'zustand';
import { UserProfile, Role, Permission } from '../types';
import { mockLogin, mockVerifyToken } from '../lib/mockAuth';

interface AuthState {
  user: UserProfile | null;
  users: UserProfile[];
  loading: boolean;
  error: string | null;
  
  // Actions
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
  fetchUsers: () => Promise<void>;
  addUser: (userData: { email: string; password: string; displayName: string; roles: Role[] }) => Promise<void>;
  updateUser: (userId: string, userData: { email: string; displayName: string; roles: Role[] }) => Promise<void>;
  deleteUser: (userId: string) => Promise<void>;
  
  // Permissions
  hasPermission: (action: string, subject: string) => boolean;
}

const roleHierarchy: Record<Role, number> = {
  admin: 3,
  moderator: 2,
  viewer: 1
};

const permissions: Record<Role, Permission[]> = {
  admin: [
    { action: 'manage', subject: 'all' }
  ],
  moderator: [
    { action: 'read', subject: 'all' },
    { action: 'manage', subject: 'servers' },
    { action: 'manage', subject: 'players' }
  ],
  viewer: [
    { action: 'read', subject: 'all' }
  ]
};

// Mock users for development
const mockUsers: UserProfile[] = [
  {
    id: '1',
    email: 'admin@example.com',
    roles: ['admin', 'moderator', 'viewer'],
    displayName: 'Admin User',
    created_at: new Date().toISOString(),
    lastLoginAt: new Date().toISOString(),
    aud: 'authenticated',
    app_metadata: {},
    user_metadata: {},
    identities: []
  },
  {
    id: '2',
    email: 'mod@example.com',
    roles: ['moderator', 'viewer'],
    displayName: 'Moderator User',
    created_at: new Date().toISOString(),
    lastLoginAt: new Date().toISOString(),
    aud: 'authenticated',
    app_metadata: {},
    user_metadata: {},
    identities: []
  },
  {
    id: '3',
    email: 'viewer@example.com',
    roles: ['viewer'],
    displayName: 'Viewer User',
    created_at: new Date().toISOString(),
    lastLoginAt: new Date().toISOString(),
    aud: 'authenticated',
    app_metadata: {},
    user_metadata: {},
    identities: []
  }
];

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  users: mockUsers,
  loading: false,
  error: null,
  
  login: async (email: string, password: string) => {
    set({ loading: true, error: null });
    
    try {
      const { token } = await mockLogin(email, password);
      localStorage.setItem('auth_token', token);
      const user = mockVerifyToken(token);
      const updatedUser = {
        ...user,
        lastLoginAt: new Date().toISOString()
      };
      
      set({ user: updatedUser, loading: false });
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
      
      if (token) {
        const user = mockVerifyToken(token);
        set({ user });
      }
      
      set({ loading: false });
    } catch (error) {
      localStorage.removeItem('auth_token');
      set({ 
        user: null,
        error: error instanceof Error ? error.message : 'Failed to check auth status',
        loading: false 
      });
    }
  },
  
  fetchUsers: async () => {
    set({ loading: true, error: null });
    
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      set({ users: mockUsers, loading: false });
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to fetch users',
        loading: false 
      });
    }
  },
  
  addUser: async (userData) => {
    set({ loading: true, error: null });
    
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const newUser: UserProfile = {
        id: Math.random().toString(36).substr(2, 9),
        email: userData.email,
        roles: userData.roles,
        displayName: userData.displayName,
        created_at: new Date().toISOString(),
        lastLoginAt: new Date().toISOString(),
        aud: 'authenticated',
        app_metadata: {},
        user_metadata: {},
        identities: []
      };
      
      set(state => ({
        users: [...state.users, newUser],
        loading: false
      }));
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to add user',
        loading: false 
      });
      throw error;
    }
  },
  
  updateUser: async (userId: string, userData) => {
    set({ loading: true, error: null });
    
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      set(state => ({
        users: state.users.map(user =>
          user.id === userId
            ? { ...user, ...userData }
            : user
        ),
        loading: false
      }));
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to update user',
        loading: false 
      });
      throw error;
    }
  },

  deleteUser: async (userId: string) => {
    set({ loading: true, error: null });
    
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      set(state => ({
        users: state.users.filter(user => user.id !== userId),
        loading: false
      }));
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to delete user',
        loading: false 
      });
      throw error;
    }
  },
  
  hasPermission: (action: string, subject: string) => {
    const { user } = get();
    if (!user || !user.roles || !Array.isArray(user.roles)) return false;
    
    // Check each role's permissions
    return user.roles.some(role => {
      // Admin role has all permissions
      if (role === 'admin') return true;
      
      // Check if the role exists in permissions
      if (!permissions[role]) return false;
      
      // Check role-specific permissions
      const rolePermissions = permissions[role];
      return rolePermissions.some(permission => {
        // Check for exact match
        if (permission.action === action && permission.subject === subject) return true;
        
        // Check for wildcard permissions
        if (permission.action === action && permission.subject === 'all') return true;
        if (permission.action === 'manage' && permission.subject === subject) return true;
        if (permission.action === 'manage' && permission.subject === 'all') return true;
        
        return false;
      });
    });
  }
}));