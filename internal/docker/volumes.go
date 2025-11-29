package docker

import (
	"fmt"
	"os"
	"os/user"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strconv"

	"go.uber.org/zap"
)

var volumeLogger = log.GetLogger()

// getCurrentUser returns the UID and GID of the current user
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

// ensureVolumeDirectoriesExist creates volume directories with proper ownership
// This prevents Docker from creating them as root on Linux
func ensureVolumeDirectoriesExist(volumes []models.SpoutServerVolumes, dataPath, containerName string) error {
	if len(volumes) == 0 {
		return nil
	}

	uid, gid, err := getCurrentUser()
	if err != nil {
		volumeLogger.Warn("Failed to get current user, directories may have incorrect ownership",
			zap.Error(err))
		// Continue anyway - directories will be created by Docker
		return nil
	}

	for _, vol := range volumes {
		hostPath := createHostPath(dataPath, containerName, vol.Containerpath)

		// Create directory with 0755 permissions (rwxr-xr-x)
		if err := os.MkdirAll(hostPath, 0755); err != nil {
			return fmt.Errorf("failed to create volume directory %s: %w", hostPath, err)
		}

		// Set ownership to current user
		if err := os.Chown(hostPath, uid, gid); err != nil {
			volumeLogger.Warn("Failed to set ownership on volume directory",
				zap.String("path", hostPath),
				zap.Int("uid", uid),
				zap.Int("gid", gid),
				zap.Error(err))
			// Don't fail - directory exists, just with wrong ownership
		} else {
			volumeLogger.Debug("Created volume directory with proper ownership",
				zap.String("path", hostPath),
				zap.Int("uid", uid),
				zap.Int("gid", gid))
		}
	}

	return nil
}
