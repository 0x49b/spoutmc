package git

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/models"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var logEmoji = "🗄️"

// LoadServersFromRepository reads all YAML files from the repository and merges them into a SpoutConfiguration
func LoadServersFromRepository(repoPath string) (*models.SpoutConfiguration, error) {
	logger.Info(logEmoji+" Loading server configurations from Git repository", zap.String("path", repoPath))

	config := &models.SpoutConfiguration{
		Servers: make([]models.SpoutServer, 0),
	}

	// Track server names to detect duplicates
	serverNames := make(map[string]bool)

	// Walk through all files in the repository
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip .git directory
		if strings.Contains(path, ".git") {
			return nil
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Read the file
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn(logEmoji+" Failed to read YAML file, skipping",
				zap.String("file", path),
				zap.Error(err))
			return nil // Continue processing other files
		}

		// Try to parse as SpoutServer
		var server models.SpoutServer
		if err := yaml.Unmarshal(data, &server); err != nil {
			logger.Warn(logEmoji+" Failed to parse YAML file as SpoutServer, skipping",
				zap.String("file", path),
				zap.Error(err))
			return nil // Continue processing other files
		}

		// Validate server name is present
		if server.Name == "" {
			logger.Warn(logEmoji+" Server configuration missing 'name' field, skipping",
				zap.String("file", path))
			return nil
		}

		// Validate server image is present
		if server.Image == "" {
			logger.Warn(logEmoji+" Server configuration missing 'image' field, skipping",
				zap.String("file", path),
				zap.String("server", server.Name))
			return nil
		}

		// Check for duplicate server names
		if serverNames[server.Name] {
			logger.Warn(logEmoji+" Duplicate server name found, skipping",
				zap.String("file", path),
				zap.String("server", server.Name))
			return nil
		}

		// Add server to configuration
		serverNames[server.Name] = true
		config.Servers = append(config.Servers, server)

		logger.Debug(logEmoji+" Loaded server configuration",
			zap.String("file", path),
			zap.String("server", server.Name),
			zap.String("image", server.Image))

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk repository: %w", err)
	}

	if len(config.Servers) == 0 {
		return nil, fmt.Errorf("no valid server configurations found in repository")
	}

	logger.Info(logEmoji+" Successfully loaded server configurations from Git",
		zap.Int("count", len(config.Servers)))

	return config, nil
}

// ValidateServerConfig validates a server configuration
func ValidateServerConfig(server *models.SpoutServer) error {
	if server.Name == "" {
		return fmt.Errorf("server name is required")
	}

	if server.Image == "" {
		return fmt.Errorf("server image is required")
	}

	// Validate port mappings if present
	for i, port := range server.Ports {
		if port.HostPort == "" || port.ContainerPort == "" {
			return fmt.Errorf("server %s: port mapping %d has empty host or container port", server.Name, i)
		}
	}

	// Validate volume mappings if present
	for i, volume := range server.Volumes {
		if len(volume.Hostpath) == 0 || volume.Containerpath == "" {
			return fmt.Errorf("server %s: volume mapping %d has empty host or container path", server.Name, i)
		}
	}

	return nil
}
