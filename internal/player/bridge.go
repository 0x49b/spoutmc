package player

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"spoutmc/internal/docker"
)

type PlayerState struct {
	Name            string   `json:"name"`
	UUID            string   `json:"uuid"`
	AvatarDataURL   string   `json:"avatarDataUrl,omitempty"`
	LastLoggedInAt  *string  `json:"lastLoggedInAt,omitempty"`
	LastLoggedOutAt *string  `json:"lastLoggedOutAt,omitempty"`
	CurrentServer   string   `json:"currentServer,omitempty"`
	Banned          bool     `json:"banned"`
	BanReason       string   `json:"banReason,omitempty"`
	Status          string   `json:"status"`
	ClientBrand     string   `json:"clientBrand,omitempty"`
	ClientMods      []string `json:"clientMods,omitempty"`
}

type PlayerCommand struct {
	Message         string `json:"message,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Sender          string `json:"sender,omitempty"`
	Role            string `json:"role,omitempty"`
	StaffUserID     uint   `json:"staffUserId"` // must be present for velocity bridge (do not omitempty)
	NewConversation bool   `json:"newConversation,omitempty"`
}

type PlayerChatMessage struct {
	Direction      string `json:"direction"`
	Player         string `json:"player"`
	StaffUserID    uint   `json:"staffUserId"`
	ConversationID uint   `json:"conversationId,omitempty"`
	Sender         string `json:"sender,omitempty"`
	Role           string `json:"role,omitempty"`
	Message        string `json:"message"`
	Timestamp      string `json:"timestamp"`
}

type bridgePlayer struct {
	Name            string   `json:"name"`
	UUID            string   `json:"uuid"`
	AvatarURL       string   `json:"avatarUrl,omitempty"`
	AvatarDataURL   string   `json:"avatarDataUrl,omitempty"`
	LastLoggedInAt  *string  `json:"lastLoggedInAt,omitempty"`
	LastLoggedOutAt *string  `json:"lastLoggedOutAt,omitempty"`
	CurrentServer   string   `json:"currentServer,omitempty"`
	Banned          bool     `json:"banned"`
	BanReason       string   `json:"banReason,omitempty"`
	Status          string   `json:"status"`
	ClientBrand     string   `json:"clientBrand,omitempty"`
	ClientMods      []string `json:"clientMods,omitempty"`
}
type BridgeClient struct {
	envURL      string // SPOUT_PLAYERS_BRIDGE_URL when set; otherwise we resolve via Docker
	resolvedURL string // cached http://<container-ip>:29132
	mu          sync.Mutex
	token       string
	httpClient  *http.Client
}

func NewBridgeClientFromEnv() *BridgeClient {
	return &BridgeClient{
		envURL: strings.TrimSpace(os.Getenv("SPOUT_PLAYERS_BRIDGE_URL")),
		token:  strings.TrimSpace(os.Getenv("SPOUT_PLAYERS_BRIDGE_TOKEN")),
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *BridgeClient) baseURL(ctx context.Context) string {
	if v := strings.TrimSpace(c.envURL); v != "" {
		return strings.TrimRight(v, "/")
	}

	c.mu.Lock()
	cached := c.resolvedURL
	c.mu.Unlock()
	if cached != "" {
		return cached
	}

	return "http://127.0.0.1:" + docker.DefaultPlayersBridgePort
}

func (c *BridgeClient) clearResolvedURL() {
	c.mu.Lock()
	c.resolvedURL = ""
	c.mu.Unlock()
}

func (c *BridgeClient) invalidateResolvedIfConnErr(err error) {
	if err == nil || strings.TrimSpace(c.envURL) != "" {
		return
	}
	s := err.Error()
	if strings.Contains(s, "refused") || strings.Contains(s, "timeout") ||
		strings.Contains(s, "no such host") || strings.Contains(s, "network is unreachable") {
		c.clearResolvedURL()
	}
}

func (c *BridgeClient) ListPlayers(ctx context.Context) ([]PlayerState, error) {
	base := c.baseURL(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/players", nil)
	if err != nil {
		return nil, err
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.invalidateResolvedIfConnErr(err)
		return nil, fmt.Errorf("failed to call players bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("players bridge list failed (%d): %s", resp.StatusCode, string(body))
	}

	var payload []bridgePlayer
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid players bridge payload: %w", err)
	}

	result := make([]PlayerState, 0, len(payload))
	for _, p := range payload {
		avatar := normalizeAvatarURL(p)

		status := normalizeStatus(p.Status, p.Banned, p.CurrentServer)

		result = append(result, PlayerState{
			Name:            p.Name,
			UUID:            p.UUID,
			AvatarDataURL:   avatar,
			LastLoggedInAt:  p.LastLoggedInAt,
			LastLoggedOutAt: p.LastLoggedOutAt,
			CurrentServer:   p.CurrentServer,
			Banned:          p.Banned,
			BanReason:       p.BanReason,
			Status:          status,
			ClientBrand:     p.ClientBrand,
			ClientMods:      p.ClientMods,
		})
	}

	return result, nil
}

func normalizeAvatarURL(p bridgePlayer) string {
	avatar := strings.TrimSpace(p.AvatarDataURL)
	if avatar == "" {
		avatar = strings.TrimSpace(p.AvatarURL)
	}

	if strings.Contains(strings.ToLower(avatar), "crafatar.com/avatars/") {
		if p.Name != "" {
			return fmt.Sprintf("https://mc-heads.net/avatar/%s/72", url.QueryEscape(p.Name))
		}
	}

	if avatar != "" {
		return avatar
	}

	if p.Name != "" {
		return fmt.Sprintf("https://mc-heads.net/avatar/%s/72", url.QueryEscape(p.Name))
	}

	return ""
}

func (c *BridgeClient) MessagePlayer(ctx context.Context, playerName string, message string) error {
	body, err := json.Marshal(PlayerCommand{Message: message})
	if err != nil {
		return err
	}
	return c.postPlayerAction(ctx, playerName, "message", body)
}

func (c *BridgeClient) MessagePlayerWithMeta(ctx context.Context, playerName, message, sender, role string, staffUserID uint, newConversation bool) error {
	body, err := json.Marshal(PlayerCommand{
		Message:         message,
		Sender:          sender,
		Role:            role,
		StaffUserID:     staffUserID,
		NewConversation: newConversation,
	})
	if err != nil {
		return err
	}
	return c.postPlayerAction(ctx, playerName, "message", body)
}

func (c *BridgeClient) KickPlayer(ctx context.Context, playerName string, reason string) error {
	body, err := json.Marshal(PlayerCommand{Reason: reason})
	if err != nil {
		return err
	}
	return c.postPlayerAction(ctx, playerName, "kick", body)
}

func (c *BridgeClient) BanPlayer(ctx context.Context, playerName string, reason string) error {
	body, err := json.Marshal(PlayerCommand{Reason: reason})
	if err != nil {
		return err
	}
	return c.postPlayerAction(ctx, playerName, "ban", body)
}

func (c *BridgeClient) UnbanPlayer(ctx context.Context, playerName string) error {
	return c.postPlayerAction(ctx, playerName, "unban", []byte("{}"))
}

func (c *BridgeClient) postPlayerAction(ctx context.Context, playerName string, action string, body []byte) error {
	base := c.baseURL(ctx)
	encodedName := url.PathEscape(playerName)
	endpoint := fmt.Sprintf("%s/players/%s/%s", base, encodedName, action)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.invalidateResolvedIfConnErr(err)
		return fmt.Errorf("failed to call players bridge action: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("players bridge action failed (%d): %s", resp.StatusCode, string(payload))
	}
	return nil
}

func (c *BridgeClient) GetPlayerChat(ctx context.Context, playerName string, staffUserID uint) ([]PlayerChatMessage, error) {
	base := c.baseURL(ctx)
	encodedName := url.PathEscape(playerName)
	endpoint := fmt.Sprintf("%s/players/%s/chat?staffUserId=%d", base, encodedName, staffUserID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.invalidateResolvedIfConnErr(err)
		return nil, fmt.Errorf("failed to call players bridge chat endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("players bridge chat failed (%d): %s", resp.StatusCode, string(payload))
	}

	var messages []PlayerChatMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("invalid players bridge chat payload: %w", err)
	}
	return messages, nil
}

func (c *BridgeClient) applyAuth(req *http.Request) {
	if c.token == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
}

func normalizeStatus(status string, banned bool, currentServer string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "banned" || status == "online" || status == "offline" {
		return status
	}
	if banned {
		return "banned"
	}
	if strings.TrimSpace(currentServer) != "" {
		return "online"
	}
	return "offline"
}
