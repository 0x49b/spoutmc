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

// EnsureConfigExists checks if config file exists, creates it if not
// Returns error if file creation fails or EULA not accepted
func EnsureConfigExists() error {
	logger := log.GetLogger(log.ModuleConfig)
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	configDir := filepath.Join(wd, "config")
	configPath := filepath.Join(configDir, "spoutmc.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create config directory if it doesn't exist
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create default configuration
		defaultConfig := createDefaultConfig(wd)

		// Write configuration to file with comments
		if err := writeConfigWithComments(configPath, defaultConfig); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}

		logger.Info("✅ Created default configuration file", zap.String("path", configPath))
		logger.Warn("⚠️  SpoutMC has been initialized with a default configuration.")
		logger.Warn("⚠️  Please review and accept the EULA in config/spoutmc.yaml")
		logger.Warn("⚠️  Set 'eula.accepted: true' to continue.")

		return fmt.Errorf("EULA not accepted - please review config/spoutmc.yaml")
	}

	// File exists, check EULA status
	return checkEULAStatus(configPath)
}

// createDefaultConfig creates a default SpoutConfiguration
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

// writeConfigWithComments writes the configuration to a YAML file with comments
func writeConfigWithComments(path string, cfg models.SpoutConfiguration) error {
	var builder strings.Builder

	// Write EULA section with comments
	builder.WriteString("# Minecraft EULA (https://www.minecraft.net/en-us/eula)\n")
	builder.WriteString("# You must accept the EULA to run Minecraft servers\n")
	builder.WriteString("eula:\n")
	builder.WriteString(fmt.Sprintf("  accepted: %t\n", cfg.EULA.Accepted))
	builder.WriteString(fmt.Sprintf("  accepted_on: %s\n", cfg.EULA.AcceptedOn.Format(time.RFC3339)))
	builder.WriteString("\n")

	// Write Git section
	builder.WriteString("# GitOps configuration\n")
	builder.WriteString("git:\n")
	builder.WriteString(fmt.Sprintf("  enabled: %t\n", cfg.Git.Enabled))
	builder.WriteString("\n")

	// Write Storage section
	builder.WriteString("# Storage configuration\n")
	builder.WriteString("storage:\n")
	builder.WriteString(fmt.Sprintf("  data_path: %s\n", cfg.Storage.DataPath))
	builder.WriteString("\n")

	// Write Files section
	builder.WriteString("# File browser configuration\n")
	builder.WriteString("files:\n")
	builder.WriteString("  exclude_patterns:\n")
	for _, pattern := range cfg.Files.ExcludePatterns {
		builder.WriteString(fmt.Sprintf("    - \"%s\"\n", pattern))
	}
	builder.WriteString("\n")

	// Write Servers section
	builder.WriteString("# Server configurations\n")
	builder.WriteString("servers: []\n")

	// Write to file
	return os.WriteFile(path, []byte(builder.String()), 0644)
}

// checkEULAStatus checks if EULA is accepted and updates timestamp if needed
func checkEULAStatus(configPath string) error {
	logger := log.GetLogger(log.ModuleConfig)

	// Read current config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg models.SpoutConfiguration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Check EULA
	if cfg.EULA == nil || !cfg.EULA.Accepted {
		logger.Warn("⚠️  EULA not accepted")
		logger.Warn("⚠️  Please set 'eula.accepted: true' in config/spoutmc.yaml")
		return fmt.Errorf("EULA not accepted - please review config/spoutmc.yaml")
	}

	// Update timestamp if it's zero value (not set)
	if cfg.EULA.AcceptedOn.IsZero() {
		logger.Info("Updating EULA acceptance timestamp")
		cfg.EULA.AcceptedOn = time.Now()

		// Re-read file to preserve comments
		if err := updateEULATimestamp(configPath, cfg.EULA.AcceptedOn); err != nil {
			logger.Warn("Failed to update EULA timestamp", zap.Error(err))
			// Don't fail startup, just log the warning
		}
	}

	logger.Info("✅ EULA accepted", zap.Time("accepted_on", cfg.EULA.AcceptedOn))
	return nil
}

// updateEULATimestamp updates the accepted_on timestamp in the YAML file while preserving structure and comments
func updateEULATimestamp(configPath string, timestamp time.Time) error {
	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML with node structure to preserve comments
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Find and update the accepted_on field
	updated := false
	if len(node.Content) > 0 {
		rootNode := node.Content[0]
		for i := 0; i < len(rootNode.Content)-1; i += 2 {
			keyNode := rootNode.Content[i]
			valueNode := rootNode.Content[i+1]

			if keyNode.Value == "eula" && valueNode.Kind == yaml.MappingNode {
				// Found EULA section
				for j := 0; j < len(valueNode.Content)-1; j += 2 {
					eulaKeyNode := valueNode.Content[j]
					eulaValueNode := valueNode.Content[j+1]

					if eulaKeyNode.Value == "accepted_on" {
						// Update the timestamp
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

	// Marshal back to YAML
	output, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
