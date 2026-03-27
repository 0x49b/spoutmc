package host

import (
	"context"
	"net/http"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleHost)

type Event struct {
	ID        []byte
	Data      []byte
	Event     []byte
	Retry     []byte
	Comment   []byte
	Timestamp int64
}

type OSInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`               // "linux", "windows", "darwin", ...
	Platform        string `json:"platform"`         // e.g. "ubuntu", "windows"
	PlatformFamily  string `json:"platform_family"`  // e.g. "debian", "fedora", "windows"
	PlatformVersion string `json:"platform_version"` // e.g. "22.04", "10.0.22631"
	KernelVersion   string `json:"kernel_version"`   // Linux kernel; Windows NT version; macOS kernel
	KernelArch      string `json:"kernel_arch"`      // e.g. "x86_64", "arm64"
}

type DockerInfo struct {
	ServerVersion       string   `json:"server_version"`
	Containers          int      `json:"containers_total"`
	ContainersRunning   int      `json:"containers_running"`
	ContainersPaused    int      `json:"containers_paused"`
	ContainersStopped   int      `json:"containers_stopped"`
	Images              int      `json:"images"`
	Driver              string   `json:"storage_driver"`
	CgroupDriver        string   `json:"cgroup_driver"`
	PluginsVolume       []string `json:"volume_plugins"`
	PluginsNetwork      []string `json:"network_plugins"`
	OperatingSystem     string   `json:"docker_os"`
	OSType              string   `json:"os_type"`
	Architecture        string   `json:"arch"`
	NCPU                int      `json:"docker_cpus"`
	MemTotalBytes       int64    `json:"docker_mem_total_bytes"`
	SwarmClusterEnabled bool     `json:"swarm_cluster_enabled"`
}

type DiskStat struct {
	Mount       string  `json:"mount"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type Stats struct {
	Timestamp time.Time `json:"timestamp"`

	OS OSInfo `json:"os_info"`

	CPUPercent     float64 `json:"cpu_percent"`
	CPUsLogical    int     `json:"cpus_logical"`
	CPUsPhysical   int     `json:"cpus_physical"`
	MemTotalBytes  uint64  `json:"mem_total_bytes"`
	MemUsedBytes   uint64  `json:"mem_used_bytes"`
	MemUsedPercent float64 `json:"mem_used_percent"`

	Load1     float64 `json:"load1"`
	Load5     float64 `json:"load5"`
	Load15    float64 `json:"load15"`
	UptimeSec uint64  `json:"uptime_seconds"`

	Disks []DiskStat `json:"disks"`

	Docker system.Info `json:"docker,omitempty"`
}

type Collector struct {
	mu       sync.RWMutex
	current  Stats
	thCPU    float64
	thMem    float64
	thLoad   float64
	thDisk   float64
	subsMu   sync.RWMutex
	nextID   atomic.Int64
	subs     map[int64]chan Stats
	interval time.Duration
}

func RegisterHostRoutes(g *echo.Group) {
	g.GET("/host/stats", getHostStats)
}

func getHostStats(c echo.Context) error {
	s := collectOnce(c.Request().Context())
	return c.JSON(http.StatusOK, s)
}

func collectOnce(ctx context.Context) Stats {
	cpuPct := 0.0
	if pct, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false); err == nil && len(pct) > 0 {
		cpuPct = pct[0]
	}

	logical, _ := cpu.CountsWithContext(ctx, true)
	physical, _ := cpu.CountsWithContext(ctx, false)

	vmTotal, vmUsed, vmUsedPct := uint64(0), uint64(0), 0.0
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		vmTotal, vmUsed, vmUsedPct = vm.Total, vm.Used, vm.UsedPercent
	}

	load1, load5, load15 := 0.0, 0.0, 0.0
	if avg, err := load.AvgWithContext(ctx); err == nil {
		load1, load5, load15 = avg.Load1, avg.Load5, avg.Load15
	}

	var osinfo OSInfo
	uptime := uint64(0)
	if hi, err := host.InfoWithContext(ctx); err == nil {
		uptime = hi.Uptime
		osinfo = OSInfo{
			Hostname:        hi.Hostname,
			OS:              hi.OS,
			Platform:        hi.Platform,
			PlatformFamily:  hi.PlatformFamily,
			PlatformVersion: hi.PlatformVersion,
			KernelVersion:   hi.KernelVersion,
			KernelArch:      hi.KernelArch,
		}
	}

	var disks []DiskStat
	if parts, err := disk.PartitionsWithContext(ctx, true); err == nil {
		for _, p := range parts {
			switch p.Fstype {
			case "proc", "sysfs", "devtmpfs", "devfs", "overlay", "tmpfs", "squashfs":
				continue
			}
			if u, err := disk.UsageWithContext(ctx, p.Mountpoint); err == nil {
				disks = append(disks, DiskStat{
					Mount:       p.Mountpoint,
					TotalBytes:  u.Total,
					UsedBytes:   u.Used,
					UsedPercent: u.UsedPercent,
				})
			}
		}
	}

	client := docker.GetDockerClient()
	info, err := client.Info(ctx)

	if err != nil {
		logger.Error("Error getting docker info", zap.Error(err))
	}

	return Stats{
		Timestamp:      time.Now().UTC(),
		OS:             osinfo,
		CPUPercent:     cpuPct,
		CPUsLogical:    logical,
		CPUsPhysical:   physical,
		MemTotalBytes:  vmTotal,
		MemUsedBytes:   vmUsed,
		MemUsedPercent: vmUsedPct,
		Load1:          load1,
		Load5:          load5,
		Load15:         load15,
		UptimeSec:      uptime,
		Disks:          disks,
		Docker:         info,
	}
}
