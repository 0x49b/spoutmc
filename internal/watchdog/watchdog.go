package watchdog

import (
	"context"
	"fmt"
	"time"

	"spoutmc/internal/config"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/models"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

type Watchdog struct {
	cli          *client.Client
	logger       *log.ModuleLogger
	excluded     map[string]struct{}
	pollInterval time.Duration
}

func NewWatchdog(pollInterval time.Duration) (*Watchdog, error) {
	cli := docker.GetDockerClient()

	return &Watchdog{
		cli:          cli,
		logger:       log.GetLogger(log.ModuleWatchdog),
		excluded:     make(map[string]struct{}),
		pollInterval: pollInterval,
	}, nil
}

func (w *Watchdog) Exclude(containerID string) {
	w.excluded[containerID] = struct{}{}
	w.logger.Debug("excluded container", zap.String("containerID", containerID))
}

func (w *Watchdog) Include(containerID string) {
	delete(w.excluded, containerID)
	w.logger.Debug("included container", zap.String("containerID", containerID))
}

func (w *Watchdog) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.logger.Info("watchdog started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("watchdog shutting down")
			return
		case <-ticker.C:
			w.checkContainers(ctx)
		}
	}
}

func (w *Watchdog) checkContainers(ctx context.Context) {
	w.checkExistingContainers(ctx)

	w.checkInfrastructureContainers(ctx)

	w.checkMissingServers(ctx)

	w.ensurePaperVelocityConfig(ctx)
}

func (w *Watchdog) ensurePaperVelocityConfig(ctx context.Context) {
	cfg := config.All()
	if cfg.Storage == nil {
		return
	}

	dataPath := cfg.Storage.DataPath
	if dataPath == "" {
		return
	}

	proxyName := ""
	for i := range cfg.Servers {
		if cfg.Servers[i].Proxy {
			proxyName = cfg.Servers[i].Name
			break
		}
	}

	velocitySecret := docker.GetOrGenerateVelocitySecret(dataPath, proxyName)

	if err := docker.CheckAndConfigurePaperServers(dataPath, velocitySecret); err != nil {
		w.logger.Debug("error configuring Paper servers for Velocity", zap.Error(err))
	}
}

func (w *Watchdog) checkExistingContainers(ctx context.Context) {
	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		w.logger.Error("error getting network containers", zap.Error(err))
		return
	}

	for _, c := range containers {
		if _, excluded := w.excluded[c.ID]; excluded {
			continue
		}

		info, err := w.cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			w.logger.Error("error inspecting container", zap.String("id", c.ID), zap.Error(err))
			continue
		}

		status := info.State.Status
		healthStatus := "none"
		if info.State.Health != nil {
			healthStatus = info.State.Health.Status
		}

		w.logger.Debug("container status",
			zap.String("hostname", info.Config.Hostname),
			zap.String("status", status),
			zap.String("health", healthStatus),
		)

		if status == "exited" || status == "dead" {
			w.logger.Warn(fmt.Sprintf("%s container is stopped (reason: %s), restarting", info.Config.Hostname, status))
			w.startContainer(ctx, c.ID, info.Config.Hostname)
			continue
		}

		if healthStatus == "unhealthy" {
			w.logger.Warn(fmt.Sprintf("%s container is unhealthy (reason: %s), restarting", info.Config.Hostname, healthStatus))
			w.restartContainer(ctx, c.ID, info.Config.Hostname)
		}
	}
}

func (w *Watchdog) checkMissingServers(ctx context.Context) {
	cfg := config.All()
	if len(cfg.Servers) == 0 {
		return
	}

	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		w.logger.Error("error getting network containers", zap.Error(err))
		return
	}

	runningContainers := make(map[string]bool)
	for _, c := range containers {
		if len(c.Names) > 0 {
			name := c.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
			runningContainers[name] = true
		}
	}

	for _, server := range cfg.Servers {
		if !runningContainers[server.Name] {
			w.logger.Warn(fmt.Sprintf("server %s defined in config but not running, creating", server.Name))
			w.createMissingServer(ctx, server, dataPath)
		}
	}
}

func (w *Watchdog) createMissingServer(ctx context.Context, server models.SpoutServer, dataPath string) {
	w.logger.Info("creating missing server", zap.String("server", server.Name))

	if err := docker.StartContainer(ctx, server, dataPath); err != nil {
		w.logger.Error(fmt.Sprintf("failed to create missing server %s", server.Name),
			zap.Error(err))
	}
}

func (w *Watchdog) startContainer(ctx context.Context, containerID, containerName string) {
	w.logger.Info(fmt.Sprintf("starting container %s", containerName))
	err := w.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		w.logger.Error(fmt.Sprintf("failed to start container %s", containerName), zap.Error(err))
	} else {
		w.logger.Info(fmt.Sprintf("container started %s", containerName))
	}
}

func (w *Watchdog) restartContainer(ctx context.Context, containerID, containerName string) {
	w.logger.Info(fmt.Sprintf("restarting container %s", containerName))
	err := w.cli.ContainerRestart(ctx, containerID, container.StopOptions{})
	if err != nil {
		w.logger.Error(fmt.Sprintf("failed to restart container %s", containerName), zap.Error(err))
	} else {
		w.logger.Info("container restarted", zap.String("container", containerName))
	}
}

func (w *Watchdog) checkInfrastructureContainers(ctx context.Context) {
	containers, err := docker.GetInfrastructureContainers(ctx)
	if err != nil {
		w.logger.Error("error getting infrastructure containers", zap.Error(err))
		return
	}

	for _, c := range containers {
		if _, excluded := w.excluded[c.ID]; excluded {
			continue
		}

		info, err := w.cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			w.logger.Error("error inspecting infrastructure container", zap.String("id", c.ID), zap.Error(err))
			continue
		}

		status := info.State.Status
		healthStatus := "none"
		if info.State.Health != nil {
			healthStatus = info.State.Health.Status
		}

		w.logger.Debug("infrastructure container status",
			zap.String("hostname", info.Config.Hostname),
			zap.String("status", status),
			zap.String("health", healthStatus),
		)

		if status == "exited" || status == "dead" {
			w.logger.Warn("infrastructure container is stopped, restarting",
				zap.String("hostname", info.Config.Hostname),
				zap.String("status", status),
			)
			w.startContainer(ctx, c.ID, info.Config.Hostname)
			continue
		}

		if healthStatus == "unhealthy" {
			w.logger.Warn("infrastructure container is unhealthy, restarting",
				zap.String("hostname", info.Config.Hostname),
				zap.String("health", healthStatus),
			)
			w.restartContainer(ctx, c.ID, info.Config.Hostname)
		}
	}
}
