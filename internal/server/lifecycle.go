package server

import (
	"context"
	"fmt"
	"spoutmc/internal/config"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleServer)

func DetermineServerType(labels map[string]string) string {
	if labels["io.spout.proxy"] == "true" {
		return "proxy"
	}
	if labels["io.spout.lobby"] == "true" {
		return "lobby"
	}
	return "game"
}

func FindProxyServer(ctx context.Context) (*container.Summary, error) {
	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	for _, c := range containers {
		containerInfo, err := docker.GetContainerById(ctx, c.ID)
		if err != nil {
			continue
		}

		for _, env := range containerInfo.Config.Env {
			if len(env) > 5 && env[:5] == "TYPE=" && env[5:] == "VELOCITY" {
				return &c, nil
			}
		}
	}

	return nil, fmt.Errorf("no proxy server found")
}

func GetDefaultEnvVars(isProxy, isLobby bool) map[string]string {
	if isProxy {
		return map[string]string{
			"TYPE": "VELOCITY",
		}
	} else {
		cfg := config.All()
		dataPath := ""
		proxyName := ""

		if cfg.Storage != nil {
			dataPath = cfg.Storage.DataPath
		}

		for i := range cfg.Servers {
			if cfg.Servers[i].Proxy {
				proxyName = cfg.Servers[i].Name
				break
			}
		}

		velocitySecret := docker.GetOrGenerateVelocitySecret(dataPath, proxyName)

		secretPreview := velocitySecret
		if len(velocitySecret) > 8 {
			secretPreview = velocitySecret[:8] + "..."
		}
		logger.Info("Configuring Paper server with Velocity forwarding",
			zap.String("secret_preview", secretPreview),
			zap.Int("secret_length", len(velocitySecret)))

		return map[string]string{
			"EULA":                     "TRUE",
			"TYPE":                     "PAPER",
			"ENABLE_RCON":              "false",
			"CREATE_CONSOLE_IN_PIPE":   "true",
			"ONLINE_MODE":              "FALSE",
			"GUI":                      "FALSE",
			"CONSOLE":                  "FALSE",
			"REPLACE_ENV_VARIABLES":    "TRUE",
			"ENV_VARIABLE_PREFIX":      "CFG_",
			"CFG_VELOCITY_ENABLED":     "true",
			"CFG_VELOCITY_ONLINE_MODE": "true",
			"CFG_VELOCITY_SECRET":      velocitySecret,
		}
	}
}

func MergeEnvVars(defaults, userProvided map[string]string) map[string]string {
	merged := make(map[string]string)

	for k, v := range defaults {
		merged[k] = v
	}

	for k, v := range userProvided {
		merged[k] = v
	}

	return merged
}

func FindNextAvailablePort() int {
	existingConfig := config.All()
	usedPorts := make(map[int]bool)

	for _, server := range existingConfig.Servers {
		for _, portMapping := range server.Ports {
			if hostPort := portMapping.HostPort; hostPort != "" {
				var port int
				fmt.Sscanf(hostPort, "%d", &port)
				if port > 0 {
					usedPorts[port] = true
				}
			}
		}
	}

	startPort := 25566
	for port := startPort; port <= 65535; port++ {
		if !usedPorts[port] {
			return port
		}
	}

	return startPort
}

func ValidateProxyConstraint() error {
	existingConfig := config.All()
	for _, server := range existingConfig.Servers {
		if server.Proxy {
			logger.Warn("Attempt to add duplicate proxy server",
				zap.String("existing", server.Name))
			return fmt.Errorf("a proxy server already exists (%s). Only one proxy is allowed in the network", server.Name)
		}
	}
	return nil
}

func ValidateLobbyConstraint() error {
	existingConfig := config.All()
	for _, server := range existingConfig.Servers {
		if server.Lobby {
			logger.Warn("Attempt to add duplicate lobby server",
				zap.String("existing", server.Name))
			return fmt.Errorf("a lobby server already exists (%s). Only one lobby is allowed in the network", server.Name)
		}
	}
	return nil
}
