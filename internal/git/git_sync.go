package git

import (
	"context"
	"fmt"
	"spoutmc/internal/config"

	"go.uber.org/zap"
)

var (
	globalRepo    *Repository
	globalPoller  *Poller
	globalWebhook *WebhookHandler
)

// InitializeGitOps initializes GitOps with the given configuration
func InitializeGitOps() error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git config is nil")
	}

	logger.Info("Initializing GitOps",
		zap.String("repository", gitConfig.Repository),
		zap.String("branch", gitConfig.Branch))

	// Create repository
	repo, err := NewRepository(gitConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	// Clone or open repository
	if err := repo.Clone(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	globalRepo = repo

	// Load initial configuration from Git
	if err := LoadConfigurationFromGit(); err != nil {
		return fmt.Errorf("failed to load initial configuration: %w", err)
	}

	// Create poller
	globalPoller = NewPoller(repo, gitConfig.PollInterval, nil)

	// Create webhook handler if secret is provided
	if gitConfig.WebhookSecret != "" {
		globalWebhook = NewWebhookHandler(globalPoller, gitConfig.WebhookSecret)
		logger.Info("Webhook handler initialized")
	}

	logger.Info("GitOps initialized successfully")
	return nil
}

// StartGitPoller starts the Git polling loop
func StartGitPoller(ctx context.Context) {
	if globalPoller == nil {
		logger.Error("Git poller not initialized")
		return
	}

	globalPoller.Start(ctx)
}

// GetWebhookHandler returns the global webhook handler
func GetWebhookHandler() *WebhookHandler {
	return globalWebhook
}

// LoadConfigurationFromGit loads configuration from the Git repository
func LoadConfigurationFromGit() error {
	if globalRepo == nil {
		return fmt.Errorf("git repository not initialized")
	}

	// Load servers from repository
	newConfig, err := LoadServersFromRepository(globalRepo.GetLocalPath())
	if err != nil {
		return err
	}

	// Preserve Git config from current configuration
	gitConfig := config.GetGitConfig()
	newConfig.Git = gitConfig

	// Update package-level configuration
	config.UpdateConfiguration(*newConfig)

	logger.Info("Configuration loaded from Git",
		zap.Int("servers", len(newConfig.Servers)))

	return nil
}

// TriggerManualSync triggers a manual sync (for testing or manual operations)
func TriggerManualSync() error {
	if globalPoller == nil {
		return fmt.Errorf("git poller not initialized")
	}

	return globalPoller.TriggerSync()
}
