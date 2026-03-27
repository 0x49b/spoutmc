package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/utils/path"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var spoutConfiguration models.SpoutConfiguration // package-scoped state

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

func All() models.SpoutConfiguration {
	return spoutConfiguration
}

func GetServerConfigForContainerName(name string) (models.SpoutServer, error) {
	for _, s := range spoutConfiguration.Servers {
		if s.Name == name {
			return s, nil
		}
	}
	return models.SpoutServer{}, errors.New("No matching config found")
}

func IsValidServerName(name string) bool {
	for _, s := range spoutConfiguration.Servers {
		if s.Name == name {
			return true
		}
	}
	return false
}

func UpdateConfiguration(newConfig models.SpoutConfiguration) {
	normalizeConfigurationPaths(&newConfig)
	spoutConfiguration = newConfig
}

func IsGitOpsEnabled() bool {
	return spoutConfiguration.Git != nil && spoutConfiguration.Git.Enabled
}

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

		if server.Proxy {
			continue
		}

		if server.Env == nil {
			server.Env = make(map[string]string)
		}

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
