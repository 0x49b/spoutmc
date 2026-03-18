package player

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/minime/processor"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"go.uber.org/zap"
)

var (
	logger = log.GetLogger(log.ModuleAPI)

	joinedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\]:\s*([A-Za-z0-9_]{3,16}) joined the game`),
		regexp.MustCompile(`(?i)\b([A-Za-z0-9_]{3,16})\b.*\bjoined the game\b`),
	}
	leftPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\]:\s*([A-Za-z0-9_]{3,16}) left the game`),
		regexp.MustCompile(`(?i)\]:\s*([A-Za-z0-9_]{3,16}) lost connection`),
		regexp.MustCompile(`(?i)\]:\s*([A-Za-z0-9_]{3,16}) disconnected`),
	}
)

type PlayerState struct {
	Name            string  `json:"name"`
	AvatarDataURL   string  `json:"avatarDataUrl,omitempty"`
	LastLoggedInAt  *string `json:"lastLoggedInAt,omitempty"`
	LastLoggedOutAt *string `json:"lastLoggedOutAt,omitempty"`
	CurrentServer   string  `json:"currentServer,omitempty"`
	Banned          bool    `json:"banned"`
	BanReason       string  `json:"banReason,omitempty"`
	Status          string  `json:"status"`
}

type PlayerCommand struct {
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type playerRecord struct {
	Name          string
	AvatarDataURL string
	LastLoginAt   *time.Time
	LastLogoutAt  *time.Time
	CurrentServer string
	Banned        bool
	BanReason     string
}

type Tracker struct {
	mu              sync.RWMutex
	started         bool
	players         map[string]*playerRecord
	containerCursor map[string]time.Time
	avatarInFlight  map[string]bool
	subscribers     map[chan []PlayerState]struct{}
}

func NewTracker() *Tracker {
	return &Tracker{
		players:         make(map[string]*playerRecord),
		containerCursor: make(map[string]time.Time),
		avatarInFlight:  make(map[string]bool),
		subscribers:     make(map[chan []PlayerState]struct{}),
	}
}

var globalTracker = NewTracker()

func GetTracker() *Tracker {
	return globalTracker
}

func (t *Tracker) EnsureStarted() {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return
	}
	t.started = true
	t.mu.Unlock()

	go t.pollLoop(context.Background())
}

func (t *Tracker) List() []PlayerState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return snapshotLocked(t.players)
}

func (t *Tracker) Subscribe() (<-chan []PlayerState, func()) {
	ch := make(chan []PlayerState, 2)

	t.mu.Lock()
	t.subscribers[ch] = struct{}{}
	initial := snapshotLocked(t.players)
	t.mu.Unlock()

	ch <- initial

	unsubscribe := func() {
		t.mu.Lock()
		delete(t.subscribers, ch)
		close(ch)
		t.mu.Unlock()
	}
	return ch, unsubscribe
}

func (t *Tracker) MessagePlayer(ctx context.Context, playerName string, message string) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("message is required")
	}
	cmd := fmt.Sprintf("msg %s %s", playerName, message)
	return executePlayerCommand(ctx, cmd)
}

func (t *Tracker) KickPlayer(ctx context.Context, playerName string, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Kicked by admin"
	}
	cmd := fmt.Sprintf("kick %s %s", playerName, reason)
	return executePlayerCommand(ctx, cmd)
}

func (t *Tracker) BanPlayer(ctx context.Context, playerName string, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Banned by admin"
	}
	cmd := fmt.Sprintf("ban %s %s", playerName, reason)
	if err := executePlayerCommand(ctx, cmd); err != nil {
		return err
	}

	now := time.Now().UTC()
	t.mu.Lock()
	p := t.ensurePlayerLocked(playerName)
	p.Banned = true
	p.BanReason = reason
	p.CurrentServer = ""
	p.LastLogoutAt = &now
	snap := snapshotLocked(t.players)
	t.mu.Unlock()
	t.broadcastSnapshot(snap)

	return nil
}

func (t *Tracker) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Prime immediately so UI gets data quickly.
	t.pollOnce(ctx, true)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.pollOnce(ctx, false)
		}
	}
}

func (t *Tracker) pollOnce(ctx context.Context, bootstrap bool) {
	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		logger.Warn("Unable to load containers for players tracking", zap.Error(err))
		return
	}

	var snapshotsToSend []PlayerState
	changed := false

	for _, c := range containers {
		serverName := getServerNameFromContainer(c)

		t.mu.RLock()
		cursor, hasCursor := t.containerCursor[c.ID]
		t.mu.RUnlock()

		if !hasCursor {
			if bootstrap {
				cursor = time.Now().UTC().Add(-2 * time.Hour)
			} else {
				cursor = time.Now().UTC().Add(-30 * time.Second)
			}
		}

		lines, err := readContainerLogsSince(ctx, c.ID, cursor)
		if err != nil {
			logger.Debug("Unable to read logs for player tracking",
				zap.String("containerID", c.ID),
				zap.String("serverName", serverName),
				zap.Error(err))
			continue
		}

		now := time.Now().UTC()
		t.mu.Lock()
		t.containerCursor[c.ID] = now
		for _, line := range lines {
			if t.applyLogLineLocked(strings.TrimSpace(line), serverName, now) {
				changed = true
			}
		}
		if changed {
			snapshotsToSend = snapshotLocked(t.players)
		}
		t.mu.Unlock()
	}

	if changed {
		t.broadcastSnapshot(snapshotsToSend)
	}
}

