import axios from 'axios';

const API_BASE_URL = 'http://localhost:3000/api/v1'; // replace with your API

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json'
    // Add any authorization headers if needed
  }
});

export const getUsers = () => api.get('/user');

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
export const completeSetup = (setupData: { dataPath: string; acceptEula: boolean }) =>
  api.post('/setup/complete', setupData);

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

// Infrastructure API
export const getInfrastructureContainer = (id: string) =>
  api.get<{
    container: { summary: any; type: string };
    inspectData: { Config?: { Env?: string[] }; State?: { StartedAt?: string } };
  }>(`/infrastructure/${id}`);
