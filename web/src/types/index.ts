export interface Player {
  id: string;
  username: string;
  serverId?: string;
  lastSeen: Date;
  status: 'online' | 'offline' | 'banned';
  permanentBanned?: boolean;
  banReason?: string;
  bannedAt?: Date;
  bannedUntil?: Date;
}

export interface Server {
  id: string;
  name: string;
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

export type Role = 'admin' | 'moderator' | 'viewer';

export interface Permission {
  action: string;
  subject: string;
}

export interface UserProfile {
  id: string;
  email: string;
  roles: Role[];
  displayName: string;
  created_at: string;
  lastLoginAt?: string;
  aud: string;
  app_metadata: Record<string, any>;
  user_metadata: Record<string, any>;
  identities: any[];
}