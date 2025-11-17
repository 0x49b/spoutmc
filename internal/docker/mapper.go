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

/*func createHostPath(hostpath []string) string {
	wd, err := os.Getwd()

	if err != nil {
		logger.Error("Could not get cwd", zap.Error(err))
		return ""
	}

	return filepath.Join(append([]string{wd}, hostpath...)...)
}*/

func createHostPath(containerName string) string {
	wd, err := os.Getwd()
	if err != nil {
		logger.Error("Could not get cwd", zap.Error(err))
		return ""
	}
	// Creates path: {workingDir}/{containerName}
	return filepath.Join(wd, containerName)
}

func MapVolumeBindings(volumes []models.SpoutServerVolumes, containerName string) []string {
	var spoutVolumes []string

	for _, v := range volumes {
		spoutVolumes = append(spoutVolumes, createHostPath(containerName)+":"+v.Containerpath)
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
