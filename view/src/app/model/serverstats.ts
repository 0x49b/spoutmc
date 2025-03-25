export interface ServerStats {
  name: string
  id: string
  read: string
  preread: string
  pids_stats: PidsStats
  blkio_stats: BlkioStats
  num_procs: number
  storage_stats: StorageStats
  cpu_stats: CpuStats
  precpu_stats: PrecpuStats
  memory_stats: MemoryStats
  networks: Networks
}

export interface PidsStats {
  current: number
  limit: number
}

export interface BlkioStats {
  io_service_bytes_recursive: IoServiceBytesRecursive[]
  io_serviced_recursive: any
  io_queue_recursive: any
  io_service_time_recursive: any
  io_wait_time_recursive: any
  io_merged_recursive: any
  io_time_recursive: any
  sectors_recursive: any
}

export interface IoServiceBytesRecursive {
  major: number
  minor: number
  op: string
  value: number
}

export interface StorageStats {
}

export interface CpuStats {
  cpu_usage: CpuUsage
  system_cpu_usage: number
  online_cpus: number
  throttling_data: ThrottlingData
}

export interface CpuUsage {
  total_usage: number
  usage_in_kernelmode: number
  usage_in_usermode: number
}

export interface ThrottlingData {
  periods: number
  throttled_periods: number
  throttled_time: number
}

export interface PrecpuStats {
  cpu_usage: CpuUsage2
  system_cpu_usage: number
  online_cpus: number
  throttling_data: ThrottlingData2
}

export interface CpuUsage2 {
  total_usage: number
  usage_in_kernelmode: number
  usage_in_usermode: number
}

export interface ThrottlingData2 {
  periods: number
  throttled_periods: number
  throttled_time: number
}

export interface MemoryStats {
  usage: number
  stats: Stats
  limit: number
}

export interface Stats {
  active_anon: number
  active_file: number
  anon: number
  anon_thp: number
  file: number
  file_dirty: number
  file_mapped: number
  file_writeback: number
  inactive_anon: number
  inactive_file: number
  kernel_stack: number
  pgactivate: number
  pgdeactivate: number
  pgfault: number
  pglazyfree: number
  pglazyfreed: number
  pgmajfault: number
  pgrefill: number
  pgscan: number
  pgsteal: number
  shmem: number
  slab: number
  slab_reclaimable: number
  slab_unreclaimable: number
  sock: number
  thp_collapse_alloc: number
  thp_fault_alloc: number
  unevictable: number
  workingset_activate: number
  workingset_nodereclaim: number
  workingset_refault: number
}

export interface Networks {
  eth0: Eth0
}

export interface Eth0 {
  rx_bytes: number
  rx_packets: number
  rx_errors: number
  rx_dropped: number
  tx_bytes: number
  tx_packets: number
  tx_errors: number
  tx_dropped: number
}