func (t *Tracker) applyLogLineLocked(line string, serverName string, ts time.Time) bool {
	if line == "" {
		return false
	}

	for _, re := range joinedPatterns {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			playerName := strings.TrimSpace(matches[1])
			p := t.ensurePlayerLocked(playerName)
			p.LastLoginAt = &ts
			p.CurrentServer = serverName
			if p.Banned {
				p.Banned = false
				p.BanReason = ""
			}
			t.startAvatarResolveLocked(playerName)
			return true
		}
	}

	for _, re := range leftPatterns {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			playerName := strings.TrimSpace(matches[1])
			p := t.ensurePlayerLocked(playerName)
			p.LastLogoutAt = &ts
			p.CurrentServer = ""
			t.startAvatarResolveLocked(playerName)
			return true
		}
	}

	return false
}

func (t *Tracker) startAvatarResolveLocked(playerName string) {
	p := t.ensurePlayerLocked(playerName)
	if p.AvatarDataURL != "" {
		return
	}
	if t.avatarInFlight[playerName] {
		return
	}
	t.avatarInFlight[playerName] = true

	go func(name string) {
		avatar, err := generateAvatarFromMinecraftSkin(name)
		if err != nil {
			logger.Debug("Unable to generate player avatar", zap.String("player", name), zap.Error(err))
		}

		t.mu.Lock()
		delete(t.avatarInFlight, name)
		if avatar != "" {
			t.ensurePlayerLocked(name).AvatarDataURL = avatar
			snap := snapshotLocked(t.players)
			t.mu.Unlock()
			t.broadcastSnapshot(snap)
			return
		}
		t.mu.Unlock()
	}(playerName)
}

func (t *Tracker) ensurePlayerLocked(playerName string) *playerRecord {
	if p, ok := t.players[playerName]; ok {
		return p
	}
	p := &playerRecord{Name: playerName}
	t.players[playerName] = p
	return p
}

func (t *Tracker) broadcastSnapshot(snapshot []PlayerState) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for sub := range t.subscribers {
		select {
		case sub <- snapshot:
		default:
		}
	}
}

func snapshotLocked(players map[string]*playerRecord) []PlayerState {
	result := make([]PlayerState, 0, len(players))
	for _, p := range players {
		lastIn := formatTimePointer(p.LastLoginAt)
		lastOut := formatTimePointer(p.LastLogoutAt)

		status := "offline"
		if p.Banned {
			status = "banned"
		} else if p.CurrentServer != "" {
			status = "online"
		}

		result = append(result, PlayerState{
			Name:            p.Name,
			AvatarDataURL:   p.AvatarDataURL,
			LastLoggedInAt:  lastIn,
			LastLoggedOutAt: lastOut,
			CurrentServer:   p.CurrentServer,
			Banned:          p.Banned,
			BanReason:       p.BanReason,
			Status:          status,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}

func formatTimePointer(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.UTC().Format(time.RFC3339)
	return &formatted
}

func getServerNameFromContainer(c container.Summary) string {
	if value, ok := c.Labels["io.spout.servername"]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	if len(c.Names) > 0 {
		return strings.TrimPrefix(c.Names[0], "/")
	}
	return c.ID[:12]
}

func executePlayerCommand(ctx context.Context, command string) error {
	// Prefer proxy because player routing is centralized there.
	if proxy, err := docker.GetProxyContainer(ctx); err == nil {
		if err := docker.ExecuteCommand(ctx, proxy.ID, command); err == nil {
			return nil
		}
	}

	// Fallback to running servers.
	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		return err
	}

	var lastErr error
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		if err := docker.ExecuteCommand(ctx, c.ID, command); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no running server accepted command")
}

func readContainerLogsSince(ctx context.Context, containerID string, since time.Time) ([]string, error) {
	cli := docker.GetDockerClient()

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Timestamps: false,
		Details:    false,
		Since:      fmt.Sprintf("%d", since.Unix()),
	}

	reader, err := cli.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if inspect.Config != nil && inspect.Config.Tty {
		return readRawLines(reader), nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, reader); err != nil && err != io.EOF {
		return nil, err
	}

	lines := append(readRawLines(&stdout), readRawLines(&stderr)...)
	return lines, nil
}

func readRawLines(r io.Reader) []string {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lines := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func generateAvatarFromMinecraftSkin(playerName string) (string, error) {
	textureURL, err := fetchPlayerSkinTexture(playerName)
	if err != nil {
		return "", err
	}
	if textureURL == "" {
		return "", fmt.Errorf("empty skin texture for player %s", playerName)
	}

	img, err := processor.ProcessSkin(textureURL, true, true, 72)
	if err != nil {
		return "", err
	}
	encoded, err := processor.EncodeToBase64(img)
	if err != nil {
		return "", err
	}

	return "data:image/png;base64," + encoded, nil
}

func fetchPlayerSkinTexture(playerName string) (string, error) {
	url := fmt.Sprintf("https://playerdb.co/api/player/minecraft/%s", playerName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("avatar lookup failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Player struct {
				SkinTexture string `json:"skin_texture"`
			} `json:"player"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if !payload.Success {
		return "", fmt.Errorf("avatar lookup did not succeed")
	}
	return payload.Data.Player.SkinTexture, nil
}
