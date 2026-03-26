package player

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"spoutmc/internal/config"
	"strings"
	"time"

	"github.com/google/uuid"
	playerpkg "spoutmc/internal/player"

	"github.com/labstack/echo/v4"
)

// RegisterPlayerChatIngestRoute exposes plugin → API persistence for incoming MC replies (shared secret).
func RegisterPlayerChatIngestRoute(g *echo.Group) {
	g.POST("/player/chat-ingest", ingestPlayerChatReply)
}

func ingestPlayerChatReply(c echo.Context) error {
	secret, err := resolveVelocityForwardingSecret()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "chat ingest is not configured"})
	}
	got := strings.TrimSpace(c.Request().Header.Get("X-Spout-Chat-Ingest"))

	configured := secret != ""
	authorized := false
	if configured {
		authorized = len(got) == len(secret) && subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1
	}

	if !configured {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "chat ingest is not configured"})
	}
	if !authorized {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		PlayerName  string `json:"playerName"`
		PlayerUUID  string `json:"playerUuid"`
		StaffUserID uint   `json:"staffUserId"`
		Message     string `json:"message"`
		Timestamp   string `json:"timestamp,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	msg := strings.TrimSpace(req.Message)
	playerName := strings.TrimSpace(req.PlayerName)
	valid := msg != "" && req.StaffUserID != 0 && playerName != ""

	if !valid {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "playerName, staffUserId, and message are required"})
	}

	at := time.Now().UTC()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			at = t.UTC()
		}
	}

	sender := playerName
	var playerUUIDParsed *uuid.UUID
	if strings.TrimSpace(req.PlayerUUID) != "" {
		if u, err := uuid.Parse(req.PlayerUUID); err == nil {
			playerUUIDParsed = &u
		}
	}

	convID, err := playerpkg.ResolveOpenConversationForIngest(playerUUIDParsed, req.StaffUserID, playerName)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}

	if err := playerpkg.AppendSupportChatMessage(convID, playerName, playerUUIDParsed, req.StaffUserID, "incoming", sender, "", msg, at); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to persist message"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "ingested"})
}

func resolveVelocityForwardingSecret() (string, error) {
	cfg := config.All()
	if cfg.Storage == nil || strings.TrimSpace(cfg.Storage.DataPath) == "" {
		return "", fmt.Errorf("storage.data_path is not configured")
	}

	proxyName := ""
	for i := range cfg.Servers {
		if cfg.Servers[i].Proxy {
			proxyName = strings.TrimSpace(cfg.Servers[i].Name)
			break
		}
	}
	if proxyName == "" {
		return "", fmt.Errorf("proxy server is not configured")
	}

	secretPath := filepath.Join(cfg.Storage.DataPath, proxyName, "server", "forwarding.secret")
	secretBytes, err := os.ReadFile(secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read forwarding secret: %w", err)
	}

	secret := strings.TrimSpace(string(secretBytes))
	if secret == "" {
		return "", fmt.Errorf("forwarding secret is empty")
	}

	return secret, nil
}
