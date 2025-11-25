package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// PaperGlobalConfig represents the structure of paper-global.yml
type PaperGlobalConfig struct {
	Version int                    `yaml:"_version"`
	Proxies map[string]interface{} `yaml:"proxies"`
	// We only care about proxies section, rest is preserved
	Other map[string]interface{} `yaml:",inline"`
}

// VelocityProxyConfig represents the velocity section in paper-global.yml
type VelocityProxyConfig struct {
	Enabled    bool   `yaml:"enabled"`
	OnlineMode bool   `yaml:"online-mode"`
	Secret     string `yaml:"secret"`
}

// EnsurePaperVelocityConfig ensures paper-global.yml has correct Velocity configuration
// If the file doesn't exist yet (first start), this returns nil and should be retried later
func EnsurePaperVelocityConfig(serverDataPath string, velocitySecret string) error {
	configPath := filepath.Join(serverDataPath, "config", "paper-global.yml")

	// Check if paper-global.yml exists (Paper generates it on first start)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Debug("paper-global.yml doesn't exist yet, will be created by Paper on first start",
			zap.String("path", configPath))
		return nil // Not an error, just not ready yet
	}

	// Read the existing file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read paper-global.yml: %w", err)
	}

	// Parse as generic YAML to preserve all existing settings
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse paper-global.yml: %w", err)
	}

	// Ensure proxies section exists
	if config["proxies"] == nil {
		config["proxies"] = make(map[string]interface{})
	}

	proxies, ok := config["proxies"].(map[string]interface{})
	if !ok {
		proxies = make(map[string]interface{})
		config["proxies"] = proxies
	}

	// Ensure velocity section exists
	if proxies["velocity"] == nil {
		proxies["velocity"] = make(map[string]interface{})
	}

	velocity, ok := proxies["velocity"].(map[string]interface{})
	if !ok {
		velocity = make(map[string]interface{})
		proxies["velocity"] = velocity
	}

	// Check if configuration is already correct
	needsUpdate := false

	// Check enabled
	if enabled, ok := velocity["enabled"].(bool); !ok || !enabled {
		needsUpdate = true
	}

	// Check online-mode
	if onlineMode, ok := velocity["online-mode"].(bool); !ok || !onlineMode {
		needsUpdate = true
	}

	// Check secret
	if secret, ok := velocity["secret"].(string); !ok || secret != velocitySecret {
		needsUpdate = true
	}

	// If configuration is already correct, skip writing
	if !needsUpdate {
		logger.Debug("paper-global.yml already has correct Velocity configuration, skipping write",
			zap.String("path", configPath))
		return nil
	}

	// Configuration needs update, log it
	logger.Info("Updating paper-global.yml with Velocity settings",
		zap.String("path", configPath))

	// Set Velocity configuration
	velocity["enabled"] = true
	velocity["online-mode"] = true
	velocity["secret"] = velocitySecret

	// Marshal back to YAML
	output, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal paper-global.yml: %w", err)
	}

	// Create backup
	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		logger.Warn("Failed to create backup of paper-global.yml", zap.Error(err))
	}

	// Write updated configuration
	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write paper-global.yml: %w", err)
	}

	logger.Info("✅ Successfully configured paper-global.yml for Velocity",
		zap.String("path", configPath),
		zap.Bool("velocity_enabled", true),
		zap.String("backup", backupPath))

	return nil
}

// CheckAndConfigurePaperServers checks all Paper servers and configures them for Velocity
// This should be called after servers have started at least once
func CheckAndConfigurePaperServers(dataPath string, velocitySecret string) error {
	// Find all Paper server directories
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return fmt.Errorf("failed to read data path: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip proxy server directory
		serverName := entry.Name()
		if strings.Contains(strings.ToLower(serverName), "proxy") {
			continue
		}

		// Check if this looks like a Paper server (has data/config directory)
		serverDataPath := filepath.Join(dataPath, serverName, "data")
		configDir := filepath.Join(serverDataPath, "config")
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			continue // Silently skip, server not ready yet
		}

		// Configure this Paper server (only writes if needed)
		if err := EnsurePaperVelocityConfig(serverDataPath, velocitySecret); err != nil {
			logger.Error("Failed to configure Paper server for Velocity",
				zap.String("server", serverName),
				zap.Error(err))
		}
	}

	return nil
}
