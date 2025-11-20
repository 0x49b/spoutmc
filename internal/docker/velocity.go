package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.uber.org/zap"
)

var velocityLogger = log.GetLogger()

// VelocityConfig represents the relevant parts of velocity.toml
type VelocityConfig struct {
	Servers     map[string]string   `toml:"servers"`
	Try         []string            `toml:"try"`
	ForcedHosts map[string][]string `toml:"forced-hosts"`
}

// SyncVelocityToml synchronizes the velocity.toml file with current server configuration
// This should be called on startup and whenever servers are added/removed
func SyncVelocityToml(cfg *models.SpoutConfiguration) error {
	velocityLogger.Info("🔄 Synchronizing velocity.toml with server configuration")

	if cfg == nil || len(cfg.Servers) == 0 {
		velocityLogger.Warn("No servers found in configuration, skipping velocity.toml sync")
		return nil
	}

	// Find proxy server
	var proxyServer *models.SpoutServer
	for i := range cfg.Servers {
		if cfg.Servers[i].Proxy {
			proxyServer = &cfg.Servers[i]
			break
		}
	}

	if proxyServer == nil {
		velocityLogger.Info("No proxy server found, skipping velocity.toml sync")
		return nil
	}

	// Get proxy server volume path
	velocityTomlPath, err := getVelocityTomlPath(proxyServer, cfg.Storage.DataPath)
	if err != nil {
		return fmt.Errorf("failed to get velocity.toml path: %w", err)
	}

	// Check if velocity.toml exists, if not, wait for it to be created
	if _, err := os.Stat(velocityTomlPath); os.IsNotExist(err) {
		velocityLogger.Info("velocity.toml not found yet, will be created by proxy server",
			zap.String("path", velocityTomlPath))
		return nil
	}

	// Read current velocity.toml
	data, err := os.ReadFile(velocityTomlPath)
	if err != nil {
		return fmt.Errorf("failed to read velocity.toml: %w", err)
	}

	// Parse TOML
	var velocityConfig map[string]interface{}
	if err := toml.Unmarshal(data, &velocityConfig); err != nil {
		return fmt.Errorf("failed to parse velocity.toml: %w", err)
	}

	// Build servers map, try list, and forced-hosts
	servers := make(map[string]string)
	var tryList []string
	forcedHosts := make(map[string][]string)
	var lobbyServer *models.SpoutServer

	// First pass: find lobby server and build server list
	for i := range cfg.Servers {
		server := &cfg.Servers[i]
		if server.Proxy {
			continue // Skip proxy server
		}

		if server.Lobby {
			lobbyServer = server
		}

		// Get port for this server
		port := getServerPort(server)
		if port != "" {
			// Server address format: containername:port (Docker network internal)
			servers[server.Name] = fmt.Sprintf("%s:%s", server.Name, port)
		}
	}

	// Build try list with lobby first
	if lobbyServer != nil {
		tryList = append(tryList, lobbyServer.Name)
	}

	// Add other servers to try list
	for i := range cfg.Servers {
		server := &cfg.Servers[i]
		if !server.Proxy && !server.Lobby {
			tryList = append(tryList, server.Name)
		}
	}

	// Build forced-hosts - map all servers to their names
	for serverName := range servers {
		forcedHosts[serverName] = []string{serverName}
	}

	// Update velocity config
	velocityConfig["servers"] = servers
	velocityConfig["try"] = tryList
	velocityConfig["forced-hosts"] = forcedHosts

	// Marshal back to TOML
	output, err := toml.Marshal(velocityConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal velocity.toml: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(velocityTomlPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write velocity.toml: %w", err)
	}

	velocityLogger.Info("✅ Successfully synchronized velocity.toml",
		zap.Int("servers", len(servers)),
		zap.Strings("try", tryList),
		zap.String("path", velocityTomlPath))

	return nil
}

// UpdateVelocityTomlAddServer adds a new server to velocity.toml
func UpdateVelocityTomlAddServer(cfg *models.SpoutConfiguration, serverName string, port int, isLobby bool) error {
	velocityLogger.Info("Adding server to velocity.toml",
		zap.String("server", serverName),
		zap.Int("port", port),
		zap.Bool("lobby", isLobby))

	// Just call the full sync - it's simpler and ensures consistency
	return SyncVelocityToml(cfg)
}

// UpdateVelocityTomlRemoveServer removes a server from velocity.toml
func UpdateVelocityTomlRemoveServer(cfg *models.SpoutConfiguration, serverName string) error {
	velocityLogger.Info("Removing server from velocity.toml",
		zap.String("server", serverName))

	// Just call the full sync - it's simpler and ensures consistency
	return SyncVelocityToml(cfg)
}

// getVelocityTomlPath returns the path to velocity.toml for the proxy server
func getVelocityTomlPath(proxyServer *models.SpoutServer, dataPath string) (string, error) {
	if len(proxyServer.Volumes) == 0 {
		return "", fmt.Errorf("proxy server has no volumes configured")
	}

	// Get the first volume's container path (usually /server for proxy)
	containerPath := proxyServer.Volumes[0].Containerpath

	// Build host path: {dataPath}/{serverName}/{containerPath}/velocity.toml
	// Remove leading slash from container path for joining
	cleanContainerPath := strings.TrimPrefix(containerPath, "/")
	velocityTomlPath := filepath.Join(dataPath, proxyServer.Name, cleanContainerPath, "velocity.toml")

	return velocityTomlPath, nil
}

// getServerPort extracts the port from server configuration
func getServerPort(server *models.SpoutServer) string {
	if len(server.Ports) > 0 {
		return server.Ports[0].ContainerPort
	}
	return ""
}
