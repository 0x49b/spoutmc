import { InfrastructureContainer } from '../types';

function toNum(val: unknown): number {
  const n = Number(val);
  return Number.isFinite(n) ? n : 0;
}

/**
 * Computes CPU and memory percentages from Docker stats and merges into container.
 * Used for infrastructure containers from SSE stream.
 */
export function updateContainerWithStats(
  container: InfrastructureContainer
): InfrastructureContainer & { cpu?: number; memory?: number } {
  const stats = container.stats;
  if (!stats) {
    return container;
  }

  let cpuPercent = 0;
  const cpuStats = stats.cpu_stats || stats.CPUStats;
  const preCpuStats = stats.precpu_stats || stats.PreCPUStats;

  if (cpuStats && preCpuStats) {
    const cpuUsage = cpuStats.cpu_usage || cpuStats.CPUUsage;
    const preCpuUsage = preCpuStats.cpu_usage || preCpuStats.CPUUsage;

    if (cpuUsage && preCpuUsage) {
      const cpuDelta = toNum(cpuUsage.total_usage) - toNum(preCpuUsage.total_usage);
      const systemDelta =
        toNum(cpuStats.system_cpu_usage) - toNum(preCpuStats.system_cpu_usage);
      const numberCpus = toNum(cpuStats.online_cpus) || 1;

      if (systemDelta > 0 && cpuDelta >= 0) {
        cpuPercent = (cpuDelta / systemDelta) * numberCpus * 100;
      }
    }
  }

  let memoryPercent = 0;
  const memoryStats = stats.memory_stats || stats.MemoryStats;

  if (memoryStats) {
    const used = toNum(memoryStats.usage);
    const limit = toNum(memoryStats.limit) || 1;
    if (limit > 0) {
      memoryPercent = (used / limit) * 100;
    }
  }

  return {
    ...container,
    cpu: Math.round(cpuPercent * 10) / 10,
    memory: Math.round(memoryPercent * 10) / 10,
  };
}
