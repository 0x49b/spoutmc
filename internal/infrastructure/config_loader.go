package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type InfrastructureContainer struct {
	Name    string            `yaml:"name"`
	Image   string            `yaml:"image"`
	Restart string            `yaml:"restart"`
	Volumes []string          `yaml:"volumes"`
	Ports   []PortMapping     `yaml:"ports"`
	Env     map[string]string `yaml:"env"`
}

type PortMapping struct {
	Host          string `yaml:"host"`
	Container     string `yaml:"container"`
	HostPort      string `yaml:"hostPort"`
	ContainerPort string `yaml:"containerPort"`
}

type InfrastructureConfig struct {
	Infrastructure []InfrastructureContainer `yaml:"infrastructure"`
}

func LoadInfrastructureFromLocalConfig(configPath string, logger *zap.Logger) ([]InfrastructureContainer, error) {
	logger.Info("Loading infrastructure from local config", zap.String("path", configPath))

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Info("Infrastructure config file not found, skipping", zap.String("path", configPath))
		return []InfrastructureContainer{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read infrastructure config: %w", err)
	}

	var config InfrastructureConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse infrastructure config: %w", err)
	}

	for i, container := range config.Infrastructure {
		if container.Name == "" {
			return nil, fmt.Errorf("infrastructure container at index %d is missing 'name' field", i)
		}
		if container.Image == "" {
			return nil, fmt.Errorf("infrastructure container '%s' is missing 'image' field", container.Name)
		}
	}

	logger.Info("Loaded infrastructure containers from local config", zap.Int("count", len(config.Infrastructure)))
	return config.Infrastructure, nil
}

func GetDefaultInfrastructureConfigPath() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return "config/infrastructure.yaml"
	}
	return filepath.Join(workingDir, "config", "infrastructure.yaml")
}
