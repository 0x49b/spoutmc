package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"spoutmc/internal/log"
	"spoutmc/internal/notifications"

	"go.uber.org/zap"
)

const (
	StateIdle        = "idle"
	StateChecking    = "checking"
	StateAvailable   = "available"
	StateDownloading = "downloading"
	StateInstalling  = "installing"
	StateRestarting  = "restarting"
	StateError       = "error"

	DefaultCheckInterval = 24 * time.Hour
	DefaultInitialDelay  = 10 * time.Second

	updateAvailableNotificationKey = "spoutmc.update.available"
)

var logger = log.GetLogger(log.ModuleMain)

type Status struct {
	Configured         bool       `json:"configured"`
	CurrentVersion     string     `json:"currentVersion"`
	LatestVersion      string     `json:"latestVersion,omitempty"`
	ReleaseURL         string     `json:"releaseUrl,omitempty"`
	ReleaseNotes       string     `json:"releaseNotes,omitempty"`
	UpdateAvailable    bool       `json:"updateAvailable"`
	MigrationRequired  bool       `json:"migrationRequired"`
	State              string     `json:"state"`
	LastCheckedAt      *time.Time `json:"lastCheckedAt,omitempty"`
	LastError          string     `json:"lastError,omitempty"`
	LastInstalledAt    *time.Time `json:"lastInstalledAt,omitempty"`
	LastBackupPath     string     `json:"lastBackupPath,omitempty"`
	CurrentAssetName   string     `json:"currentAssetName,omitempty"`
	CheckIntervalHours int64      `json:"checkIntervalHours"`
}

type ticker interface {
	Chan() <-chan time.Time
	Stop()
}

type realTicker struct {
	t *time.Ticker
}

func (r *realTicker) Chan() <-chan time.Time {
	return r.t.C
}

func (r *realTicker) Stop() {
	r.t.Stop()
}

type Manager struct {
	mu sync.RWMutex

	repo             string
	githubToken      string
	githubAPIBaseURL string
	currentVersion   string
	httpClient       *http.Client
	checkInterval    time.Duration
	initialDelay     time.Duration

	tickerFactory func(time.Duration) ticker
	timerFactory  func(time.Duration) *time.Timer

	status Status
}

var (
	globalMu      sync.Mutex
	globalManager *Manager
)

func Init(currentVersion string) *Manager {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalManager != nil {
		globalManager.mu.Lock()
		globalManager.currentVersion = normalizeVersion(currentVersion)
		globalManager.status.CurrentVersion = normalizeVersion(currentVersion)
		globalManager.mu.Unlock()
		return globalManager
	}

	interval := DefaultCheckInterval
	if raw := strings.TrimSpace(os.Getenv("SPOUTMC_UPDATE_CHECK_INTERVAL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			interval = parsed
		}
	}

	repo := strings.TrimSpace(os.Getenv("SPOUTMC_UPDATE_REPO"))
	mgr := &Manager{
		repo:             repo,
		githubToken:      strings.TrimSpace(os.Getenv("SPOUTMC_GITHUB_TOKEN")),
		githubAPIBaseURL: strings.TrimSpace(os.Getenv("SPOUTMC_GITHUB_API_BASE")),
		currentVersion:   normalizeVersion(currentVersion),
		httpClient:       newHTTPClient(),
		checkInterval:    interval,
		initialDelay:     DefaultInitialDelay,
		tickerFactory: func(d time.Duration) ticker {
			return &realTicker{t: time.NewTicker(d)}
		},
		timerFactory: time.NewTimer,
	}
	mgr.status = Status{
		Configured:         repo != "",
		CurrentVersion:     mgr.currentVersion,
		State:              StateIdle,
		MigrationRequired:  false,
		CurrentAssetName:   assetNameForRuntime(runtime.GOOS, runtime.GOARCH),
		CheckIntervalHours: int64(interval.Hours()),
	}
	globalManager = mgr
	return mgr
}

func Get() *Manager {
	globalMu.Lock()
	defer globalMu.Unlock()
	return globalManager
}

func StartScheduler(ctx context.Context, currentVersion string) {
	mgr := Init(currentVersion)
	go mgr.runScheduler(ctx)
}

func (m *Manager) runScheduler(ctx context.Context) {
	if !m.isConfigured() {
		logger.Info("Update scheduler disabled: SPOUTMC_UPDATE_REPO not configured")
		return
	}

	timer := m.timerFactory(m.initialDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		m.checkAndPublish(context.Background(), "startup")
	}

	tk := m.tickerFactory(m.checkInterval)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.Chan():
			m.checkAndPublish(context.Background(), "scheduled")
		}
	}
}

func (m *Manager) isConfigured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.repo != ""
}

func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) CheckNow(ctx context.Context) (Status, error) {
	err := m.checkAndPublish(ctx, "manual")
	return m.GetStatus(), err
}

func (m *Manager) StartUpdate() error {
	if !m.isConfigured() {
		return fmt.Errorf("update repository is not configured (set SPOUTMC_UPDATE_REPO)")
	}

	m.mu.Lock()
	switch m.status.State {
	case StateDownloading, StateInstalling, StateRestarting:
		m.mu.Unlock()
		return fmt.Errorf("update already in progress")
	}
	m.status.State = StateChecking
	m.status.LastError = ""
	m.mu.Unlock()

	go m.performUpdate()
	return nil
}

