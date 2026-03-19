export interface Player {
  id: string;
  username: string;
  avatarDataUrl?: string;
  currentServer?: string;
  lastLoggedInAt?: string;
  lastLoggedOutAt?: string;
  status: 'online' | 'offline' | 'banned';
  banned: boolean;
  banReason?: string;
}

export interface Server {
  id: string;
  name: string;
  type: 'proxy' | 'lobby' | 'game'; // Server type from backend
  status: 'online' | 'offline' | 'restarting';
  ip: string;
  port: number;
  players: number;
  maxPlayers: number;
  uptime: string;
  version: string;
  cpu: number;
  memory: number;
  plugins: string[];
  location?: string;
  description?: string;
}

export interface Plugin {
  id: string;
  name: string;
  version: string;
  author: string;
  status: 'enabled' | 'disabled';
  description: string;
  dependencies: string[];
  downloadUrl?: string;
  installedAt?: string;
}

export interface BreadcrumbItem {
  label: string;
  path: string;
}

export type Role = 'admin' | 'manager' | 'editor' | 'mod' | 'support';

export interface Permission {
  action: string;
  subject: string;
}

export interface RoleDTO {
  id: number;
  name: string;
}

export interface UserProfile {
  id: string;
  email: string;
  roles: Role[];
  displayName: string;
  minecraftName?: string;
  created_at: string;
  lastLoginAt?: string;
  aud: string;
  app_metadata: Record<string, any>;
  user_metadata: Record<string, any>;
  identities: any[];
}

export interface InfrastructureContainer {
  summary: {
    /** Docker API returns "Id" (capital I, lowercase d) */
    Id?: string;
    /** Fallback for alternate serialization */
    ID?: string;
    Names: string[];
    Image: string;
    State: string;
    Status: string;
    Ports: any[];
    Labels: Record<string, string>;
  };
  type: string;
  stats?: any; // Docker stats from SSE stream
  cpu?: number; // Computed from stats in store
  memory?: number; // Computed from stats in store
}

/** Get container ID from summary (handles both Id and ID from Docker API) */
export function getContainerId(summary: InfrastructureContainer['summary']): string {
  return summary.Id ?? summary.ID ?? '';
}