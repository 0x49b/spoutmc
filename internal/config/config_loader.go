package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/models"

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

	return nil
}

// All returns the currently loaded configuration.
func All() models.SpoutConfiguration {
	return spoutConfiguration
}

// GetServerConfigForContainerName looks up a server by its Name.
func GetServerConfigForContainerName(name string) models.SpoutServer {
	for _, s := range spoutConfiguration.Servers {
		if s.Name == name {
			return s
		}
	}
	return models.SpoutServer{}
}
