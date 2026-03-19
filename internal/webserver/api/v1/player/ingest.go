package player

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"time"

	playerpkg "spoutmc/internal/player"

	"github.com/labstack/echo/v4"
)

// RegisterPlayerChatIngestRoute exposes plugin → API persistence for incoming MC replies (shared secret).
func RegisterPlayerChatIngestRoute(g *echo.Group) {
	g.POST("/player/chat-ingest", ingestPlayerChatReply)
}

func ingestPlayerChatReply(c echo.Context) error {
	secret := strings.TrimSpace(os.Getenv("SPOUT_PLAYER_CHAT_INGEST_SECRET"))
	if secret == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "chat ingest is not configured"})
	}
	got := strings.TrimSpace(c.Request().Header.Get("X-Spout-Chat-Ingest"))
	if len(got) != len(secret) || subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		PlayerName  string `json:"playerName"`
		StaffUserID uint   `json:"staffUserId"`
		Message     string `json:"message"`
		Timestamp   string `json:"timestamp,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	msg := strings.TrimSpace(req.Message)
	playerName := strings.TrimSpace(req.PlayerName)
	if msg == "" || req.StaffUserID == 0 || playerName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "playerName, staffUserId, and message are required"})
	}

	at := time.Now().UTC()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			at = t.UTC()
		}
	}

	sender := playerName
	if err := playerpkg.AppendSupportChatMessage(playerName, req.StaffUserID, "incoming", sender, "", msg, at); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to persist message"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "ingested"})
}
