package server

import (
	"fmt"
	"spoutmc/internal/config"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleServer)

// DetermineServerType determines the server type based on container labels
// Returns "proxy", "lobby", or "game"
func DetermineServerType(labels map[string]string) string {
	if labels["io.spout.proxy"] == "true" {
		return "proxy"
	}
	if labels["io.spout.lobby"] == "true" {
		return "lobby"
	}
	return "game"
}

// FindProxyServer finds the proxy server container
func FindProxyServer() (*container.Summary, error) {
	containers, err := docker.GetNetworkContainers()
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	for _, c := range containers {
		// Get detailed container info to check environment variables
		containerInfo, err := docker.GetContainerById(c.ID)
		if err != nil {
			continue
		}

		// Check if this is a proxy server
		for _, env := range containerInfo.Config.Env {
			if len(env) > 5 && env[:5] == "TYPE=" && env[5:] == "VELOCITY" {
				return &c, nil
			}
		}
	}

	return nil, fmt.Errorf("no proxy server found")
}

// GetDefaultEnvVars returns system-managed environment variables for each server type
// These defaults can be overridden by user-provided values
func GetDefaultEnvVars(isProxy, isLobby bool) map[string]string {
	if isProxy {
		// Proxy server defaults
		return map[string]string{
			"TYPE": "VELOCITY",
		}
	} else {
		// Get the Velocity forwarding secret automatically
		// Try to get data path and proxy name from config
		cfg := config.All()
		dataPath := ""
		proxyName := ""

		if cfg.Storage != nil {
			dataPath = cfg.Storage.DataPath
		}

		// Find proxy server name
		for i := range cfg.Servers {
			if cfg.Servers[i].Proxy {
				proxyName = cfg.Servers[i].Name
				break
			}
		}

		velocitySecret := docker.GetOrGenerateVelocitySecret(dataPath, proxyName)

		// Log secret preview for debugging (first 8 chars only for security)
		secretPreview := velocitySecret
		if len(velocitySecret) > 8 {
			secretPreview = velocitySecret[:8] + "..."
		}
		logger.Info("Configuring Paper server with Velocity forwarding",
			zap.String("secret_preview", secretPreview),
			zap.Int("secret_length", len(velocitySecret)))

		// Lobby and game server defaults with Velocity forwarding support
		return map[string]string{
			"EULA":                     "TRUE",
			"TYPE":                     "PAPER",
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

// MergeEnvVars merges default environment variables with user-provided ones
// User-provided values override defaults
func MergeEnvVars(defaults, userProvided map[string]string) map[string]string {
	merged := make(map[string]string)

	// Start with defaults
	for k, v := range defaults {
		merged[k] = v
	}

	// Override with user-provided values
	for k, v := range userProvided {
		merged[k] = v
	}

	return merged
}

// FindNextAvailablePort finds the next available port starting from 25566
// Returns the next available port number
func FindNextAvailablePort() int {
	existingConfig := config.All()
	usedPorts := make(map[int]bool)

	// Collect all ports currently in use
	for _, server := range existingConfig.Servers {
		for _, portMapping := range server.Ports {
			// Parse host port as integer
			if hostPort := portMapping.HostPort; hostPort != "" {
				var port int
				fmt.Sscanf(hostPort, "%d", &port)
				if port > 0 {
					usedPorts[port] = true
				}
			}
		}
	}

	// Start from 25566 (25565 is typically for proxy) and find first available
	startPort := 25566
	for port := startPort; port <= 65535; port++ {
		if !usedPorts[port] {
			return port
		}
	}

	// Fallback (should never happen unless all ports are used)
	return startPort
}

// ValidateProxyConstraint checks if adding a proxy server is allowed
// Returns error if a proxy already exists
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

// ValidateLobbyConstraint checks if adding a lobby server is allowed
// Returns error if a lobby already exists
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
