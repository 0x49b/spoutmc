package config

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func EnsureConfigExists() error {
	logger := log.GetLogger(log.ModuleConfig)
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	configDir := filepath.Join(wd, "config")
	configPath := filepath.Join(configDir, "spoutmc.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		defaultConfig := createDefaultConfig(wd)

		if err := writeConfigWithComments(configPath, defaultConfig); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}

		logger.Info("✅ Created default configuration file", zap.String("path", configPath))
		logger.Warn("⚠️  SpoutMC has been initialized with a default configuration.")
		logger.Warn("⚠️  Please review and accept the EULA in config/spoutmc.yaml")
		logger.Warn("⚠️  Set 'eula.accepted: true' to continue.")

		return fmt.Errorf("EULA not accepted - please review config/spoutmc.yaml")
	}

	return checkEULAStatus(configPath)
}

func createDefaultConfig(workingDir string) models.SpoutConfiguration {
	serverDataPath := filepath.Join(workingDir, "server_data")

	return models.SpoutConfiguration{
		EULA: &models.EULAConfig{
			Accepted:   false,
			AcceptedOn: time.Now(),
		},
		Git: &models.GitConfig{
			Enabled: false,
		},
		Storage: &models.StorageConfig{
			DataPath: serverDataPath,
		},
		Files: &models.FilesConfig{
			ExcludePatterns: []string{
				"*.jar",
				"world",
				"world*",
				".DS_Store",
				".cache",
				"cache",
				"*.env",
				".rcon*",
				"eula.txt",
				"libraries",
				"versions",
				"plugins",
			},
		},
		Servers: []models.SpoutServer{},
	}
}

func writeConfigWithComments(path string, cfg models.SpoutConfiguration) error {
	var builder strings.Builder

	builder.WriteString("# Minecraft EULA (https://www.minecraft.net/en-us/eula)\n")
	builder.WriteString("# You must accept the EULA to run Minecraft servers\n")
	builder.WriteString("eula:\n")
	builder.WriteString(fmt.Sprintf("  accepted: %t\n", cfg.EULA.Accepted))
	builder.WriteString(fmt.Sprintf("  accepted_on: %s\n", cfg.EULA.AcceptedOn.Format(time.RFC3339)))
	builder.WriteString("\n")

	builder.WriteString("# GitOps configuration\n")
	builder.WriteString("git:\n")
	builder.WriteString(fmt.Sprintf("  enabled: %t\n", cfg.Git.Enabled))
	builder.WriteString("\n")

	builder.WriteString("# Storage configuration\n")
	builder.WriteString("storage:\n")
	builder.WriteString("  data_path: /path/where/server/data/is/stored\n")
	builder.WriteString("\n")

	builder.WriteString("# File browser configuration\n")
	builder.WriteString("files:\n")
	builder.WriteString("  exclude_patterns:\n")
	for _, pattern := range cfg.Files.ExcludePatterns {
		builder.WriteString(fmt.Sprintf("    - \"%s\"\n", pattern))
	}
	builder.WriteString("\n")

	builder.WriteString("# Server configurations, only needed if you do not use GitOps\n")
	builder.WriteString("servers: []\n")

	return os.WriteFile(path, []byte(builder.String()), 0644)
}

func checkEULAStatus(configPath string) error {
	logger := log.GetLogger(log.ModuleConfig)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg models.SpoutConfiguration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.EULA == nil || !cfg.EULA.Accepted {
		logger.Warn("⚠️  EULA not accepted")
		logger.Warn("⚠️  Please set 'eula.accepted: true' in config/spoutmc.yaml")
		return fmt.Errorf("EULA not accepted - please review config/spoutmc.yaml")
	}

	if cfg.EULA.AcceptedOn.IsZero() {
		logger.Info("Updating EULA acceptance timestamp")
		cfg.EULA.AcceptedOn = time.Now()

		if err := updateEULATimestamp(configPath, cfg.EULA.AcceptedOn); err != nil {
			logger.Warn("Failed to update EULA timestamp", zap.Error(err))
		}
	}

	logger.Info("✅ EULA accepted", zap.Time("accepted_on", cfg.EULA.AcceptedOn))
	return nil
}

func updateEULATimestamp(configPath string, timestamp time.Time) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	updated := false
	if len(node.Content) > 0 {
		rootNode := node.Content[0]
		for i := 0; i < len(rootNode.Content)-1; i += 2 {
			keyNode := rootNode.Content[i]
			valueNode := rootNode.Content[i+1]

			if keyNode.Value == "eula" && valueNode.Kind == yaml.MappingNode {
				for j := 0; j < len(valueNode.Content)-1; j += 2 {
					eulaKeyNode := valueNode.Content[j]
					eulaValueNode := valueNode.Content[j+1]

					if eulaKeyNode.Value == "accepted_on" {
						eulaValueNode.Value = timestamp.Format(time.RFC3339)
						updated = true
						break
					}
				}
				break
			}
		}
	}

	if !updated {
		return fmt.Errorf("failed to find accepted_on field in YAML structure")
	}

	output, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
