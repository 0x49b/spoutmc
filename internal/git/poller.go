package git

import (
	"context"
	"spoutmc/internal/config"
	"time"

	"go.uber.org/zap"
)

// Poller manages periodic polling of a Git repository for changes
type Poller struct {
	repo     *Repository
	interval time.Duration
	onChange func()
}

// NewPoller creates a new Git poller
func NewPoller(repo *Repository, interval time.Duration, onChange func()) *Poller {
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &Poller{
		repo:     repo,
		interval: interval,
		onChange: onChange,
	}
}

// Start starts the polling loop
func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	logger.Info("Git poller started", zap.Duration("interval", p.interval))

	for {
		select {
		case <-ctx.Done():
			logger.Info("Git poller shutting down")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll performs a single poll cycle
func (p *Poller) poll(ctx context.Context) {
	logger.Debug("Polling Git repository for changes")

	// Pull latest changes
	hasChanges, err := p.repo.Pull()
	if err != nil {
		logger.Error("Failed to pull Git repository", zap.Error(err))
		return
	}

	if !hasChanges {
		logger.Debug("No changes detected in Git repository")
		return
	}

	logger.Info("Changes detected in Git repository, reloading configuration")

	// Load new configuration
	if err := p.reloadConfiguration(ctx); err != nil {
		logger.Error("Failed to reload configuration from Git", zap.Error(err))
		return
	}

	// Trigger onChange callback if provided
	if p.onChange != nil {
		p.onChange()
	}
}

// reloadConfiguration loads the new configuration from Git and applies changes
func (p *Poller) reloadConfiguration(ctx context.Context) error {
	// Get current configuration
	currentConfig := config.All()

	// Load servers from Git repository
	newServers, err := LoadServersFromRepository(p.repo.GetLocalPath())
	if err != nil {
		return err
	}

	// Preserve Git config, storage, and EULA from current configuration
	// These should always come from local config/spoutmc.yaml
	newConfig := *newServers
	newConfig.Git = currentConfig.Git
	newConfig.Storage = currentConfig.Storage
	newConfig.EULA = currentConfig.EULA

	// Update package-level configuration
	config.UpdateConfiguration(newConfig)

	// Apply configuration changes
	config.ApplyConfigChanges(ctx, currentConfig, newConfig)

	return nil
}

// TriggerSync manually triggers a sync (used by webhooks)
func (p *Poller) TriggerSync(ctx context.Context) error {
	logger.Info("Manual sync triggered via webhook")

	// Pull latest changes
	hasChanges, err := p.repo.Pull()
	if err != nil {
		return err
	}

	if !hasChanges {
		logger.Info("No changes to apply")
		return nil
	}

	// Reload configuration
	if err := p.reloadConfiguration(ctx); err != nil {
		return err
	}

	// Trigger onChange callback if provided
	if p.onChange != nil {
		p.onChange()
	}

	return nil
}
