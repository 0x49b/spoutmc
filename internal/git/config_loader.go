package git

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/infrastructure"
	"spoutmc/internal/models"
	"spoutmc/internal/notifications"
	"strings"

	"go.uber.org/zap"
)

var logEmoji = "🗄️"

func LoadServersFromRepository(repoPath string) (*models.SpoutConfiguration, error) {
	logger.Info(logEmoji+" Loading server configurations from Git repository", zap.String("path", repoPath))

	config := &models.SpoutConfiguration{
		Servers: make([]models.SpoutServer, 0),
	}

	serverNames := make(map[string]bool)

	serversPath := filepath.Join(repoPath, "servers")
	searchPath := repoPath
	if info, err := os.Stat(serversPath); err == nil && info.IsDir() {
		searchPath = serversPath
	} else {
		logger.Warn(logEmoji+" No servers directory found, using legacy repository-wide YAML scan",
			zap.String("path", repoPath))
	}

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.Contains(path, ".git") {
			return nil
		}

		if strings.Contains(path, "infrastructure") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn(logEmoji+" Failed to read YAML file, skipping",
				zap.String("file", path),
				zap.Error(err))
			return nil // Continue processing other files
		}

		server, err := ParseServerYAML(data)
		if err != nil {
			logger.Warn(logEmoji+" Failed to parse server YAML file, skipping",
				zap.String("file", path),
				zap.Error(err))
			_ = notifications.UpsertOpen(
				fmt.Sprintf("gitops:server-parse:%s", path),
				"warning",
				"GitOps server manifest skipped",
				fmt.Sprintf("File %s was skipped: %v", path, err),
				"gitops",
			)
			return nil // Continue processing other files
		}

		if server.Name == "" {
			logger.Warn(logEmoji+" Server configuration missing 'name' field, skipping",
				zap.String("file", path))
			_ = notifications.UpsertOpen(
				fmt.Sprintf("gitops:server-invalid:%s", path),
				"warning",
				"GitOps server manifest skipped",
				fmt.Sprintf("File %s is missing required field 'name'.", path),
				"gitops",
			)
			return nil
		}

		if server.Image == "" {
			logger.Warn(logEmoji+" Server configuration missing 'image' field, skipping",
				zap.String("file", path),
				zap.String("server", server.Name))
			_ = notifications.UpsertOpen(
				fmt.Sprintf("gitops:server-invalid:%s", path),
				"warning",
				"GitOps server manifest skipped",
				fmt.Sprintf("File %s is missing required field 'image'.", path),
				"gitops",
			)
			return nil
		}

		if err := ValidateServerConfig(&server); err != nil {
			logger.Warn(logEmoji+" Invalid server configuration, skipping",
				zap.String("file", path),
				zap.Error(err))
			_ = notifications.UpsertOpen(
				fmt.Sprintf("gitops:server-invalid:%s", path),
				"warning",
				"GitOps server manifest skipped",
				fmt.Sprintf("File %s is invalid: %v", path, err),
				"gitops",
			)
			return nil
		}

		if serverNames[server.Name] {
			logger.Warn(logEmoji+" Duplicate server name found, skipping",
				zap.String("file", path),
				zap.String("server", server.Name))
			_ = notifications.UpsertOpen(
				fmt.Sprintf("gitops:server-duplicate:%s", server.Name),
				"warning",
				"GitOps duplicate server name",
				fmt.Sprintf("Server name %q appears more than once. One manifest was skipped.", server.Name),
				"gitops",
			)
			return nil
		}

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

func ValidateServerConfig(server *models.SpoutServer) error {
	if server.Name == "" {
		return fmt.Errorf("server name is required")
	}

	if server.Image == "" {
		return fmt.Errorf("server image is required")
	}

	for i, port := range server.Ports {
		if port.HostPort == "" || port.ContainerPort == "" {
			return fmt.Errorf("server %s: port mapping %d has empty host or container port", server.Name, i)
		}
	}

	for i, volume := range server.Volumes {
		if volume.Containerpath == "" {
			return fmt.Errorf("server %s: volume mapping %d has empty container path", server.Name, i)
		}
	}

	if server.RestartPolicy != nil && server.RestartPolicy.Container != nil {
		containerPolicy := server.RestartPolicy.Container
		switch containerPolicy.Policy {
		case models.DockerRestartPolicyNo,
			models.DockerRestartPolicyOnFailure,
			models.DockerRestartPolicyAlways,
			models.DockerRestartPolicyUnlessStopped:
		default:
			return fmt.Errorf("server %s: restartPolicy.container.policy must be one of: no, on-failure, always, unless-stopped", server.Name)
		}

		if containerPolicy.Policy != models.DockerRestartPolicyOnFailure && containerPolicy.MaxRetries != nil {
			return fmt.Errorf("server %s: restartPolicy.container.maxRetries is only supported when policy is on-failure", server.Name)
		}

		if containerPolicy.Policy == models.DockerRestartPolicyOnFailure &&
			containerPolicy.MaxRetries != nil &&
			*containerPolicy.MaxRetries < 1 {
			return fmt.Errorf("server %s: restartPolicy.container.maxRetries must be at least 1 when policy is on-failure", server.Name)
		}
	}

	return nil
}

func LoadInfrastructureFromRepository(repoPath string) ([]infrastructure.InfrastructureContainer, error) {
	logger.Info(logEmoji+" Loading infrastructure configurations from Git repository", zap.String("path", repoPath))

	infrastructurePath := filepath.Join(repoPath, "infrastructure")

	if _, err := os.Stat(infrastructurePath); os.IsNotExist(err) {
		logger.Info(logEmoji + " No infrastructure directory found, skipping")
		return []infrastructure.InfrastructureContainer{}, nil
	}

	containers := make([]infrastructure.InfrastructureContainer, 0)

	err := filepath.Walk(infrastructurePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn(logEmoji+" Failed to read infrastructure YAML file, skipping",
				zap.String("file", path),
				zap.Error(err))
			return nil
		}

		container, err := ParseInfrastructureYAML(data)
		if err != nil {
			logger.Warn(logEmoji+" Failed to parse infrastructure YAML file, skipping",
				zap.String("file", path),
				zap.Error(err))
			return nil
		}

		if container.Name == "" {
			logger.Warn(logEmoji+" Infrastructure configuration missing 'name' field, skipping",
				zap.String("file", path))
			return nil
		}

		if container.Image == "" {
			logger.Warn(logEmoji+" Infrastructure configuration missing 'image' field, skipping",
				zap.String("file", path),
				zap.String("name", container.Name))
			return nil
		}

		containers = append(containers, container)

		logger.Debug(logEmoji+" Loaded infrastructure configuration",
			zap.String("file", path),
			zap.String("name", container.Name),
			zap.String("image", container.Image))

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk infrastructure directory: %w", err)
	}

	logger.Info(logEmoji+" Successfully loaded infrastructure configurations from Git",
		zap.Int("count", len(containers)))

	return containers, nil
}
