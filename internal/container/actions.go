package container

import (
	"context"
	"spoutmc/internal/docker"
	"spoutmc/internal/global"
	"spoutmc/internal/log"

	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleContainer)

// StartContainer starts a container and includes it in watchdog monitoring
func StartContainer(ctx context.Context, containerID string) error {
	logger.Info("Starting container", zap.String("id", containerID[:12]))

	// Start the container
	docker.StartContainerById(ctx, containerID)

	// Include in watchdog monitoring
	if global.Watchdog != nil {
		global.Watchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", containerID[:12]))
	}

	return nil
}

// StopContainer stops a container and excludes it from watchdog monitoring
func StopContainer(ctx context.Context, containerID string) error {
	logger.Info("Stopping container", zap.String("id", containerID[:12]))

	// Exclude from watchdog monitoring BEFORE stopping
	// This prevents the watchdog from immediately restarting it
	if global.Watchdog != nil {
		global.Watchdog.Exclude(containerID)
		logger.Info("Container excluded from watchdog monitoring", zap.String("id", containerID[:12]))
	}

	// Stop the container
	docker.StopContainerById(ctx, containerID)

	return nil
}

// RestartContainer restarts a container (remains in watchdog monitoring)
func RestartContainer(ctx context.Context, containerID string) error {
	logger.Info("Restarting container", zap.String("id", containerID[:12]))

	// Restart the container
	docker.RestartContainerById(ctx, containerID)

	// Ensure it's included in watchdog monitoring after restart
	if global.Watchdog != nil {
		global.Watchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", containerID[:12]))
	}

	return nil
}
