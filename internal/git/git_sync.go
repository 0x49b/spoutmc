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

func InitializeGitOps() error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		InitializeStatus(false)
		return fmt.Errorf("git config is nil")
	}
	InitializeStatus(true)

	logger.Info("Initializing GitOps",
		zap.String("repository", gitConfig.Repository),
		zap.String("branch", gitConfig.Branch))

	repo, err := NewRepository(gitConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	if err := repo.Clone(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	globalRepo = repo

	if err := LoadConfigurationFromGit(); err != nil {
		MarkSyncError("", "", err)
		return fmt.Errorf("failed to load initial configuration: %w", err)
	}

	globalPoller = NewPoller(repo, gitConfig.PollInterval, nil)

	globalWebhook = NewWebhookHandler(globalPoller, gitConfig.WebhookSecret)
	logger.Info("Webhook handler initialized")

	logger.Info("GitOps initialized successfully")
	MarkSyncSuccess(repo.GetLastCommit(), repo.GetLastCommitMessage(), SyncSummary{})
	return nil
}
func StartGitPoller(ctx context.Context) {
	if globalPoller == nil {
		logger.Error("Git poller not initialized, try to start")
		StartGitPoller(ctx)
		return
	}

	globalPoller.Start(ctx)
}
func GetWebhookHandler() *WebhookHandler {
	return globalWebhook
}
func GetLocalRepoPath() string {
	if globalRepo == nil {
		return ""
	}
	return globalRepo.GetLocalPath()
}
func LoadConfigurationFromGit() error {
	if globalRepo == nil {
		return fmt.Errorf("git repository not initialized")
	}

	newConfig, err := LoadServersFromRepository(globalRepo.GetLocalPath())
	if err != nil {
		return err
	}

	currentConfig := config.All()
	newConfig.Git = currentConfig.Git
	newConfig.Storage = currentConfig.Storage
	newConfig.EULA = currentConfig.EULA

	config.UpdateConfiguration(*newConfig)

	logger.Info("Configuration loaded from Git",
		zap.Int("servers", len(newConfig.Servers)))

	return nil
}
func TriggerManualSync(ctx context.Context) error {
	if globalPoller == nil {
		if !config.IsGitOpsEnabled() {
			return fmt.Errorf("git poller not initialized")
		}
		if err := InitializeGitOps(); err != nil {
			return fmt.Errorf("git poller not initialized and on-demand initialization failed: %w", err)
		}
	}

	if globalPoller == nil {
		return fmt.Errorf("git poller not initialized")
	}

	return globalPoller.TriggerSync(ctx)
}
func GetSyncStatus() GitOpsStatus {
	return GetStatus()
}
