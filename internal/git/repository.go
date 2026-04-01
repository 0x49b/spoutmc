package git

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleGit)

func resolveEnvTemplate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	// Backward-compatible parser for legacy template format:
	// ${SPOUTMC_GIT_TOKEN | "fallback" }
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "${"), "}"))
		if strings.Contains(inner, "|") {
			parts := strings.SplitN(inner, "|", 2)
			envKey := strings.TrimSpace(parts[0])
			fallbackRaw := strings.TrimSpace(parts[1])
			if envKey != "" {
				if envValue, ok := os.LookupEnv(envKey); ok && envValue != "" {
					return envValue
				}
			}
			return strings.Trim(strings.TrimSpace(fallbackRaw), "\"")
		}
	}

	return os.ExpandEnv(trimmed)
}

type Repository struct {
	config            *models.GitConfig
	repo              *git.Repository
	lastCommit        string
	lastCommitMessage string
}

func NewRepository(config *models.GitConfig) (*Repository, error) {
	if config == nil {
		return nil, fmt.Errorf("git config is nil")
	}

	config.Token = resolveEnvTemplate(config.Token)
	config.WebhookSecret = resolveEnvTemplate(config.WebhookSecret)

	localPath := os.ExpandEnv(config.LocalPath)
	if localPath == "" {
		localPath = "/tmp/spoutmc-git"
	}
	config.LocalPath = localPath

	if config.Branch == "" {
		config.Branch = "main"
	}

	return &Repository{
		config: config,
	}, nil
}
func (r *Repository) Clone() error {
	logger.Info("Cloning Git repository", zap.String("repository", r.config.Repository))

	if _, err := os.Stat(r.config.LocalPath); err == nil {
		if err := validateSafeDeletePath(r.config.LocalPath); err != nil {
			return fmt.Errorf("unsafe git local path for cleanup: %w", err)
		}

		logger.Info("Removing existing local Git repository before fresh clone",
			zap.String("path", r.config.LocalPath))
		if err := os.RemoveAll(r.config.LocalPath); err != nil {
			return fmt.Errorf("failed to remove existing local repository: %w", err)
		}
	}

	if err := os.MkdirAll(r.config.LocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:           r.config.Repository,
		ReferenceName: plumbing.NewBranchReferenceName(r.config.Branch),
		SingleBranch:  true,
		Depth:         1,
		Progress:      nil,
	}

	if r.config.Token != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: "token",
			Password: r.config.Token,
		}
	}

	repo, err := git.PlainClone(r.config.LocalPath, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	r.repo = repo

	if err := r.updateCommitHash(); err != nil {
		return err
	}

	logger.Info("Repository cloned successfully", zap.String("commit", shortCommit(r.lastCommit)))
	return nil
}

