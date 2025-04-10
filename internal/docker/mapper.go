package docker

import (
	"fmt"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"spoutmc/internal/models"
)

func MapExposedPorts(ports []models.SpoutServerPorts) (nat.PortSet, nat.PortMap) {
	exposedPorts := nat.PortSet{}
	containerPortBinding := nat.PortMap{}

	for _, p := range ports {
		// Skip zero values
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

func createHostPath(hostpath []string) string {
	wd, err := os.Getwd()

	if err != nil {
		logger.Error("Could not get cwd", zap.Error(err))
		return ""
	}

	return filepath.Join(append([]string{wd}, hostpath...)...)
}

func MapVolumeBindings(volumes []models.SpoutServerVolumes) []string {
	var spoutVolumes []string

	for _, v := range volumes {
		logger.Info(fmt.Sprintf("Testing new path creation --> %s", createHostPath(v.Hostpath)))
		spoutVolumes = append(spoutVolumes, createHostPath(v.Hostpath)+":"+v.Containerpath)
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
