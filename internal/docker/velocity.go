package docker

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strings"
	"sync"

	"go.uber.org/zap"
)

var velocityLogger = log.GetLogger(log.ModuleDocker)

// Global Velocity secret management
var (
	velocitySecretMutex  sync.RWMutex
	cachedVelocitySecret string
)

// VelocityConfig represents the relevant parts of velocity.toml
type VelocityConfig struct {
	Servers     map[string]string   `toml:"servers"`
	Try         []string            `toml:"try"`
	ForcedHosts map[string][]string `toml:"forced-hosts"`
}

// CreateOrUpdateVelocityToml creates or updates velocity.toml with proper section ordering
// This ensures [servers] section comes before the try array
func CreateOrUpdateVelocityToml(cfg *models.SpoutConfiguration) error {
	velocityLogger.Info("📝 Creating/updating velocity.toml with server configuration")

	if cfg == nil || len(cfg.Servers) == 0 {
		velocityLogger.Warn("No servers found in configuration, skipping velocity.toml creation")
		return nil
	}

	if cfg.Storage == nil {
		velocityLogger.Warn("Storage configuration is nil, cannot create velocity.toml")
		return fmt.Errorf("storage configuration is required for velocity.toml creation")
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
		velocityLogger.Info("No proxy server found, skipping velocity.toml creation")
		return nil
	}

	// Get proxy server volume path
	velocityTomlPath, err := getVelocityTomlPath(proxyServer, cfg.Storage.DataPath)
	if err != nil {
		return fmt.Errorf("failed to get velocity.toml path: %w", err)
	}

	// Ensure the directory exists
	velocityDir := filepath.Dir(velocityTomlPath)
	if err := os.MkdirAll(velocityDir, 0755); err != nil {
		return fmt.Errorf("failed to create velocity directory: %w", err)
	}

	// Generate or load forwarding secret
	forwardingSecretPath := filepath.Join(velocityDir, "forwarding.secret")
	if err := ensureForwardingSecret(forwardingSecretPath); err != nil {
		velocityLogger.Warn("Failed to create forwarding secret", zap.Error(err))
		// Continue anyway - user can manually create it
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

		// Get internal container port for this server
		// Minecraft servers always listen on 25565 internally (container port)
		// regardless of the host port mapping
		port := getInternalContainerPort(server)
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
	/*for i := range cfg.Servers {
		server := &cfg.Servers[i]
		if !server.Proxy && !server.Lobby {
			tryList = append(tryList, server.Name)
		}
	}*/

	// Build forced-hosts - map all servers to their names
	for serverName := range servers {
		forcedHosts[serverName] = []string{serverName}
	}

	// Check if velocity.toml exists
	var existingContent string
	if data, err := os.ReadFile(velocityTomlPath); err == nil {
		// File exists, read it to preserve other settings
		existingContent = string(data)
		velocityLogger.Info("Updating existing velocity.toml", zap.String("path", velocityTomlPath))
	} else {
		// File doesn't exist yet - Velocity will create it on first run
		// We'll create a minimal config with just our servers
		velocityLogger.Info("Creating new velocity.toml", zap.String("path", velocityTomlPath))
	}

	// Build the updated velocity.toml content
	output := buildVelocityTomlWithServers(existingContent, servers, tryList, forcedHosts)

	// Write to file
	if err := os.WriteFile(velocityTomlPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write velocity.toml: %w", err)
	}

	velocityLogger.Info("✅ Successfully created/updated velocity.toml",
		zap.Int("servers", len(servers)),
		zap.Strings("try", tryList),
		zap.String("path", velocityTomlPath))

	return nil
}

// buildVelocityTomlWithServers updates or creates velocity.toml with server configuration
// It preserves ALL existing configuration and only updates critical settings and managed sections
func buildVelocityTomlWithServers(existingContent string, servers map[string]string, tryList []string, forcedHosts map[string][]string) string {
	var builder strings.Builder

	// Settings we need to update for Velocity/Minecraft 1.19+ compatibility
	criticalSettings := map[string]string{
		"bind":                        "\"0.0.0.0:25565\"",
		"online-mode":                 "true",
		"force-key-authentication":    "false",
		"player-info-forwarding-mode": "\"modern\"",
		"forwarding-secret-file":      "\"forwarding.secret\"",
	}

	if existingContent != "" {
		lines := strings.Split(existingContent, "\n")
		inServersSection := false
		inForcedHostsSection := false
		inOtherSection := false
		skipSection := false
		insideTryArray := false

		for i := 0; i < len(lines); i++ {
			line := lines[i]
			trimmedLine := strings.TrimSpace(line)

			// Detect section headers
			if strings.HasPrefix(trimmedLine, "[") && strings.Contains(trimmedLine, "]") {
				sectionName := strings.TrimSpace(strings.Trim(trimmedLine, "[]"))

				// Check which section we're in
				if sectionName == "servers" {
					inServersSection = true
					inForcedHostsSection = false
					inOtherSection = false
					skipSection = true
					continue // Skip [servers] - we'll add it back later
				} else if sectionName == "forced-hosts" {
					inServersSection = false
					inForcedHostsSection = true
					inOtherSection = false
					skipSection = true
					continue // Skip [forced-hosts] - we'll add it back later
				} else {
					// Other section like [advanced], [query], etc.
					inServersSection = false
					inForcedHostsSection = false
					inOtherSection = true
					skipSection = false
					builder.WriteString(line + "\n")
					continue
				}
			}

			// Skip content in managed sections
			if skipSection && (inServersSection || inForcedHostsSection) {
				continue
			}

			// Handle top-level "try" array
			if !inOtherSection && (strings.HasPrefix(trimmedLine, "try =") || strings.HasPrefix(trimmedLine, "try=")) {
				insideTryArray = true
				// Skip until we find the closing bracket
				for i < len(lines) {
					if strings.Contains(lines[i], "]") {
						insideTryArray = false
						break
					}
					i++
				}
				continue
			}

			// Skip if inside try array
			if insideTryArray {
				continue
			}

			// Update critical settings (only if not in a section)
			if !inOtherSection {
				updated := false
				for key, value := range criticalSettings {
					if strings.HasPrefix(trimmedLine, key+" =") || strings.HasPrefix(trimmedLine, key+"=") {
						// Preserve indentation
						indent := ""
						if len(line) > len(trimmedLine) {
							indent = line[:len(line)-len(trimmedLine)]
						}
						builder.WriteString(fmt.Sprintf("%s%s = %s\n", indent, key, value))
						updated = true
						break
					}
				}

				if updated {
					continue
				}
			}

			// Keep all other lines (preserves comments, settings, sections)
			builder.WriteString(line + "\n")
		}
	} else {
		// No existing content - create default config with all standard settings
		builder.WriteString("# Config version. Do not change this\n")
		builder.WriteString("config-version = \"2.7\"\n\n")
		builder.WriteString("# What port should the proxy be bound to?\n")
		builder.WriteString("bind = \"0.0.0.0:25565\"\n\n")
		builder.WriteString("# What should be the MOTD?\n")
		builder.WriteString("motd = \"<#09add3>A Velocity Server\"\n\n")
		builder.WriteString("# Maximum number of players\n")
		builder.WriteString("show-max-players = 500\n\n")
		builder.WriteString("# Should we authenticate players with Mojang?\n")
		builder.WriteString("online-mode = true\n\n")
		builder.WriteString("# Should the proxy enforce the new public key security standard?\n")
		builder.WriteString("force-key-authentication = false\n\n")
		builder.WriteString("# Player info forwarding mode\n")
		builder.WriteString("player-info-forwarding-mode = \"modern\"\n\n")
		builder.WriteString("# Forwarding secret file\n")
		builder.WriteString("forwarding-secret-file = \"forwarding.secret\"\n\n")
	}

	// Add SpoutMC managed sections at the end
	builder.WriteString("\n# ===== SpoutMC Managed Sections =====\n")
	builder.WriteString("# The following sections are automatically managed by SpoutMC\n")
	builder.WriteString("# Manual changes will be overwritten\n\n")

	// Write [servers] section
	builder.WriteString("[servers]\n")
	builder.WriteString("# Configure your servers here. Each key represents the server's name, and the value\n")
	builder.WriteString("# represents the IP address of the server to connect to.\n")
	for serverName, serverAddr := range servers {
		builder.WriteString(fmt.Sprintf("%s = \"%s\"\n", serverName, serverAddr))
	}

	// Write try array
	builder.WriteString("\n# In what order we should try servers when a player logs in or is kicked from a server.\n")
	builder.WriteString("try = [\n")
	for i, serverName := range tryList {
		if i == len(tryList)-1 {
			builder.WriteString(fmt.Sprintf("    \"%s\"\n", serverName))
		} else {
			builder.WriteString(fmt.Sprintf("    \"%s\",\n", serverName))
		}
	}
	builder.WriteString("]\n")

	// Write [forced-hosts] section
	if len(forcedHosts) > 0 {
		builder.WriteString("\n[forced-hosts]\n")
		builder.WriteString("# Configure your forced hosts here.\n")
		for host, targets := range forcedHosts {
			builder.WriteString(fmt.Sprintf("\"%s\" = [\n", host))
			for i, target := range targets {
				if i == len(targets)-1 {
					builder.WriteString(fmt.Sprintf("    \"%s\"\n", target))
				} else {
					builder.WriteString(fmt.Sprintf("    \"%s\",\n", target))
				}
			}
			builder.WriteString("]\n")
		}
	}

	return builder.String()
}

// SyncVelocityToml synchronizes the velocity.toml file with current server configuration
// This should be called on startup and whenever servers are added/removed
// This is now a wrapper around CreateOrUpdateVelocityToml for backwards compatibility
func SyncVelocityToml(cfg *models.SpoutConfiguration) error {
	return CreateOrUpdateVelocityToml(cfg)
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

// getInternalContainerPort returns the internal container port for velocity.toml
// This is always the container port (usually 25565 for game servers)
// NOT the host port, since Velocity communicates via Docker network
func getInternalContainerPort(server *models.SpoutServer) string {
	if len(server.Ports) > 0 {
		// Use the container port (internal), not host port
		return server.Ports[0].ContainerPort
	}
	// Default to 25565 if no port is configured
	return "25565"
}

// GetOrGenerateVelocitySecret returns the Velocity forwarding secret
// It checks (in order): environment variable, cached value, proxy's forwarding.secret file, or generates new
// Parameters: dataPath and proxyName are optional - if empty, will skip reading from file
func GetOrGenerateVelocitySecret(dataPath string, proxyName string) string {
	velocitySecretMutex.RLock()
	if cachedVelocitySecret != "" {
		secret := cachedVelocitySecret
		velocitySecretMutex.RUnlock()
		return secret
	}
	velocitySecretMutex.RUnlock()

	velocitySecretMutex.Lock()
	defer velocitySecretMutex.Unlock()

	// Double-check after acquiring write lock
	if cachedVelocitySecret != "" {
		return cachedVelocitySecret
	}

	// 1. Check environment variable
	envSecret := os.Getenv("VELOCITY_SECRET")
	if envSecret != "" {
		velocityLogger.Info("Using Velocity secret from VELOCITY_SECRET environment variable")
		cachedVelocitySecret = envSecret
		return cachedVelocitySecret
	}

	// 2. Try to read from proxy's forwarding.secret file if it exists
	if dataPath != "" && proxyName != "" {
		proxySecret, err := readProxyForwardingSecret(dataPath, proxyName)
		if err == nil && proxySecret != "" {
			velocityLogger.Info("Using existing Velocity secret from proxy's forwarding.secret file")
			cachedVelocitySecret = proxySecret
			return cachedVelocitySecret
		}
	}

	// 3. Generate a new random secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		velocityLogger.Error("Failed to generate random secret", zap.Error(err))
		// Fallback to a default (not secure, but prevents failure)
		return "changeme-please-set-velocity-secret"
	}

	cachedVelocitySecret = base64.StdEncoding.EncodeToString(secretBytes)
	velocityLogger.Info("Generated new Velocity forwarding secret",
		zap.String("secret", cachedVelocitySecret))
	velocityLogger.Warn("⚠️  IMPORTANT: This secret will be used for all backend servers")
	velocityLogger.Warn("⚠️  Optionally set VELOCITY_SECRET environment variable to use a custom secret")

	return cachedVelocitySecret
}

// readProxyForwardingSecret attempts to read the forwarding secret from the proxy server's volume
func readProxyForwardingSecret(dataPath string, proxyName string) (string, error) {
	// Build path to forwarding.secret
	secretPath := filepath.Join(dataPath, proxyName, "server", "forwarding.secret")

	// Try to read the secret
	if data, err := os.ReadFile(secretPath); err == nil {
		secret := strings.TrimSpace(string(data))
		if secret != "" {
			return secret, nil
		}
	}

	return "", fmt.Errorf("forwarding.secret file not found at %s", secretPath)
}

// ensureForwardingSecret creates a forwarding secret file if it doesn't exist
// This secret is used by Velocity for modern player info forwarding
func ensureForwardingSecret(secretPath string) error {
	// Check if secret already exists
	if _, err := os.Stat(secretPath); err == nil {
		// Read existing secret and cache it
		if data, err := os.ReadFile(secretPath); err == nil {
			secret := strings.TrimSpace(string(data))
			velocitySecretMutex.Lock()
			cachedVelocitySecret = secret
			velocitySecretMutex.Unlock()
			velocityLogger.Info("Using existing forwarding secret", zap.String("path", secretPath))
			return nil
		}
	}

	// Get or generate the secret (will use cached value if available)
	// We don't pass dataPath/proxyName here because we're in the process of creating the secret
	secret := GetOrGenerateVelocitySecret("", "")

	// Write secret to file
	if err := os.WriteFile(secretPath, []byte(secret), 0600); err != nil {
		return fmt.Errorf("failed to write forwarding secret: %w", err)
	}

	velocityLogger.Info("✅ Forwarding secret created successfully",
		zap.String("path", secretPath),
		zap.Int("length", len(secret)))

	return nil
}