func (m *Manager) checkAndPublish(ctx context.Context, source string) error {
	if !m.isConfigured() {
		m.setError("update repository is not configured")
		return fmt.Errorf("update repository is not configured")
	}

	m.setState(StateChecking)
	release, err := m.fetchLatestRelease(ctx)
	if err != nil {
		m.setError(err.Error())
		return err
	}

	available := isNewerVersion(m.currentVersion, release.Version)
	now := time.Now().UTC()

	m.mu.Lock()
	m.status.LatestVersion = release.Version
	m.status.ReleaseURL = release.ReleaseURL
	m.status.ReleaseNotes = release.ReleaseNotes
	m.status.UpdateAvailable = available
	m.status.LastCheckedAt = &now
	m.status.LastError = ""
	if available {
		m.status.State = StateAvailable
	} else {
		m.status.State = StateIdle
	}
	m.mu.Unlock()

	logger.Info("Checked latest SpoutMC release",
		zap.String("source", source),
		zap.String("current_version", m.currentVersion),
		zap.String("latest_version", release.Version),
		zap.Bool("update_available", available),
	)

	if available {
		msg := fmt.Sprintf("SpoutMC %s is available (current: %s).", release.Version, m.currentVersion)
		if err := notifications.UpsertOpen(
			updateAvailableNotificationKey,
			"info",
			"SpoutMC update available",
			msg,
			"update",
		); err != nil {
			logger.Warn("Failed to upsert update notification", zap.Error(err))
		}
	}

	return nil
}

func (m *Manager) performUpdate() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	release, err := m.fetchLatestRelease(ctx)
	if err != nil {
		m.setError(err.Error())
		return
	}
	if !isNewerVersion(m.currentVersion, release.Version) {
		m.setState(StateIdle)
		return
	}

	assetName := assetNameForRuntime(runtime.GOOS, runtime.GOARCH)
	checksumName := assetName + ".sha256"
	assetURL, ok := release.Assets[assetName]
	if !ok {
		m.setError(fmt.Sprintf("release asset %q not found", assetName))
		return
	}
	checksumURL, ok := release.Assets[checksumName]
	if !ok {
		m.setError(fmt.Sprintf("checksum asset %q not found", checksumName))
		return
	}

	m.setState(StateDownloading)
	tmpDir, err := os.MkdirTemp("", "spoutmc-update-*")
	if err != nil {
		m.setError(err.Error())
		return
	}
	defer os.RemoveAll(tmpDir)

	binaryPath := filepath.Join(tmpDir, assetName)
	checksumPath := filepath.Join(tmpDir, checksumName)
	if err := m.downloadToFile(ctx, assetURL, binaryPath); err != nil {
		m.setError(fmt.Sprintf("download binary: %v", err))
		return
	}
	if err := m.downloadToFile(ctx, checksumURL, checksumPath); err != nil {
		m.setError(fmt.Sprintf("download checksum: %v", err))
		return
	}
	if err := verifyChecksum(binaryPath, checksumPath, assetName); err != nil {
		m.setError(fmt.Sprintf("checksum validation failed: %v", err))
		return
	}

	m.setState(StateInstalling)
	backupPath, err := replaceExecutable(binaryPath)
	if err != nil {
		m.setError(fmt.Sprintf("install update: %v", err))
		return
	}

	now := time.Now().UTC()
	m.mu.Lock()
	m.currentVersion = release.Version
	m.status.CurrentVersion = release.Version
	m.status.LatestVersion = release.Version
	m.status.UpdateAvailable = false
	m.status.LastInstalledAt = &now
	m.status.LastBackupPath = backupPath
	m.status.LastError = ""
	m.status.State = StateRestarting
	m.mu.Unlock()

	logger.Info("SpoutMC update installed, requesting restart",
		zap.String("new_version", release.Version),
		zap.String("backup_path", backupPath),
	)

	if err := signalSelfTerminate(); err != nil {
		m.setError(fmt.Sprintf("request restart: %v", err))
	}
}

func (m *Manager) setState(state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.State = state
}

func (m *Manager) setError(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.State = StateError
	m.status.LastError = message
}

func (m *Manager) downloadToFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "spoutmc-updater")
	if m.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+m.githubToken)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

func assetNameForRuntime(goos, goarch string) string {
	name := fmt.Sprintf("spoutmc-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func verifyChecksum(binaryPath, checksumPath, expectedFileName string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	expected, err := parseChecksumFile(string(data), expectedFileName)
	if err != nil {
		return err
	}

	file, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch (expected %s, got %s)", expected, actual)
	}
	return nil
}

func parseChecksumFile(content, expectedFileName string) (string, error) {
	lines := strings.Split(content, "\n")
	type candidate struct {
		name string
		sum  string
	}

	candidates := make([]candidate, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		sum := fields[0]
		name := ""
		if len(fields) > 1 {
			name = strings.TrimPrefix(fields[1], "*")
		}
		candidates = append(candidates, candidate{name: name, sum: sum})
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("checksum file is empty")
	}

	for _, c := range candidates {
		if c.name == expectedFileName {
			return c.sum, nil
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return len(candidates[i].name) > len(candidates[j].name)
	})
	return candidates[0].sum, nil
}

func replaceExecutable(newBinaryPath string) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	resolvedExecPath, err := filepath.EvalSymlinks(execPath)
	if err == nil {
		execPath = resolvedExecPath
	}

	currentStat, err := os.Stat(execPath)
	if err != nil {
		return "", err
	}

	if err := os.Chmod(newBinaryPath, currentStat.Mode()); err != nil {
		return "", err
	}

	backupPath := execPath + ".bak"
	_ = os.Remove(backupPath)
	if err := os.Rename(execPath, backupPath); err != nil {
		return "", err
	}
	if err := os.Rename(newBinaryPath, execPath); err != nil {
		_ = os.Rename(backupPath, execPath)
		return "", err
	}
	return backupPath, nil
}

func signalSelfTerminate() error {
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		return proc.Signal(os.Interrupt)
	}
	return proc.Signal(syscall.SIGTERM)
}
