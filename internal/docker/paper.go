package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type PaperGlobalConfig struct {
	Version int                    `yaml:"_version"`
	Proxies map[string]interface{} `yaml:"proxies"`
	Other   map[string]interface{} `yaml:",inline"`
}

type VelocityProxyConfig struct {
	Enabled    bool   `yaml:"enabled"`
	OnlineMode bool   `yaml:"online-mode"`
	Secret     string `yaml:"secret"`
}

func EnsurePaperVelocityConfig(serverDataPath string, velocitySecret string) error {
	configPath := filepath.Join(serverDataPath, "config", "paper-global.yml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Debug("paper-global.yml doesn't exist yet, will be created by Paper on first start",
			zap.String("path", configPath))
		return nil // Not an error, just not ready yet
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read paper-global.yml: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse paper-global.yml: %w", err)
	}

	if config["proxies"] == nil {
		config["proxies"] = make(map[string]interface{})
	}

	proxies, ok := config["proxies"].(map[string]interface{})
	if !ok {
		proxies = make(map[string]interface{})
		config["proxies"] = proxies
	}

	if proxies["velocity"] == nil {
		proxies["velocity"] = make(map[string]interface{})
	}

	velocity, ok := proxies["velocity"].(map[string]interface{})
	if !ok {
		velocity = make(map[string]interface{})
		proxies["velocity"] = velocity
	}

	needsUpdate := false

	if enabled, ok := velocity["enabled"].(bool); !ok || !enabled {
		needsUpdate = true
	}

	if onlineMode, ok := velocity["online-mode"].(bool); !ok || !onlineMode {
		needsUpdate = true
	}

	if secret, ok := velocity["secret"].(string); !ok || secret != velocitySecret {
		needsUpdate = true
	}

	if !needsUpdate {
		logger.Debug("paper-global.yml already has correct Velocity configuration, skipping write",
			zap.String("path", configPath))
		return nil
	}

	logger.Info("Updating paper-global.yml with Velocity settings",
		zap.String("path", configPath))

	velocity["enabled"] = true
	velocity["online-mode"] = true
	velocity["secret"] = velocitySecret

	output, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal paper-global.yml: %w", err)
	}

	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		logger.Warn("Failed to create backup of paper-global.yml", zap.Error(err))
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write paper-global.yml: %w", err)
	}

	logger.Info("✅ Successfully configured paper-global.yml for Velocity",
		zap.String("path", configPath),
		zap.Bool("velocity_enabled", true),
		zap.String("backup", backupPath))

	return nil
}

func CheckAndConfigurePaperServers(dataPath string, velocitySecret string) error {
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return fmt.Errorf("failed to read data path: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serverName := entry.Name()
		if strings.Contains(strings.ToLower(serverName), "proxy") {
			continue
		}

		serverDataPath := filepath.Join(dataPath, serverName, "data")
		configDir := filepath.Join(serverDataPath, "config")
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			continue // Silently skip, server not ready yet
		}

		if err := EnsurePaperVelocityConfig(serverDataPath, velocitySecret); err != nil {
			logger.Error("Failed to configure Paper server for Velocity",
				zap.String("server", serverName),
				zap.Error(err))
		}
	}

	return nil
}
