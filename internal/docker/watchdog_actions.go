package docker

import (
	"context"

	"go.uber.org/zap"
)

type watchdogActions interface {
	Include(containerID string)
	Exclude(containerID string)
}

var dockerWatchdog watchdogActions

// SetWatchdogActions configures optional watchdog include/exclude integration.
func SetWatchdogActions(w watchdogActions) {
	dockerWatchdog = w
}

func shortContainerID(containerID string) string {
	if len(containerID) <= 12 {
		return containerID
	}
	return containerID[:12]
}

// StartContainerWithWatchdog starts a container and includes it in watchdog monitoring.
func StartContainerWithWatchdog(ctx context.Context, containerID string) error {
	logger.Info("Starting container", zap.String("id", shortContainerID(containerID)))

	StartContainerById(ctx, containerID)

	if dockerWatchdog != nil {
		dockerWatchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", shortContainerID(containerID)))
	}

	return nil
}

// StopContainerWithWatchdog stops a container and excludes it from watchdog monitoring.
func StopContainerWithWatchdog(ctx context.Context, containerID string) error {
	logger.Info("Stopping container", zap.String("id", shortContainerID(containerID)))

	// Exclude before stopping to prevent immediate watchdog restart.
	if dockerWatchdog != nil {
		dockerWatchdog.Exclude(containerID)
		logger.Info("Container excluded from watchdog monitoring", zap.String("id", shortContainerID(containerID)))
	}

	StopContainerById(ctx, containerID)
	return nil
}

// RestartContainerWithWatchdog restarts a container and keeps it included in watchdog monitoring.
func RestartContainerWithWatchdog(ctx context.Context, containerID string) error {
	logger.Info("Restarting container", zap.String("id", shortContainerID(containerID)))

	RestartContainerById(ctx, containerID)

	if dockerWatchdog != nil {
		dockerWatchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", shortContainerID(containerID)))
	}

	return nil
}
