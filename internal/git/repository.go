package git

import (
	"fmt"
	"os"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"go.uber.org/zap"
)

var logger = log.GetLogger()

// Repository manages a Git repository for configuration
type Repository struct {
	config     *models.GitConfig
	repo       *git.Repository
	lastCommit string
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

// Clone clones the repository or opens it if it already exists
func (r *Repository) Clone() error {
	logger.Info("Cloning Git repository", zap.String("repository", r.config.Repository))

	// Check if directory exists
	if _, err := os.Stat(r.config.LocalPath); err == nil {
		// Directory exists, try to open it
		logger.Info("Local repository exists, opening", zap.String("path", r.config.LocalPath))
		return r.open()
	}

	// Create parent directory if needed
	if err := os.MkdirAll(r.config.LocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prepare clone options
	cloneOpts := &git.CloneOptions{
		URL:           r.buildAuthURL(),
		ReferenceName: plumbing.NewBranchReferenceName(r.config.Branch),
		SingleBranch:  true,
		Depth:         1, // Shallow clone for performance
		Progress:      nil,
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

	logger.Info("Repository cloned successfully", zap.String("commit", r.lastCommit[:7]))
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

	// Prepare pull options
	pullOpts := &git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(r.config.Branch),
		SingleBranch:  true,
		Force:         true, // Always prefer remote changes
		Progress:      nil,
	}

	// Only add auth if token is provided
	if r.config.Token != "" {
		pullOpts.Auth = &http.BasicAuth{
			Username: "token", // Can be anything for PAT
			Password: r.config.Token,
		}
	}

	// Pull changes
	err = worktree.Pull(pullOpts)
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			logger.Debug("Repository already up to date")
			return false, nil
		}
		return false, fmt.Errorf("failed to pull: %w", err)
	}

	// Get new commit hash
	oldCommit := r.lastCommit
	if err := r.updateCommitHash(); err != nil {
		return false, err
	}

	hasChanges := oldCommit != r.lastCommit
	if hasChanges {
		logger.Info("Repository updated",
			zap.String("old_commit", oldCommit[:7]),
			zap.String("new_commit", r.lastCommit[:7]))
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

// updateCommitHash updates the last commit hash
func (r *Repository) updateCommitHash() error {
	ref, err := r.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	r.lastCommit = ref.Hash().String()
	return nil
}

// buildAuthURL builds the repository URL with authentication
func (r *Repository) buildAuthURL() string {
	if r.config.Token == "" {
		return r.config.Repository
	}

	// Add token to URL for HTTPS authentication
	// Format: https://token@github.com/user/repo.git
	if strings.HasPrefix(r.config.Repository, "https://") {
		url := strings.TrimPrefix(r.config.Repository, "https://")
		return fmt.Sprintf("https://%s@%s", r.config.Token, url)
	}

	return r.config.Repository
}
