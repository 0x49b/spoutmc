import axios from 'axios';
import { clearToken, getToken } from '../security/tokenVault';

const API_BASE_URL = 'http://localhost:3000/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json'
  }
});

// Attach JWT to requests when available
api.interceptors.request.use(async (config) => {
  const token = await getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle 401 - clear token and redirect to login
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      await clearToken();
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
  uuid?: string;
  avatarDataUrl?: string;
  lastLoggedInAt?: string;
  lastLoggedOutAt?: string;
  currentServer?: string;
  clientBrand?: string;
  clientMods?: string[];
  banned: boolean;
  banReason?: string;
  status: 'online' | 'offline' | 'banned' | string;
}

export interface PlayerChatMessageDTO {
  direction: 'incoming' | 'outgoing' | string;
  player: string;
  staffUserId?: number;
  conversationId?: number;
  sender?: string;
  role?: string;
  message: string;
  timestamp: string;
}

export interface PlayerSummaryDTO {
  minecraftUuid: string;
  minecraftName?: string;
  avatarDataUrl?: string;
  status?: string;
  currentServer?: string;
  lastLoggedInAt?: string;
  lastLoggedOutAt?: string;
  clientBrand?: string;
  clientMods?: string[];
  banned: boolean;
  banReason?: string;
  banUntilAt?: string;
}

export interface PlayerConversationDTO {
  id: number;
  staffUserId: number;
  staffDisplayName: string;
  lastMessage: string;
  lastOccurredAt: string;
  closed: boolean;
  closedAt?: string;
}

export interface PlayerConversationListDTO {
  conversations: PlayerConversationDTO[];
  hasOtherConversations: boolean;
}

export interface PlayerBanHistoryDTO {
  staffUserId: number;
  staffDisplayName: string;
  reason: string;
  createdAt: string;
  untilAt?: string;
  liftedAt?: string;
  permanent: boolean;
}

export interface PlayerKickHistoryDTO {
  staffUserId: number;
  staffDisplayName: string;
  reason: string;
  occurredAt: string;
}

export interface PlayerJournalEntryDTO {
  staffUserId: number;
  staffDisplayName: string;
  entry: string;
  occurredAt: string;
}

export interface BanDurationOptionDTO {
  key: string;
  label: string;
  durationSeconds: number;
}

export interface BanDurationsResponseDTO {
  options: BanDurationOptionDTO[];
}

export const getPlayers = () => api.get<PlayerDTO[]>('/player');
export const sendPlayerMessage = (
  name: string,
  message: string,
  opts?: { newConversation?: boolean; conversationId?: number }
) =>
  api.post<{ status: string; conversationId?: number }>(`/player/${encodeURIComponent(name)}/message`, {
    message,
    ...(opts?.newConversation ? { newConversation: true } : {}),
    ...(opts?.conversationId != null ? { conversationId: opts.conversationId } : {})
  });
export const getPlayerChat = (name: string, scope?: 'all') =>
  api.get<PlayerChatMessageDTO[]>(`/player/${encodeURIComponent(name)}/chat`, {
    params: scope ? { scope } : undefined
  });
export const kickPlayer = (name: string, reason: string) => api.post(`/player/${encodeURIComponent(name)}/kick`, { reason });
export const banPlayer = (
  name: string,
  reason: string,
  opts?: { untilAt?: string; permanent?: boolean }
) =>
  api.post(`/player/${encodeURIComponent(name)}/ban`, {
    reason,
    ...(opts?.untilAt ? { untilAt: opts.untilAt } : {}),
    ...(opts?.permanent !== undefined ? { permanent: opts.permanent } : {})
  });
export const unbanPlayer = (name: string) => api.post(`/player/${encodeURIComponent(name)}/unban`);

// Player detail / moderation APIs (UUID-keyed)
export const getPlayerSummary = (playerUuid: string) => api.get<PlayerSummaryDTO>(`/player/${encodeURIComponent(playerUuid)}`);
export const getPlayerConversations = (playerUuid: string) =>
  api.get<PlayerConversationListDTO>(`/player/${encodeURIComponent(playerUuid)}/conversations`);
export const getConversationMessages = (playerUuid: string, conversationId: number) =>
  api.get<PlayerChatMessageDTO[]>(`/player/${encodeURIComponent(playerUuid)}/conversations/${conversationId}/messages`);

export const closePlayerConversation = (playerUuid: string, conversationId: number) =>
  api.post<{ status: string }>(`/player/${encodeURIComponent(playerUuid)}/conversations/${conversationId}/close`, {});
export const getPlayerBans = (playerUuid: string) =>
  api.get<PlayerBanHistoryDTO[]>(`/player/${encodeURIComponent(playerUuid)}/bans`);
export const getPlayerKicks = (playerUuid: string) =>
  api.get<PlayerKickHistoryDTO[]>(`/player/${encodeURIComponent(playerUuid)}/kicks`);
export const getPlayerAliases = (playerUuid: string) =>
  api.get<{ aliases: string[] }>(`/player/${encodeURIComponent(playerUuid)}/aliases`);
export const getPlayerJournal = (playerUuid: string) =>
  api.get<PlayerJournalEntryDTO[]>(`/player/${encodeURIComponent(playerUuid)}/journal`);
export const addPlayerJournalEntry = (playerUuid: string, entry: string) =>
  api.post<PlayerJournalEntryDTO>(`/player/${encodeURIComponent(playerUuid)}/journal`, { entry });
export const getBanDurations = () => api.get<BanDurationsResponseDTO>(`/player/ban-durations`);

// Setup API functions
export interface CompleteSetupRequest {
  dataPath: string;
  acceptEula: boolean;
  adminEmail?: string;
  adminPassword?: string;
  adminDisplayName?: string;
  enableGitOps?: boolean;
  gitPollInterval?: string;
  gitRepository?: string;
  gitBranch?: string;
}

export const completeSetup = (setupData: CompleteSetupRequest) => api.post('/setup/complete', setupData);

export interface SetupStatusDTO {
  completed: boolean;
  eulaAccepted: boolean;
  adminExists: boolean;
}

export const getSetupStatus = () => api.get<SetupStatusDTO>('/setup/status');

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

export const updateServer = (
  id: string,
  data: { name?: string; env?: Record<string, string> },
  opts?: { applyImmediately?: boolean }
) =>
  api.put(`/server/${id}`, {
    ...data,
    ...(opts?.applyImmediately !== undefined ? { applyImmediately: opts.applyImmediately } : {})
  });

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

export const updateServerBinaryFile = (serverId: string, path: string, contentBase64: string, volume?: string) =>
  api.put(
    `/server/${serverId}/file/binary`,
    { contentBase64 },
    { params: { path, volume } }
  );

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

// Self-update API
export interface UpdateStatusDTO {
  configured: boolean;
  currentVersion: string;
  latestVersion?: string;
  releaseUrl?: string;
  releaseNotes?: string;
  updateAvailable: boolean;
  migrationRequired: boolean;
  state: 'idle' | 'checking' | 'available' | 'downloading' | 'installing' | 'restarting' | 'error' | string;
  lastCheckedAt?: string;
  lastError?: string;
  lastInstalledAt?: string;
  lastBackupPath?: string;
  currentAssetName?: string;
  checkIntervalHours?: number;
}

export const getUpdateStatus = () => api.get<UpdateStatusDTO>('/update/status');
export const checkForUpdates = () => api.post<UpdateStatusDTO>('/update/check');
export const startSelfUpdate = () => api.post<{ status: string; message: string }>('/update/start');

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
