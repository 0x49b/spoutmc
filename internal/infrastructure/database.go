package infrastructure

import (
	"context"
	"fmt"

	"spoutmc/internal/docker"
	"spoutmc/internal/models"

	"go.uber.org/zap"
)

const (
	InfrastructureLabel = "io.spout.infrastructure"
	DatabaseLabel       = "io.spout.database"
)

// InfrastructureContainer represents an infrastructure container configuration
type InfrastructureContainer struct {
	Name    string            `yaml:"name"`
	Image   string            `yaml:"image"`
	Restart string            `yaml:"restart"`
	Volumes []string          `yaml:"volumes"`
	Ports   []PortMapping     `yaml:"ports"`
	Env     map[string]string `yaml:"env"`
}

// PortMapping represents a port mapping configuration
type PortMapping struct {
	Host      string `yaml:"host"`
	Container string `yaml:"container"`
}

// CreateDatabaseContainer creates and starts the database container
func CreateDatabaseContainer(ctx context.Context, infraConfig InfrastructureContainer, dataPath string, passwords map[string]string, logger *zap.Logger) error {
	// Replace password placeholders with generated passwords
	env := make(map[string]string)
	for k, v := range infraConfig.Env {
		if v == "changeme" {
			// Replace with generated password
			if generatedPassword, exists := passwords[k]; exists {
				env[k] = generatedPassword
				logger.Info("Replaced password placeholder",
					zap.String("env_var", k))
			} else {
				env[k] = v
			}
		} else {
			env[k] = v
		}
	}

	// Convert infrastructure config to SpoutServer model
	server := models.SpoutServer{
		Name:  infraConfig.Name,
		Image: infraConfig.Image,
		Env:   env,
		Ports: convertPorts(infraConfig.Ports),
		Volumes: []models.SpoutServerVolumes{
			{Containerpath: "/var/lib/mysql"}, // Data directory
		},
	}

	logger.Info("Creating database container",
		zap.String("name", server.Name),
		zap.String("image", server.Image))

	// Pull image if needed
	docker.PullImage(ctx, server.Image)

	// Create container with infrastructure label
	containerID, err := docker.CreateInfrastructureContainer(ctx, server, dataPath)
	if err != nil {
		return fmt.Errorf("failed to create database container: %w", err)
	}

	logger.Info("Database container created",
		zap.String("container_id", containerID))

	// Start container
	if err := docker.StartContainerByIdSimple(ctx, containerID); err != nil {
		return fmt.Errorf("failed to start database container: %w", err)
	}

	logger.Info("Database container started successfully")
	return nil
}

// convertPorts converts infrastructure port mappings to SpoutServer port mappings
func convertPorts(infraPorts []PortMapping) []models.SpoutServerPorts {
	ports := make([]models.SpoutServerPorts, len(infraPorts))
	for i, p := range infraPorts {
		ports[i] = models.SpoutServerPorts{
			HostPort:      p.Host,
			ContainerPort: p.Container,
		}
	}
	return ports
}

// IsInfrastructureContainer checks if a container is an infrastructure container
func IsInfrastructureContainer(labels map[string]string) bool {
	value, exists := labels[InfrastructureLabel]
	return exists && value == "true"
}

// IsDatabaseContainer checks if a container is a database container
func IsDatabaseContainer(labels map[string]string) bool {
	value, exists := labels[DatabaseLabel]
	return exists && value == "true"
}
