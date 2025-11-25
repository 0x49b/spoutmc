package container

import (
	"spoutmc/internal/docker"
	"spoutmc/internal/global"
	"spoutmc/internal/log"

	"go.uber.org/zap"
)

var logger = log.GetLogger()

// StartContainer starts a container and includes it in watchdog monitoring
func StartContainer(containerID string) error {
	logger.Info("Starting container", zap.String("id", containerID[:12]))

	// Start the container
	docker.StartContainerById(containerID)

	// Include in watchdog monitoring
	if global.Watchdog != nil {
		global.Watchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", containerID[:12]))
	}

	return nil
}

// StopContainer stops a container and excludes it from watchdog monitoring
func StopContainer(containerID string) error {
	logger.Info("Stopping container", zap.String("id", containerID[:12]))

	// Exclude from watchdog monitoring BEFORE stopping
	// This prevents the watchdog from immediately restarting it
	if global.Watchdog != nil {
		global.Watchdog.Exclude(containerID)
		logger.Info("Container excluded from watchdog monitoring", zap.String("id", containerID[:12]))
	}

	// Stop the container
	docker.StopContainerById(containerID)

	return nil
}

// RestartContainer restarts a container (remains in watchdog monitoring)
func RestartContainer(containerID string) error {
	logger.Info("Restarting container", zap.String("id", containerID[:12]))

	// Restart the container
	docker.RestartContainerById(containerID)

	// Ensure it's included in watchdog monitoring after restart
	if global.Watchdog != nil {
		global.Watchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", containerID[:12]))
	}

	return nil
}
