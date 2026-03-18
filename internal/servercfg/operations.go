package servercfg

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/config"
	"spoutmc/internal/git"
	"spoutmc/internal/log"
	"spoutmc/internal/models"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var logger = log.GetLogger(log.ModuleServerCfg)

// AddServerToGit adds a new server configuration to the git repository
func AddServerToGit(server models.SpoutServer) error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git configuration not found")
	}

	// Marshal using GitOps manifest format (apiVersion/kind/metadata/spec)
	yamlData, err := git.MarshalServerManifest(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	// Write to git repo under /servers directory
	repoPath := gitConfig.LocalPath
	serversDir := filepath.Join(repoPath, "servers")

	// Create servers directory if it doesn't exist
	if err := os.MkdirAll(serversDir, 0755); err != nil {
		return fmt.Errorf("failed to create servers directory: %w", err)
	}

	serverFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", server.Name))

	if err := os.WriteFile(serverFilePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write server config file: %w", err)
	}

	// Commit and push changes
	if err := git.CommitAndPushChanges(repoPath, fmt.Sprintf("Add server: %s", server.Name)); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	// Reload configuration from git to update in-memory state
	if err := git.LoadConfigurationFromGit(); err != nil {
		return fmt.Errorf("failed to reload configuration from git: %w", err)
	}

	logger.Info("Server config added to git repository and configuration reloaded", zap.String("file", serverFilePath))
	return nil
}

// AddServerToLocalConfig adds a new server to the local spoutmc.yaml file
func AddServerToLocalConfig(server models.SpoutServer) error {
	// Get current configuration
	currentConfig := config.All()

	// Add new server
	currentConfig.Servers = append(currentConfig.Servers, server)

	// Write back to file
	return writeLocalConfig(currentConfig)
}

// UpdateServerInGit updates a server configuration in the git repository
func UpdateServerInGit(oldName string, server models.SpoutServer) error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git configuration not found")
	}

	repoPath := gitConfig.LocalPath
	serversDir := filepath.Join(repoPath, "servers")

	// If name changed, remove old file
	if oldName != server.Name {
		oldFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", oldName))
		if err := os.Remove(oldFilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old server config file: %w", err)
		}
	}

	// Marshal using GitOps manifest format (apiVersion/kind/metadata/spec)
	yamlData, err := git.MarshalServerManifest(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	// Write to git repo
	serverFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", server.Name))
	if err := os.WriteFile(serverFilePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write server config file: %w", err)
	}

	// Commit and push changes
	if err := git.CommitAndPushChanges(repoPath, fmt.Sprintf("Update server: %s", server.Name)); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	// Reload configuration from git to update in-memory state
	if err := git.LoadConfigurationFromGit(); err != nil {
		return fmt.Errorf("failed to reload configuration from git: %w", err)
	}

	logger.Info("Server config updated in git repository and configuration reloaded", zap.String("file", serverFilePath))
	return nil
}

// UpdateServerInLocalConfig updates a server in the local spoutmc.yaml file
func UpdateServerInLocalConfig(oldName string, server models.SpoutServer) error {
	// Get current configuration
	currentConfig := config.All()

	// Find and update the server
	found := false
	for i, s := range currentConfig.Servers {
		if s.Name == oldName {
			currentConfig.Servers[i] = server
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("server %s not found in configuration", oldName)
	}

	// Write back to file
	return writeLocalConfig(currentConfig)
}

// RemoveServerFromGit removes a server configuration from the git repository
func RemoveServerFromGit(serverName string) error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git configuration not found")
	}

	// Remove the server file
	repoPath := gitConfig.LocalPath
	serversDir := filepath.Join(repoPath, "servers")
	serverFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", serverName))

	if err := os.Remove(serverFilePath); err != nil {
		return fmt.Errorf("failed to remove server config file: %w", err)
	}

	// Commit and push changes
	if err := git.CommitAndPushChanges(repoPath, fmt.Sprintf("Remove server: %s", serverName)); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	// Reload configuration from git to update in-memory state
	if err := git.LoadConfigurationFromGit(); err != nil {
		return fmt.Errorf("failed to reload configuration from git: %w", err)
	}

	logger.Info("Server config removed from git repository and configuration reloaded", zap.String("file", serverFilePath))
	return nil
}

// RemoveServerFromLocalConfig removes a server from the local spoutmc.yaml file
func RemoveServerFromLocalConfig(serverName string) error {
	// Get current configuration
	currentConfig := config.All()

	// Find and remove the server
	newServers := make([]models.SpoutServer, 0)
	found := false
	for _, server := range currentConfig.Servers {
		if server.Name != serverName {
			newServers = append(newServers, server)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("server %s not found in configuration", serverName)
	}

	// Update servers list
	currentConfig.Servers = newServers

	// Write back to file
	return writeLocalConfig(currentConfig)
}

// writeLocalConfig writes the configuration to the local spoutmc.yaml file
func writeLocalConfig(cfg models.SpoutConfiguration) error {
	// Marshal to YAML
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Write to config file (try both .yaml and .yml)
	configPaths := []string{
		filepath.Join(wd, "config", "spoutmc.yaml"),
		filepath.Join(wd, "config", "spoutmc.yml"),
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		// Default to .yaml if neither exists
		configPath = configPaths[0]
	}

	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Reload configuration
	if err := config.ReadConfiguration(); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	logger.Info("Local config updated successfully", zap.String("path", configPath))
	return nil
}
