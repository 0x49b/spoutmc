import { Server } from '../types';

/**
 * Docker container interface from backend API
 * Based on github.com/docker/docker/api/types/container.Summary
 * Enriched with additional runtime information
 */
export interface DockerContainer {
  Id: string;
  Names: string[];
  Image: string;
  ImageID: string;
  Command: string;
  Created: number;
  StartedAt?: string; // ISO 8601 timestamp when container was started (enriched field)
  Type?: string; // Server type: "proxy", "lobby", or "game" (enriched field)
  Ports: Array<{
    IP?: string;
    PrivatePort: number;
    PublicPort?: number;
    Type: string;
  }>;
  Labels: Record<string, string>;
  State: string;
  Status: string;
  HostConfig: {
    NetworkMode: string;
  };
  NetworkSettings: {
    Networks: Record<string, any>;
  };
  Mounts: Array<any>;
}

/**
 * Maps Docker container state to Server status
 */
function mapContainerStatus(state: string): 'online' | 'offline' | 'restarting' {
  const stateLower = state.toLowerCase();

  if (stateLower === 'running') {
    return 'online';
  } else if (stateLower === 'restarting') {
    return 'restarting';
  } else {
    return 'offline';
  }
}

/**
 * Extracts the container name (removes leading slash)
 */
