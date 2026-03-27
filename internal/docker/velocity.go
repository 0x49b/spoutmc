package docker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/models"
	"strings"
	"sync"

	"go.uber.org/zap"
)

var (
	velocitySecretMutex  sync.RWMutex
	cachedVelocitySecret string
)

type VelocityConfig struct {
	Servers     map[string]string   `toml:"servers"`
	Try         []string            `toml:"try"`
	ForcedHosts map[string][]string `toml:"forced-hosts"`
}

func CreateOrUpdateVelocityToml(cfg *models.SpoutConfiguration) error {
	logger.Info("Creating/updating velocity.toml with server configuration")

	if cfg == nil || len(cfg.Servers) == 0 {
		logger.Warn("No servers found in configuration, skipping velocity.toml creation")
		return nil
	}

	if cfg.Storage == nil {
		logger.Error("Storage configuration is nil, cannot create velocity.toml")
		return fmt.Errorf("storage configuration is required for velocity.toml creation")
	}

	var proxyServer *models.SpoutServer
	for i := range cfg.Servers {
		if cfg.Servers[i].Proxy {
			proxyServer = &cfg.Servers[i]
			break
		}
	}

	if proxyServer == nil {
		logger.Info("No proxy server found, skipping velocity.toml creation")
		return nil
	}

	velocityTomlPath, err := getVelocityTomlPath(proxyServer, cfg.Storage.DataPath)
	if err != nil {
		return fmt.Errorf("failed to get velocity.toml path: %w", err)
	}

	velocityDir := filepath.Dir(velocityTomlPath)
	if err := os.MkdirAll(velocityDir, 0755); err != nil {
		return fmt.Errorf("failed to create velocity directory: %w", err)
	}

	forwardingSecretPath := filepath.Join(velocityDir, "forwarding.secret")
	if err := ensureForwardingSecret(forwardingSecretPath); err != nil {
		logger.Warn("Failed to create forwarding secret", zap.Error(err))
	}

	servers := make(map[string]string)
	var tryList []string
	forcedHosts := make(map[string][]string)
	var lobbyServer *models.SpoutServer

	for i := range cfg.Servers {
		server := &cfg.Servers[i]
		if server.Proxy {
			continue
		}

		if server.Lobby {
			lobbyServer = server
		}

		port := getInternalContainerPort(server)
		if port != "" {
			servers[server.Name] = fmt.Sprintf("%s:%s", server.Name, port)
		}
	}

	if lobbyServer != nil {
		tryList = append(tryList, lobbyServer.Name)
	}

	for serverName := range servers {
		forcedHosts[serverName] = []string{serverName}
	}

	var existingContent string
	if data, err := os.ReadFile(velocityTomlPath); err == nil {
		existingContent = string(data)
		logger.Info("Updating existing velocity.toml", zap.String("path", velocityTomlPath))
	} else {
		logger.Info("Creating new velocity.toml", zap.String("path", velocityTomlPath))
	}
	output := buildVelocityTomlWithServers(existingContent, servers, tryList, forcedHosts)

	if err := os.WriteFile(velocityTomlPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write velocity.toml: %w", err)
	}

	logger.Info("Successfully created/updated velocity.toml",
		zap.Int("servers", len(servers)),
		zap.Strings("try", tryList),
		zap.String("path", velocityTomlPath))

	return nil
}

