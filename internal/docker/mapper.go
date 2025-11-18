package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/models"

	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

func MapExposedPorts(ports []models.SpoutServerPorts) (nat.PortSet, nat.PortMap) {
	exposedPorts := nat.PortSet{}
	containerPortBinding := nat.PortMap{}

	for _, p := range ports {
		if p.HostPort == "" || p.ContainerPort == "" {
			continue
		}
		port := nat.Port(p.ContainerPort + "/tcp")
		exposedPorts[port] = struct{}{}
		hostBinding := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: p.HostPort,
		}
		containerPortBinding[port] = append(containerPortBinding[port], hostBinding)
	}
	return exposedPorts, containerPortBinding
}

// createHostPath generates the host path for a volume binding
// Format: {dataPath}/{containerName}/{containerPath}
// Uses filepath.Join to handle OS-specific path separators (/ for Unix, \ for Windows)
func createHostPath(dataPath, containerName, containerPath string) string {
	if dataPath == "" {
		// Fallback to working directory if no data path configured
		wd, err := os.Getwd()
		if err != nil {
			logger.Error("Could not get cwd", zap.Error(err))
			return ""
		}
		dataPath = wd
	}

	// Remove leading slash from containerPath to avoid double slashes
	// filepath.Join handles this, but we'll clean it explicitly for clarity
	cleanContainerPath := filepath.Clean(containerPath)
	if len(cleanContainerPath) > 0 && cleanContainerPath[0] == '/' {
		cleanContainerPath = cleanContainerPath[1:]
	}

	// Creates path: {dataPath}/{containerName}/{containerPath}
	return filepath.Join(dataPath, containerName, cleanContainerPath)
}

func MapVolumeBindings(volumes []models.SpoutServerVolumes, dataPath, containerName string) []string {
	var spoutVolumes []string

	for _, v := range volumes {
		hostPath := createHostPath(dataPath, containerName, v.Containerpath)
		spoutVolumes = append(spoutVolumes, hostPath+":"+v.Containerpath)
	}
	return spoutVolumes
}

func MapEnvironmentVariables(environment map[string]string) []string {
	var containerEnv []string

	for k, v := range environment {
		containerEnv = append(containerEnv, fmt.Sprintf("%s=%s", k, v))
	}
	return containerEnv
}