function extractContainerName(names: string[]): string {
  if (names.length === 0) return 'Unknown';
  // Docker names start with /, remove it
  return names[0].replace(/^\//, '');
}

/**
 * Calculates uptime from container start timestamp
 * Uses StartedAt if available (accurate for restarts), falls back to Created
 */
function calculateUptime(created: number, startedAt?: string): string {
  let startTime: number;

  if (startedAt) {
    // Parse ISO 8601 timestamp from StartedAt (e.g., "2024-01-15T10:30:00.123456789Z")
    const startDate = new Date(startedAt);
    startTime = Math.floor(startDate.getTime() / 1000);
  } else {
    // Fall back to Created timestamp if StartedAt not available
    startTime = created;
  }

  const now = Math.floor(Date.now() / 1000);
  const uptimeSeconds = now - startTime;

  // Handle negative uptime (shouldn't happen, but just in case)
  if (uptimeSeconds < 0) {
    return '0s';
  }

  const days = Math.floor(uptimeSeconds / 86400);
  const hours = Math.floor((uptimeSeconds % 86400) / 3600);
  const minutes = Math.floor((uptimeSeconds % 3600) / 60);
  const seconds = Math.floor(uptimeSeconds % 60);

  // Build uptime string based on what units are present
  const parts: string[] = [];

  if (days > 0) {
    parts.push(`${days}d`);
  }
  if (hours > 0 || days > 0) {
    parts.push(`${hours}h`);
  }
  if (minutes > 0 || hours > 0 || days > 0) {
    parts.push(`${minutes}m`);
  }
  if (seconds > 0 || uptimeSeconds < 60) {
    parts.push(`${seconds}s`);
  }

  return parts.join(' ');
}

/**
 * Extracts the first public port from container ports
 */
function extractPort(ports: DockerContainer['Ports']): number {
  if (!ports || ports.length === 0) return 0;

  // Find first port with public mapping
  const publicPort = ports.find(p => p.PublicPort !== undefined);
  if (publicPort && publicPort.PublicPort) {
    return publicPort.PublicPort;
  }

  // Fallback to first private port
  return ports[0].PrivatePort || 0;
}

/**
 * Extracts Minecraft version from container image or labels
 */
function extractVersion(image: string, labels: Record<string, string>): string {
  // Check if version is in labels
  if (labels.VERSION) {
    return labels.VERSION;
  }

  // Try to extract from image name
  // Example: itzg/minecraft-server:1.19.3
  const versionMatch = image.match(/:(\d+\.\d+\.?\d*)/);
  if (versionMatch) {
    return versionMatch[1];
  }

  return 'latest';
}

/**
 * Transforms Docker container data to Server interface
 */
export function mapDockerContainerToServer(container: DockerContainer): Server {
  const name = extractContainerName(container.Names);
  const isProxy = container.Labels['io.spout.proxy'] === 'true';
  const isLobby = container.Labels['io.spout.lobby'] === 'true';

  // Extract server type from backend or determine from labels
  let serverType: 'proxy' | 'lobby' | 'game' = 'game';
  if (container.Type) {
    serverType = container.Type as 'proxy' | 'lobby' | 'game';
  } else if (isProxy) {
    serverType = 'proxy';
  } else if (isLobby) {
    serverType = 'lobby';
  }

  return {
    id: container.Id,
    name: name,
    type: serverType,
    status: mapContainerStatus(container.State),
    ip: 'localhost', // Default for local Docker
    port: extractPort(container.Ports),
    players: 0, // Will be updated via stats SSE
    maxPlayers: 100, // Default, could be extracted from env vars
    uptime: calculateUptime(container.Created, container.StartedAt),
    version: extractVersion(container.Image, container.Labels),
    cpu: 0, // Will be updated via stats SSE
    memory: 0, // Will be updated via stats SSE
    plugins: [], // Could be populated from volumes or API
    location: isProxy ? 'Proxy' : isLobby ? 'Lobby' : undefined,
    description: container.Status
  };
}

/**
 * Container with stats from SSE stream
 */
export interface ContainerWithStats {
  container: DockerContainer;
  stats?: any;
}

/**
 * Maps an array of Docker containers to Server array
 */
export function mapDockerContainersToServers(containers: DockerContainer[]): Server[] {
  return containers.map(mapDockerContainerToServer);
}

/**
 * Maps containers with stats from SSE stream to Server array
 */
export function mapContainersWithStatsToServers(containersWithStats: ContainerWithStats[]): Server[] {
  return containersWithStats.map(({ container, stats }) => {
    const server = mapDockerContainerToServer(container);

    // If stats are available, update CPU and memory
    if (stats) {
      return updateServerWithStats(server, stats);
    }

    return server;
  });
}

/**
 * Updates server stats from Docker stats API response
 * This merges real-time stats into an existing server object
 */
export function updateServerWithStats(server: Server, stats: any): Server {
  if (!stats) return server;

  // Extract CPU percentage
  let cpuPercent = 0;

  // Try both uppercase and lowercase field names (Docker API can vary)
  const cpuStats = stats.cpu_stats || stats.CPUStats;
  const preCpuStats = stats.precpu_stats || stats.PreCPUStats;

  if (cpuStats && preCpuStats) {
    const cpuUsage = cpuStats.cpu_usage || cpuStats.CPUUsage;
    const preCpuUsage = preCpuStats.cpu_usage || preCpuStats.CPUUsage;

    if (cpuUsage && preCpuUsage) {
      const cpuDelta = cpuUsage.total_usage - preCpuUsage.total_usage;
      const systemDelta = cpuStats.system_cpu_usage - preCpuStats.system_cpu_usage;
      const numberCpus = cpuStats.online_cpus || 1;

      if (systemDelta > 0 && cpuDelta > 0) {
        cpuPercent = (cpuDelta / systemDelta) * numberCpus * 100;
      }
    }
  }

  // Extract memory percentage
  let memoryPercent = 0;
  const memoryStats = stats.memory_stats || stats.MemoryStats;

  if (memoryStats) {
    const used = memoryStats.usage || 0;
    const limit = memoryStats.limit || 1;
    if (limit > 0) {
      memoryPercent = (used / limit) * 100;
    }
  }

  return {
    ...server,
    cpu: Math.round(cpuPercent * 10) / 10, // Round to 1 decimal
    memory: Math.round(memoryPercent * 10) / 10, // Round to 1 decimal
  };
}
