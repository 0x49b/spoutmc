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

/** Plugin registry entry from GET /api/v1/plugin (user-defined + system-managed). */
export interface RegistryPluginEntry {
  id: string;
  name: string;
  url: string;
  description?: string;
  systemManaged: boolean;
  serverNames: string[];
  kinds?: string[];
}

export interface BreadcrumbItem {
  label: string;
  path: string;
}

/** Role name from the API (built-in or custom). */
export type Role = string;

export interface UserProfile {
  id: string;
  email: string;
  roles: Role[];
  /** Effective permission keys (component.module.permission), union of roles + direct grants. */
  permissions: string[];
  /** Direct permission grants only (for editing). */
  directPermissions?: { id: number; key: string; description?: string }[];
  displayName: string;
  minecraftName?: string;
  /** Raw base64 image (PNG minime or legacy JPEG default) */
  avatar?: string;
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