func validateSafeDeletePath(pathValue string) error {
	cleanPath := filepath.Clean(pathValue)
	if cleanPath == "." || cleanPath == string(filepath.Separator) {
		return fmt.Errorf("refusing to delete path %q", cleanPath)
	}

	volume := filepath.VolumeName(cleanPath)
	if volume != "" {
		withoutVolume := strings.TrimPrefix(cleanPath, volume)
		withoutVolume = strings.Trim(withoutVolume, `/\`)
		if withoutVolume == "" {
			return fmt.Errorf("refusing to delete volume root %q", cleanPath)
		}
	}

	return nil
}

func (r *Repository) open() error {
	repo, err := git.PlainOpen(r.config.LocalPath)
	if err != nil {
		logger.Warn("Failed to open existing repository, removing and re-cloning", zap.Error(err))
		if err := os.RemoveAll(r.config.LocalPath); err != nil {
			return fmt.Errorf("failed to remove corrupted repository: %w", err)
		}
		return r.Clone()
	}

	r.repo = repo

	if err := r.updateCommitHash(); err != nil {
		return err
	}

	return nil
}
func (r *Repository) Pull() (bool, error) {
	if r.repo == nil {
		return false, fmt.Errorf("repository not initialized")
	}

	logger.Debug("Pulling latest changes from Git repository")

	worktree, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	oldCommit := r.lastCommit

	fetchOpts := &git.FetchOptions{
		RemoteName: "origin",
		Progress:   nil,
		Force:      true,
	}

	if r.config.Token != "" {
		fetchOpts.Auth = &http.BasicAuth{
			Username: "token",
			Password: r.config.Token,
		}
	}

	err = r.repo.Fetch(fetchOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return false, fmt.Errorf("failed to fetch: %w", err)
	}

	remoteBranch := fmt.Sprintf("refs/remotes/origin/%s", r.config.Branch)
	remoteRef, err := r.repo.Reference(plumbing.ReferenceName(remoteBranch), true)
	if err != nil {
		return false, fmt.Errorf("failed to get remote reference: %w", err)
	}

	headRef, err := r.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}

	hasChanges := headRef.Hash() != remoteRef.Hash()

	if hasChanges {
		logger.Info("Changes detected, forcing reset to remote branch",
			zap.String("old_commit", shortCommit(headRef.Hash().String())),
			zap.String("new_commit", shortCommit(remoteRef.Hash().String())))

		err = worktree.Reset(&git.ResetOptions{
			Commit: remoteRef.Hash(),
			Mode:   git.HardReset,
		})
		if err != nil {
			return false, fmt.Errorf("failed to reset to remote: %w", err)
		}

		err = worktree.Clean(&git.CleanOptions{
			Dir: true,
		})
		if err != nil {
			logger.Warn("Failed to clean untracked files", zap.Error(err))
		}

		if err := r.updateCommitHash(); err != nil {
			return false, err
		}

		logger.Info("Repository updated successfully",
			zap.String("old_commit", shortCommit(oldCommit)),
			zap.String("new_commit", shortCommit(r.lastCommit)))
	} else {
		logger.Debug("Repository already up to date")
	}

	return hasChanges, nil
}

func (r *Repository) GetLocalPath() string {
	return r.config.LocalPath
}

func (r *Repository) GetLastCommit() string {
	return r.lastCommit
}

func (r *Repository) GetLastCommitMessage() string {
	return r.lastCommitMessage
}

func (r *Repository) updateCommitHash() error {
	ref, err := r.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	r.lastCommit = ref.Hash().String()

	commitObj, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return fmt.Errorf("failed to get commit object: %w", err)
	}
	r.lastCommitMessage = normalizeCommitMessage(commitObj.Message)
	return nil
}
func (r *Repository) CommitAndPush(message string) error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	logger.Info("Committing and pushing changes", zap.String("message", message))

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := worktree.AddWithOptions(&git.AddOptions{
		All: true,
	}); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	commit, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "SpoutMC",
			Email: "spoutmc@noreply.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	logger.Info("Changes committed", zap.String("commit", shortCommit(commit.String())))

	pushOpts := &git.PushOptions{
		RemoteName: "origin",
		Progress:   nil,
	}

	if r.config.Token != "" {
		pushOpts.Auth = &http.BasicAuth{
			Username: "token",
			Password: r.config.Token,
		}
	}

	if err := r.repo.Push(pushOpts); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	if err := r.updateCommitHash(); err != nil {
		return err
	}

	logger.Info("Changes pushed successfully", zap.String("commit", shortCommit(r.lastCommit)))
	return nil
}
func CommitAndPushChanges(repoPath, message string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := worktree.AddWithOptions(&git.AddOptions{
		All: true,
	}); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	commit, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "SpoutMC",
			Email: "spoutmc@noreply.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	logger.Info("Changes committed", zap.String("commit", shortCommit(commit.String())))

	gitConfig, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get git config: %w", err)
	}

	pushOpts := &git.PushOptions{
		RemoteName: "origin",
		Progress:   nil,
	}

	token := os.Getenv("GIT_TOKEN")
	if token != "" {
		pushOpts.Auth = &http.BasicAuth{
			Username: "token",
			Password: token,
		}
	} else if remoteConfig, exists := gitConfig.Remotes["origin"]; exists && len(remoteConfig.URLs) > 0 {
		url := remoteConfig.URLs[0]
		if strings.Contains(url, "@") {
			logger.Debug("Using embedded token from remote URL")
		}
	}

	if err := repo.Push(pushOpts); err != nil {
		if err == git.NoErrAlreadyUpToDate {
			logger.Debug("Repository already up to date")
			return nil
		}
		return fmt.Errorf("failed to push: %w", err)
	}

	logger.Info("Changes pushed successfully")
	return nil
}

func shortCommit(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}

func normalizeCommitMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	return strings.TrimSpace(lines[0])
}
