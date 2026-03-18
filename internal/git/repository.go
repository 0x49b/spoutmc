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

// Repository manages a Git repository for configuration
type Repository struct {
	config            *models.GitConfig
	repo              *git.Repository
	lastCommit        string
	lastCommitMessage string
}

// NewRepository creates a new repository manager
func NewRepository(config *models.GitConfig) (*Repository, error) {
	if config == nil {
		return nil, fmt.Errorf("git config is nil")
	}

	// Expand environment variables in token
	config.Token = os.ExpandEnv(config.Token)
	config.WebhookSecret = os.ExpandEnv(config.WebhookSecret)

	// Expand local path
	localPath := os.ExpandEnv(config.LocalPath)
	if localPath == "" {
		localPath = "/tmp/spoutmc-git"
	}
	config.LocalPath = localPath

	// Set default branch
	if config.Branch == "" {
		config.Branch = "main"
	}

	return &Repository{
		config: config,
	}, nil
}

// Clone performs a fresh clone by clearing the local repository path first.
func (r *Repository) Clone() error {
	logger.Info("Cloning Git repository", zap.String("repository", r.config.Repository))

	// Always start from a clean repository directory to avoid stale/deleted files
	// or local drift impacting startup behavior.
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

	// Create parent directory if needed
	if err := os.MkdirAll(r.config.LocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prepare clone options
	cloneOpts := &git.CloneOptions{
		URL:           r.config.Repository,
		ReferenceName: plumbing.NewBranchReferenceName(r.config.Branch),
		SingleBranch:  true,
		Depth:         1, // Shallow clone for performance
		Progress:      nil,
	}

	// Only add auth if token is provided
	if r.config.Token != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: "token", // Can be anything for PAT
			Password: r.config.Token,
		}
	}

	// Clone the repository
	repo, err := git.PlainClone(r.config.LocalPath, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	r.repo = repo

	// Get current commit hash
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

// open opens an existing repository
func (r *Repository) open() error {
	repo, err := git.PlainOpen(r.config.LocalPath)
	if err != nil {
		// If opening fails, remove and re-clone
		logger.Warn("Failed to open existing repository, removing and re-cloning", zap.Error(err))
		if err := os.RemoveAll(r.config.LocalPath); err != nil {
			return fmt.Errorf("failed to remove corrupted repository: %w", err)
		}
		return r.Clone()
	}

	r.repo = repo

	// Get current commit hash
	if err := r.updateCommitHash(); err != nil {
		return err
	}

	return nil
}

// Pull pulls the latest changes from the remote repository
func (r *Repository) Pull() (bool, error) {
	if r.repo == nil {
		return false, fmt.Errorf("repository not initialized")
	}

	logger.Debug("Pulling latest changes from Git repository")

	// Get working tree
	worktree, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Store old commit hash before fetching
	oldCommit := r.lastCommit

	// Prepare fetch options
	fetchOpts := &git.FetchOptions{
		RemoteName: "origin",
		Progress:   nil,
		Force:      true, // Force fetch to overwrite local refs
	}

	// Only add auth if token is provided
	if r.config.Token != "" {
		fetchOpts.Auth = &http.BasicAuth{
			Username: "token", // Can be anything for PAT
			Password: r.config.Token,
		}
	}

	// Fetch latest changes from remote
	err = r.repo.Fetch(fetchOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return false, fmt.Errorf("failed to fetch: %w", err)
	}

	// Get the remote branch reference
	remoteBranch := fmt.Sprintf("refs/remotes/origin/%s", r.config.Branch)
	remoteRef, err := r.repo.Reference(plumbing.ReferenceName(remoteBranch), true)
	if err != nil {
		return false, fmt.Errorf("failed to get remote reference: %w", err)
	}

	// Get current HEAD
	headRef, err := r.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Check if there are changes
	hasChanges := headRef.Hash() != remoteRef.Hash()

	if hasChanges {
		logger.Info("Changes detected, forcing reset to remote branch",
			zap.String("old_commit", shortCommit(headRef.Hash().String())),
			zap.String("new_commit", shortCommit(remoteRef.Hash().String())))

		// Force reset to remote branch (this discards all local changes including untracked files)
		err = worktree.Reset(&git.ResetOptions{
			Commit: remoteRef.Hash(),
			Mode:   git.HardReset,
		})
		if err != nil {
			return false, fmt.Errorf("failed to reset to remote: %w", err)
		}

		// Clean untracked files and directories
		err = worktree.Clean(&git.CleanOptions{
			Dir: true, // Remove untracked directories too
		})
		if err != nil {
			// Log but don't fail - clean is best effort
			logger.Warn("Failed to clean untracked files", zap.Error(err))
		}

		// Update commit hash
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

// GetLocalPath returns the local path of the repository
func (r *Repository) GetLocalPath() string {
	return r.config.LocalPath
}

// GetLastCommit returns the last commit hash
func (r *Repository) GetLastCommit() string {
	return r.lastCommit
}

// GetLastCommitMessage returns the latest commit's subject/message.
func (r *Repository) GetLastCommitMessage() string {
	return r.lastCommitMessage
}

// updateCommitHash updates the last commit hash
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

// CommitAndPush commits all changes and pushes to the remote repository
func (r *Repository) CommitAndPush(message string) error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	logger.Info("Committing and pushing changes", zap.String("message", message))

	// Get working tree
	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	if err := worktree.AddWithOptions(&git.AddOptions{
		All: true,
	}); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit changes
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

	// Prepare push options
	pushOpts := &git.PushOptions{
		RemoteName: "origin",
		Progress:   nil,
	}

	// Only add auth if token is provided
	if r.config.Token != "" {
		pushOpts.Auth = &http.BasicAuth{
			Username: "token",
			Password: r.config.Token,
		}
	}

	// Push changes
	if err := r.repo.Push(pushOpts); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	// Update commit hash
	if err := r.updateCommitHash(); err != nil {
		return err
	}

	logger.Info("Changes pushed successfully", zap.String("commit", shortCommit(r.lastCommit)))
	return nil
}

// CommitAndPushChanges is a convenience function that commits and pushes changes to the git repository
// It opens the repository at the given path, commits all changes, and pushes them
func CommitAndPushChanges(repoPath, message string) error {
	// Open the repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get working tree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	if err := worktree.AddWithOptions(&git.AddOptions{
		All: true,
	}); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit changes
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

	// Get git config to check for token
	gitConfig, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get git config: %w", err)
	}

	// Prepare push options
	pushOpts := &git.PushOptions{
		RemoteName: "origin",
		Progress:   nil,
	}

	// Try to get token from environment or git config
	// The token should have been embedded in the remote URL during clone
	token := os.Getenv("GIT_TOKEN")
	if token != "" {
		pushOpts.Auth = &http.BasicAuth{
			Username: "token",
			Password: token,
		}
	} else if remoteConfig, exists := gitConfig.Remotes["origin"]; exists && len(remoteConfig.URLs) > 0 {
		// Check if URL contains embedded token
		url := remoteConfig.URLs[0]
		if strings.Contains(url, "@") {
			// Token is embedded, no need to add auth
			logger.Debug("Using embedded token from remote URL")
		}
	}

	// Push changes
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
