package docker

import (
	"fmt"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"spoutmc/backend/models"
)

func MapExposedPorts(p models.SpoutServerPorts) (nat.PortSet, nat.PortMap) {
	var exposedPorts nat.PortSet
	var hostBinding nat.PortBinding
	var containerPortBinding nat.PortMap

	if (models.SpoutServerPorts{}) != p {

		exposedPorts = map[nat.Port]struct{}{
			nat.Port(p.ContainerPort + "/tcp"): {},
		}
		hostBinding = nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: p.HostPort,
		}
		containerPortBinding = nat.PortMap{
			nat.Port(p.ContainerPort + "/tcp"): []nat.PortBinding{hostBinding},
		}
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

func MapEnvironmentVariables(s models.SpoutServerEnv) []string {
	var containerEnv []string

	if s.Eula != "" {
		containerEnv = append(containerEnv, "EULA="+s.Eula)
	}
	if s.Type != "" {
		containerEnv = append(containerEnv, "TYPE="+s.Type)
	}
	if s.OnlineMode != "" {
		containerEnv = append(containerEnv, "ONLINE_MODE="+s.OnlineMode)
	}
	if s.EnforceSecureProfile != "" {
		containerEnv = append(containerEnv, "ENFORCE_SECURE_PROFILE="+s.EnforceSecureProfile)
	}
	if s.MaxMemory != "" {
		containerEnv = append(containerEnv, "MAX_MEMORY="+s.MaxMemory)
	}
	if s.Gui != "" {
		containerEnv = append(containerEnv, "GUI="+s.Gui)
	}
	if s.Console != "" {
		containerEnv = append(containerEnv, "CONSOLE="+s.Console)
	}
	if s.LogTimestamp != "" {
		containerEnv = append(containerEnv, "LOG_TIMESTAMP="+s.LogTimestamp)
	}
	if s.Tz != "" {
		containerEnv = append(containerEnv, "TZ="+s.Tz)
	}

	return containerEnv
}
