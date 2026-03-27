package git

import (
	"context"
	"spoutmc/internal/config"
	"spoutmc/internal/notifications"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Poller struct {
	repo     *Repository
	interval time.Duration
	onChange func()
	syncMu   sync.Mutex
}

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
func (p *Poller) poll(ctx context.Context) {
	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	logger.Debug("Polling Git repository for changes")
	MarkSyncStart(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage())

	hasChanges, err := p.repo.Pull()
	if err != nil {
		logger.Error("Failed to pull Git repository", zap.Error(err))
		_ = notifications.UpsertOpen(
			"gitops:pull-failed",
			"danger",
			"GitOps pull failed",
			err.Error(),
			"gitops",
		)
		MarkSyncError(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), err)
		return
	}

	if hasChanges {
		MarkChangeDetected()
		logger.Info("Changes detected in Git repository, reloading configuration")

		summary, err := p.reloadConfiguration(ctx)
		if err != nil {
			logger.Error("Failed to reload configuration from Git", zap.Error(err))
			MarkSyncError(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), err)
			return
		}
		MarkSyncSuccess(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), summary)
	} else {
		logger.Debug("No changes detected in Git repository")
		MarkSyncSuccess(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), SyncSummary{})
	}

	if p.onChange != nil {
		p.onChange()
	}
}
func (p *Poller) reloadConfiguration(ctx context.Context) (SyncSummary, error) {
	currentConfig := config.All()

	newServers, err := LoadServersFromRepository(p.repo.GetLocalPath())
	if err != nil {
		return SyncSummary{}, err
	}

	newConfig := *newServers
	newConfig.Git = currentConfig.Git
	newConfig.Storage = currentConfig.Storage
	newConfig.EULA = currentConfig.EULA

	changeSet := config.DiffServers(currentConfig.Servers, newConfig.Servers)

	config.UpdateConfiguration(newConfig)

	config.ApplyConfigChanges(ctx, currentConfig, newConfig)

	return SyncSummary{
		Added:   len(changeSet.Added),
		Updated: len(changeSet.Updated),
		Removed: len(changeSet.Removed),
	}, nil
}
func (p *Poller) TriggerSync(ctx context.Context) error {
	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	logger.Info("Manual sync triggered via webhook")
	MarkSyncStart(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage())

	hasChanges, err := p.repo.Pull()
	if err != nil {
		MarkSyncError(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), err)
		return err
	}

	summary := SyncSummary{}
	if !hasChanges {
		logger.Info("No repository changes detected")
		MarkSyncSuccess(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), summary)
	} else {
		MarkChangeDetected()
		summary, err = p.reloadConfiguration(ctx)
		if err != nil {
			MarkSyncError(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), err)
			return err
		}
		MarkSyncSuccess(p.repo.GetLastCommit(), p.repo.GetLastCommitMessage(), summary)
	}

	if p.onChange != nil {
		p.onChange()
	}

	return nil
}
