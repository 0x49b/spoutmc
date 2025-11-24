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

// Setup API functions
export const completeSetup = (setupData: { dataPath: string; acceptEula: boolean }) =>
  api.post('/setup/complete', setupData);

// Server API functions
export const getServers = () => api.get('/server');
export const getServerById = (id: string) => api.get(`/server/${id}`);
export const startServer = (id: string) => api.post(`/server/${id}/start`);
export const stopServer = (id: string) => api.post(`/server/${id}/stop`);
export const restartServer = (id: string) => api.post(`/server/${id}/restart`);
export const getVersions = () => api.get<string[]>('/versions');
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