func buildVelocityTomlWithServers(existingContent string, servers map[string]string, tryList []string, forcedHosts map[string][]string) string {
	var builder strings.Builder

	criticalSettings := map[string]string{
		"bind":                        "\"0.0.0.0:25565\"",
		"online-mode":                 "true",
		"force-key-authentication":    "false",
		"player-info-forwarding-mode": "\"modern\"",
		"forwarding-secret-file":      "\"forwarding.secret\"",
	}

	if existingContent != "" {
		managedSectionCommentLines := map[string]struct{}{
			"# ===== SpoutMC Managed Sections =====":                        {},
			"# The following sections are automatically managed by SpoutMC": {},
			"# Manual changes will be overwritten":                          {},
		}

		lines := strings.Split(existingContent, "\n")
		inServersSection := false
		inForcedHostsSection := false
		inOtherSection := false
		skipSection := false
		insideTryArray := false

		for i := 0; i < len(lines); i++ {
			line := lines[i]
			trimmedLine := strings.TrimSpace(line)

			if _, isManagedComment := managedSectionCommentLines[trimmedLine]; isManagedComment {
				continue
			}

			if strings.HasPrefix(trimmedLine, "[") && strings.Contains(trimmedLine, "]") {
				sectionName := strings.TrimSpace(strings.Trim(trimmedLine, "[]"))

				if sectionName == "servers" {
					inServersSection = true
					inForcedHostsSection = false
					inOtherSection = false
					skipSection = true
					continue
				} else if sectionName == "forced-hosts" {
					inServersSection = false
					inForcedHostsSection = true
					inOtherSection = false
					skipSection = true
					continue
				} else {
					inServersSection = false
					inForcedHostsSection = false
					inOtherSection = true
					skipSection = false
					builder.WriteString(line + "\n")
					continue
				}
			}

			if skipSection && (inServersSection || inForcedHostsSection) {
				continue
			}

			if !inOtherSection && (strings.HasPrefix(trimmedLine, "try =") || strings.HasPrefix(trimmedLine, "try=")) {
				insideTryArray = true
				for i < len(lines) {
					if strings.Contains(lines[i], "]") {
						insideTryArray = false
						break
					}
					i++
				}
				continue
			}

			if insideTryArray {
				continue
			}
			if !inOtherSection {
				updated := false
				for key, value := range criticalSettings {
					if strings.HasPrefix(trimmedLine, key+" =") || strings.HasPrefix(trimmedLine, key+"=") {
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
			builder.WriteString(line + "\n")
		}
	} else {
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

	builder.WriteString("\n# ===== SpoutMC Managed Sections =====\n")
	builder.WriteString("# The following sections are automatically managed by SpoutMC\n")
	builder.WriteString("# Manual changes will be overwritten\n\n")

	builder.WriteString("[servers]\n")
	for serverName, serverAddr := range servers {
		builder.WriteString(fmt.Sprintf("%s = \"%s\"\n", serverName, serverAddr))
	}

	builder.WriteString("try = [\n")
	for i, serverName := range tryList {
		if i == len(tryList)-1 {
			builder.WriteString(fmt.Sprintf("    \"%s\"\n", serverName))
		} else {
			builder.WriteString(fmt.Sprintf("    \"%s\",\n", serverName))
		}
	}
	builder.WriteString("]\n")

	if len(forcedHosts) > 0 {
		builder.WriteString("\n[forced-hosts]\n")
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

func SyncVelocityToml(cfg *models.SpoutConfiguration) error {
	return CreateOrUpdateVelocityToml(cfg)
}

func SyncVelocityTomlAndRestartProxy(ctx context.Context, cfg *models.SpoutConfiguration) error {
	if err := SyncVelocityToml(cfg); err != nil {
		return err
	}

	if err := RestartProxyContainer(ctx); err != nil {
		return fmt.Errorf("velocity.toml updated but proxy restart failed: %w", err)
	}

	return nil
}

func UpdateVelocityTomlAddServer(cfg *models.SpoutConfiguration, serverName string, port int, isLobby bool) error {
	logger.Info("Adding server to velocity.toml",
		zap.String("server", serverName),
		zap.Int("port", port),
		zap.Bool("lobby", isLobby))

	return SyncVelocityToml(cfg)
}

func UpdateVelocityTomlRemoveServer(cfg *models.SpoutConfiguration, serverName string) error {
	logger.Info("Removing server from velocity.toml",
		zap.String("server", serverName))

	return SyncVelocityToml(cfg)
}

func getVelocityTomlPath(proxyServer *models.SpoutServer, dataPath string) (string, error) {
	if len(proxyServer.Volumes) == 0 {
		return "", fmt.Errorf("proxy server has no volumes configured")
	}

	containerPath := proxyServer.Volumes[0].Containerpath

	cleanContainerPath := strings.TrimPrefix(containerPath, "/")
	velocityTomlPath := filepath.Join(dataPath, proxyServer.Name, cleanContainerPath, "velocity.toml")

	return velocityTomlPath, nil
}
func getServerPort(server *models.SpoutServer) string {
	if len(server.Ports) > 0 {
		return server.Ports[0].ContainerPort
	}
	return ""
}

func getInternalContainerPort(server *models.SpoutServer) string {
	if len(server.Ports) > 0 {
		return server.Ports[0].ContainerPort
	}
	return "25565"
}

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

	if cachedVelocitySecret != "" {
		return cachedVelocitySecret
	}

	envSecret := os.Getenv("VELOCITY_SECRET")
	if envSecret != "" {
		logger.Info("Using Velocity secret from VELOCITY_SECRET environment variable")
		cachedVelocitySecret = envSecret
		return cachedVelocitySecret
	}

	if dataPath != "" && proxyName != "" {
		proxySecret, err := readProxyForwardingSecret(dataPath, proxyName)
		if err == nil && proxySecret != "" {
			logger.Info("Using existing Velocity secret from proxy's forwarding.secret file")
			cachedVelocitySecret = proxySecret
			return cachedVelocitySecret
		}
	}

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		logger.Error("Failed to generate random secret", zap.Error(err))
		return "changeme-please-set-velocity-secret"
	}

	cachedVelocitySecret = base64.StdEncoding.EncodeToString(secretBytes)
	logger.Info("Generated new Velocity forwarding secret",
		zap.String("secret", cachedVelocitySecret))
	logger.Warn("⚠️  IMPORTANT: This secret will be used for all backend servers")
	logger.Warn("⚠️  Optionally set VELOCITY_SECRET environment variable to use a custom secret")

	return cachedVelocitySecret
}

func readProxyForwardingSecret(dataPath string, proxyName string) (string, error) {
	secretPath := filepath.Join(dataPath, proxyName, "server", "forwarding.secret")

	if data, err := os.ReadFile(secretPath); err == nil {
		secret := strings.TrimSpace(string(data))
		if secret != "" {
			return secret, nil
		}
	}

	return "", fmt.Errorf("forwarding.secret file not found at %s", secretPath)
}

func ensureForwardingSecret(secretPath string) error {
	if _, err := os.Stat(secretPath); err == nil {
		if data, err := os.ReadFile(secretPath); err == nil {
			secret := strings.TrimSpace(string(data))
			velocitySecretMutex.Lock()
			cachedVelocitySecret = secret
			velocitySecretMutex.Unlock()
			logger.Info("Using existing forwarding secret", zap.String("path", secretPath))
			return nil
		}
	}

	secret := GetOrGenerateVelocitySecret("", "")

	if err := os.WriteFile(secretPath, []byte(secret), 0600); err != nil {
		return fmt.Errorf("failed to write forwarding secret: %w", err)
	}

	logger.Info("✅ Forwarding secret created successfully",
		zap.String("path", secretPath),
		zap.Int("length", len(secret)))

	return nil
}
