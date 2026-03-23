package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/pathutil"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var spoutConfiguration models.SpoutConfiguration // package-scoped state

// ReadConfiguration finds and loads spoutmc.yaml|yml into package state.
// It also returns the loaded config for convenience.
func ReadConfiguration() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	candidates := []string{
		filepath.Join(wd, "config", "spoutmc.yaml"),
		filepath.Join(wd, "config", "spoutmc.yml"),
	}

	var data []byte
	var usedPath string
	for _, candidate := range candidates {
		if _, statErr := os.Stat(candidate); statErr == nil {
			usedPath = candidate
			data, err = os.ReadFile(candidate)
			if err != nil {
				return err
			}
			break
		}
	}

	if usedPath == "" {
		return errors.New(fmt.Sprintf("no configuration file found, looked for: %v", candidates))
	}

	if err := yaml.Unmarshal(data, &spoutConfiguration); err != nil {
		return err
	}

	normalizeConfigurationPaths(&spoutConfiguration)

	return nil
}

// All returns the currently loaded configuration.
func All() models.SpoutConfiguration {
	return spoutConfiguration
}

// GetServerConfigForContainerName looks up a server by its Name.
func GetServerConfigForContainerName(name string) (models.SpoutServer, error) {
	for _, s := range spoutConfiguration.Servers {
		if s.Name == name {
			return s, nil
		}
	}
	return models.SpoutServer{}, errors.New("No matching config found")
}

// IsValidServerName reports whether name matches a server in the loaded Spout configuration.
func IsValidServerName(name string) bool {
	for _, s := range spoutConfiguration.Servers {
		if s.Name == name {
			return true
		}
	}
	return false
}

// UpdateConfiguration updates the package-scoped configuration.
// This is used by GitOps to update configuration from Git repository.
func UpdateConfiguration(newConfig models.SpoutConfiguration) {
	normalizeConfigurationPaths(&newConfig)
	spoutConfiguration = newConfig
}

// IsGitOpsEnabled checks if GitOps mode is enabled in the configuration.
func IsGitOpsEnabled() bool {
	return spoutConfiguration.Git != nil && spoutConfiguration.Git.Enabled
}

// GetGitConfig returns the Git configuration if GitOps is enabled.
func GetGitConfig() *models.GitConfig {
	if spoutConfiguration.Git != nil {
		return spoutConfiguration.Git
	}
	return nil
}

func normalizeConfigurationPaths(cfg *models.SpoutConfiguration) {
	if cfg == nil || cfg.Storage == nil {
		return
	}

	normalizedPath := pathutil.NormalizeHostPath(cfg.Storage.DataPath)
	if normalizedPath != cfg.Storage.DataPath {
		log.GetLogger(log.ModuleConfig).Info("Normalized storage data path",
			zap.String("original", cfg.Storage.DataPath),
			zap.String("normalized", normalizedPath))
	}
	cfg.Storage.DataPath = normalizedPath
}

// EnsureVelocityEnvVars checks all backend servers and injects required Velocity forwarding
// environment variables if they're missing. Returns true if any servers were updated.
func EnsureVelocityEnvVars(velocitySecret string) bool {
	updated := false
	requiredVars := map[string]string{
		"REPLACE_ENV_VARIABLES":    "TRUE",
		"ENV_VARIABLE_PREFIX":      "CFG_",
		"CFG_VELOCITY_ENABLED":     "true",
		"CFG_VELOCITY_ONLINE_MODE": "true",
		"CFG_VELOCITY_SECRET":      velocitySecret,
	}

	for i := range spoutConfiguration.Servers {
		server := &spoutConfiguration.Servers[i]

		// Skip proxy servers - they don't need backend forwarding config
		if server.Proxy {
			continue
		}

		// Initialize env map if nil
		if server.Env == nil {
			server.Env = make(map[string]string)
		}

		// Check if server is missing any required vars
		serverUpdated := false
		for key, value := range requiredVars {
			if _, exists := server.Env[key]; !exists {
				server.Env[key] = value
				serverUpdated = true
			}
		}

		if serverUpdated {
			updated = true
			logger := log.GetLogger(log.ModuleConfig)
			logger.Info("Injected Velocity forwarding env vars",
				zap.String("server", server.Name),
				zap.Bool("lobby", server.Lobby))
		}
	}

	return updated
}
