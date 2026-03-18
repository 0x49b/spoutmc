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
	"time"
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

type bridgePlayer struct {
	Name            string  `json:"name"`
	AvatarURL       string  `json:"avatarUrl,omitempty"`
	AvatarDataURL   string  `json:"avatarDataUrl,omitempty"`
	LastLoggedInAt  *string `json:"lastLoggedInAt,omitempty"`
	LastLoggedOutAt *string `json:"lastLoggedOutAt,omitempty"`
	CurrentServer   string  `json:"currentServer,omitempty"`
	Banned          bool    `json:"banned"`
	BanReason       string  `json:"banReason,omitempty"`
	Status          string  `json:"status"`
}

type BridgeClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewBridgeClientFromEnv() *BridgeClient {
	baseURL := strings.TrimSpace(os.Getenv("SPOUT_PLAYERS_BRIDGE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:29132"
	}

	token := strings.TrimSpace(os.Getenv("SPOUT_PLAYERS_BRIDGE_TOKEN"))

	return &BridgeClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *BridgeClient) ListPlayers(ctx context.Context) ([]PlayerState, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/players", nil)
	if err != nil {
		return nil, err
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
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
			AvatarDataURL:   avatar,
			LastLoggedInAt:  p.LastLoggedInAt,
			LastLoggedOutAt: p.LastLoggedOutAt,
			CurrentServer:   p.CurrentServer,
			Banned:          p.Banned,
			BanReason:       p.BanReason,
			Status:          status,
		})
	}

	return result, nil
}

func normalizeAvatarURL(p bridgePlayer) string {
	avatar := strings.TrimSpace(p.AvatarDataURL)
	if avatar == "" {
		avatar = strings.TrimSpace(p.AvatarURL)
	}

	// Crafatar can occasionally fail with 52x; use mc-heads as a reliable fallback.
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
	encodedName := url.PathEscape(playerName)
	endpoint := fmt.Sprintf("%s/players/%s/%s", c.baseURL, encodedName, action)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call players bridge action: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("players bridge action failed (%d): %s", resp.StatusCode, string(payload))
	}
	return nil
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
