package docker

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strconv"

	"go.uber.org/zap"
)

var volumeLogger = log.GetLogger(log.ModuleDocker)

func getCurrentUser() (int, int, error) {
	currentUser, err := user.Current()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get current user: %w", err)
	}

	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse UID: %w", err)
	}

	gid, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse GID: %w", err)
	}

	return uid, gid, nil
}

func ensureVolumeDirectoriesExist(volumes []models.SpoutServerVolumes, dataPath, containerName string) error {
	if len(volumes) == 0 {
		return nil
	}

	if runtime.GOOS == "windows" {
		for _, vol := range volumes {
			hostPath := createHostPath(dataPath, containerName, vol.Containerpath)
			if err := os.MkdirAll(hostPath, 0755); err != nil {
				return fmt.Errorf("failed to create volume directory %s: %w", hostPath, err)
			}
			volumeLogger.Debug("Created volume directory (Windows)",
				zap.String("path", hostPath))
		}
		return nil
	}

	uid, gid, err := getCurrentUser()
	if err != nil {
		volumeLogger.Warn("Failed to get current user, directories may have incorrect ownership",
			zap.Error(err))
		for _, vol := range volumes {
			hostPath := createHostPath(dataPath, containerName, vol.Containerpath)
			if mkErr := os.MkdirAll(hostPath, 0755); mkErr != nil {
				return fmt.Errorf("failed to create volume directory %s: %w", hostPath, mkErr)
			}
		}
		return nil
	}

	for _, vol := range volumes {
		hostPath := createHostPath(dataPath, containerName, vol.Containerpath)

		if err := os.MkdirAll(hostPath, 0755); err != nil {
			return fmt.Errorf("failed to create volume directory %s: %w", hostPath, err)
		}

		if err := os.Chown(hostPath, uid, gid); err != nil {
			volumeLogger.Warn("Failed to set ownership on volume directory",
				zap.String("path", hostPath),
				zap.Int("uid", uid),
				zap.Int("gid", gid),
				zap.Error(err))
		} else {
			volumeLogger.Debug("Created volume directory with proper ownership",
				zap.String("path", hostPath),
				zap.Int("uid", uid),
				zap.Int("gid", gid))
		}
	}

	return nil
}
