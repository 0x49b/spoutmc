package docker

import (
	"fmt"
	"path/filepath"
	"spoutmc/internal/models"
	"spoutmc/internal/utils/path"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
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
	normalizedDataPath := pathutil.NormalizeHostPath(dataPath)
	normalizedContainerPath := pathutil.NormalizeContainerPath(containerPath)
	relativeContainerPath := strings.TrimPrefix(normalizedContainerPath, "/")

	// Convert slash separators before joining on the host OS.
	return filepath.Join(normalizedDataPath, containerName, filepath.FromSlash(relativeContainerPath))
}

func MapVolumeBindings(volumes []models.SpoutServerVolumes, dataPath, containerName string) []mount.Mount {
	var spoutVolumes []mount.Mount

	for _, v := range volumes {
		containerPath := pathutil.NormalizeContainerPath(v.Containerpath)
		hostPath := createHostPath(dataPath, containerName, containerPath)
		spoutVolumes = append(spoutVolumes, mount.Mount{
			Type:   mount.TypeBind,
			Source: hostPath,
			Target: containerPath,
		})
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
