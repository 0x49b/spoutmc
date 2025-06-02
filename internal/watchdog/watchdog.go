package watchdog

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
)

type Watchdog struct {
	cli          *client.Client
	logger       *zap.Logger
	excluded     map[string]struct{}
	pollInterval time.Duration
}

func NewWatchdog(pollInterval time.Duration) (*Watchdog, error) {
	cli, err := docker.GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker client: %w", err)
	}

	return &Watchdog{
		cli:          cli,
		logger:       log.GetLogger(),
		excluded:     make(map[string]struct{}),
		pollInterval: pollInterval,
	}, nil
}

func (w *Watchdog) Exclude(containerID string) {
	w.excluded[containerID] = struct{}{}
	w.logger.Debug("🐺 excluded container", zap.String("containerID", containerID))
}

func (w *Watchdog) Include(containerID string) {
	delete(w.excluded, containerID)
	w.logger.Debug("🐺 included container", zap.String("containerID", containerID))
}

func (w *Watchdog) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.logger.Info("🐺 watchdog started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("🐺 watchdog shutting down")
			return
		case <-ticker.C:
			w.checkContainers(ctx)
		}
	}
}

func (w *Watchdog) checkContainers(ctx context.Context) {
	containers, err := docker.GetNetworkContainers()
	if err != nil {
		w.logger.Error("🐺 error getting network containers", zap.Error(err))
		return
	}

	for _, c := range containers {
		if _, excluded := w.excluded[c.ID]; excluded {
			continue
		}

		info, err := w.cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			w.logger.Error("🐺 error inspecting container", zap.String("id", c.ID), zap.Error(err))
			continue
		}

		status := info.State.Status
		w.logger.Debug("🐺 container status",
			zap.String("hostname", info.Config.Hostname),
			zap.String("status", status),
		)

		if status == "exited" || status == "dead" {
			w.logger.Warn("🐺 restarting container",
				zap.String("hostname", info.Config.Hostname),
				zap.String("status", status),
			)
			w.startContainer(ctx, c.ID, info.Config.Hostname)
		}
	}
}

func (w *Watchdog) startContainer(ctx context.Context, containerID, containerName string) {
	w.logger.Info("🐺 starting container", zap.String("container", containerName))
	err := w.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		w.logger.Error("🐺 failed to start container", zap.String("container", containerName), zap.Error(err))
	} else {
		w.logger.Info("🐺 container started", zap.String("container", containerName))
	}
}
