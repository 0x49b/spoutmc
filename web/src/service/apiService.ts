import axios from 'axios';

const API_BASE_URL = 'http://localhost:3000/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json'
  }
});

// Attach JWT to requests when available
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle 401 - clear token and redirect to login
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token');
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

/**
 * EventSource cannot send Authorization headers. Protected SSE routes accept the same JWT
 * via the access_token query parameter (see server JWT middleware).
 */
export function withSSEAuth(url: string): string {
  const token = localStorage.getItem('auth_token');
  if (!token) return url;
  try {
    const u = new URL(url);
    u.searchParams.set('access_token', token);
    return u.toString();
  } catch {
    return url;
  }
}

/** For fetch() calls that bypass axios but still need the session JWT. */
export function getAuthFetchHeaders(): Record<string, string> {
  const token = localStorage.getItem('auth_token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json'
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

// Auth API
export interface LoginResponse {
  token: string;
  user: UserDTO;
}

export interface PermissionDTO {
  id: number;
  key: string;
  description?: string;
}

export interface UserDTO {
  id: number;
  createdAt: string;
  minecraftId?: string;
  minecraftName?: string;
  displayName: string;
  email: string;
  roles: { id: number; name: string; displayName?: string }[];
  /** Effective permission keys (roles + direct). */
  permissions?: string[];
  /** Directly granted permissions only. */
  directPermissions?: PermissionDTO[];
  avatar?: string;
}

export const login = (email: string, password: string) =>
  api.post<LoginResponse>('/auth/login', { email, password });

export const verifyToken = () => api.get<UserDTO>('/auth/verify');

// User API
export const getUsers = () => api.get<UserDTO[]>('/user');
export const getUser = (id: string) => api.get<UserDTO>(`/user/${id}`);
export const createUser = (data: {
  email: string;
  password: string;
  displayName: string;
  minecraftName?: string;
  roleIds?: number[];
  permissionIds?: number[];
}) => api.post<UserDTO>('/user', data);
export const updateUser = (id: string, data: {
  email?: string;
  displayName?: string;
  minecraftName?: string;
  roleIds?: number[];
  permissionIds?: number[];
}) => api.put<UserDTO>(`/user/${id}`, data);
export const deleteUser = (id: string) => api.delete(`/user/${id}`);
export const updateProfile = (data: {
  email?: string;
  password?: string;
  displayName?: string;
  minecraftName?: string;
}) => api.put<UserDTO>('/user/profile', data);

// Role API
export interface RoleDTO {
  id: number;
  name: string;
  displayName: string;
  slug: string;
  userCount?: number;
  permissions?: PermissionDTO[];
}

export const getRoles = () => api.get<RoleDTO[]>('/role');
export const getRole = (id: string) => api.get<RoleDTO>(`/role/${id}`);
export const getPermissions = () => api.get<PermissionDTO[]>('/permission');
export const createPermission = (body: { key: string; description?: string }) =>
  api.post<PermissionDTO>('/permission', body);
export const updatePermission = (id: number, body: { key?: string; description?: string }) =>
  api.put<PermissionDTO>(`/permission/${id}`, body);
export const deletePermission = (id: number) => api.delete(`/permission/${id}`);
export const createRole = (displayName: string) =>
  api.post<RoleDTO>('/role', { displayName });
export const updateRole = (
  id: string,
  body: { displayName: string; permissionIds: number[] }
) => api.put<RoleDTO>(`/role/${id}`, body);
export const deleteRole = (id: string) => api.delete(`/role/${id}`);

// Players API
export interface PlayerDTO {
  name: string;
  avatarDataUrl?: string;
  lastLoggedInAt?: string;
  lastLoggedOutAt?: string;
  currentServer?: string;
  banned: boolean;
  banReason?: string;
  status: 'online' | 'offline' | 'banned' | string;
}

export interface PlayerChatMessageDTO {
  direction: 'incoming' | 'outgoing' | string;
  player: string;
  sender?: string;
  role?: string;
  message: string;
  timestamp: string;
}

export const getPlayers = () => api.get<PlayerDTO[]>('/player');
export const sendPlayerMessage = (name: string, message: string, sender?: string, role?: string) =>
  api.post(`/player/${encodeURIComponent(name)}/message`, { message, sender, role });
export const getPlayerChat = (name: string) => api.get<PlayerChatMessageDTO[]>(`/player/${encodeURIComponent(name)}/chat`);
export const kickPlayer = (name: string, reason: string) => api.post(`/player/${encodeURIComponent(name)}/kick`, { reason });
export const banPlayer = (name: string, reason: string) => api.post(`/player/${encodeURIComponent(name)}/ban`, { reason });
export const unbanPlayer = (name: string) => api.post(`/player/${encodeURIComponent(name)}/unban`);

// Setup API functions
export const completeSetup = (setupData: {
  dataPath: string;
  acceptEula: boolean;
  adminEmail?: string;
  adminPassword?: string;
  adminDisplayName?: string;
}) => api.post('/setup/complete', setupData);

// Server API functions
export const getServers = () => api.get('/server');
export const getServerById = (id: string) => api.get(`/server/${id}`);
export const startServer = (id: string) => api.post(`/server/${id}/start`);
export const stopServer = (id: string) => api.post(`/server/${id}/stop`);
export const restartServer = (id: string) => api.post(`/server/${id}/restart`);
export const addServer = (serverData: {
  name: string;
  image: string;
  port?: number;
  proxy?: boolean;
  lobby?: boolean;
  env: Record<string, string>;
}) => api.post('/server', serverData);

export const deleteServer = (id: string, removeData: boolean = true) =>
  api.delete(`/server/${id}`, { params: { removeData } });

export const updateServer = (id: string, data: { name?: string; env?: Record<string, string> }) =>
  api.put(`/server/${id}`, data);

export const getServerEnv = (id: string) =>
  api.get<Record<string, string>>(`/server/${id}/env`);

// Config file API functions
export const listConfigFiles = (serverId: string) =>
  api.get<{ files: string[] }>(`/server/${serverId}/config/files`);

export const getConfigFile = (serverId: string, filename: string) =>
  api.get<{ filename: string; content: string }>(`/server/${serverId}/config/${filename}`);

export const updateConfigFile = (serverId: string, filename: string, content: string) =>
  api.put(`/server/${serverId}/config/${filename}`, { content });

// File browser API functions
export interface FileNode {
  name: string;
  path: string;
  isDir: boolean;
  size?: number;
  modTime?: string;
  children?: FileNode[];
}

export interface VolumeFiles {
  containerPath: string;
  hostPath: string;
  files: FileNode;
}

export const listServerFiles = (serverId: string) =>
  api.get<{ volumes: VolumeFiles[] }>(`/server/${serverId}/files`);

export const getServerFile = (serverId: string, path: string, volume?: string) =>
  api.get<{ path: string; content: string }>(`/server/${serverId}/file`, {
    params: { path, volume },
  });

export const updateServerFile = (serverId: string, path: string, content: string, volume?: string) =>
  api.put(`/server/${serverId}/file`, { content }, { params: { path, volume } });

// Plugin registry API
export interface PluginRegistryEntryDTO {
  id: string;
  name: string;
  url: string;
  description?: string;
  systemManaged: boolean;
  serverNames: string[];
  kinds?: string[];
}

export const getPlugins = () => api.get<PluginRegistryEntryDTO[]>('/plugin');

export const createRegistryPlugin = (body: {
  name: string;
  url: string;
  description?: string;
  serverNames: string[];
}) => api.post<PluginRegistryEntryDTO>('/plugin', body);

export const updateRegistryPlugin = (
  id: string,
  body: { name: string; url: string; description?: string; serverNames: string[] }
) => api.put<PluginRegistryEntryDTO>(`/plugin/${id}`, body);

export const deleteRegistryPlugin = (id: string) => api.delete(`/plugin/${id}`);

// GitOps API
export interface GitOpsSyncSummary {
  added: number;
  updated: number;
  removed: number;
  created: number;
  recreated: number;
  pruned: number;
  driftCorrections: number;
}

export interface GitOpsStatus {
  enabled: boolean;
  state: 'disabled' | 'initializing' | 'syncing' | 'synced' | 'error' | string;
  lastSyncAt?: string;
  lastSuccessfulSyncAt?: string;
  lastChangeDetectedAt?: string;
  lastSyncCommit?: string;
  lastSyncCommitMessage?: string;
  lastError?: string;
  lastSummary?: GitOpsSyncSummary;
}

export const getGitOpsStatus = () => api.get<GitOpsStatus>('/git/status');
export const triggerGitOpsSync = () => api.post('/git/sync');

// Notifications API
export interface SystemNotification {
  id: number;
  key: string;
  severity: 'info' | 'warning' | 'danger' | 'success' | string;
  title: string;
  message: string;
  source: string;
  isOpen: boolean;
  createdAt: string;
  updatedAt: string;
  dismissedAt?: string;
}

export const getNotifications = () => api.get<SystemNotification[]>('/notification');
export const dismissNotification = (id: number) => api.post(`/notification/${id}/dismiss`);

// Infrastructure API
export const getInfrastructureContainer = (id: string) =>
  api.get<{
    container: { summary: any; type: string };
    inspectData: { Config?: { Env?: string[] }; State?: { StartedAt?: string } };
  }>(`/infrastructure/${id}`);
