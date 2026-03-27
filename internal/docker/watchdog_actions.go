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

func SetWatchdogActions(w watchdogActions) {
	dockerWatchdog = w
}

func shortContainerID(containerID string) string {
	if len(containerID) <= 12 {
		return containerID
	}
	return containerID[:12]
}

func StartContainerWithWatchdog(ctx context.Context, containerID string) error {
	logger.Info("Starting container", zap.String("id", shortContainerID(containerID)))

	StartContainerById(ctx, containerID)

	if dockerWatchdog != nil {
		dockerWatchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", shortContainerID(containerID)))
	}

	return nil
}

func StopContainerWithWatchdog(ctx context.Context, containerID string) error {
	logger.Info("Stopping container", zap.String("id", shortContainerID(containerID)))

	if dockerWatchdog != nil {
		dockerWatchdog.Exclude(containerID)
		logger.Info("Container excluded from watchdog monitoring", zap.String("id", shortContainerID(containerID)))
	}

	StopContainerById(ctx, containerID)
	return nil
}

func RestartContainerWithWatchdog(ctx context.Context, containerID string) error {
	logger.Info("Restarting container", zap.String("id", shortContainerID(containerID)))

	RestartContainerById(ctx, containerID)

	if dockerWatchdog != nil {
		dockerWatchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", shortContainerID(containerID)))
	}

	return nil
}
