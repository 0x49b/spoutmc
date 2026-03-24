package player

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"spoutmc/internal/access"
	playerpkg "spoutmc/internal/player"
	"spoutmc/internal/sse"
	"spoutmc/internal/storage"
	"spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
	"gorm.io/gorm"
)

func RegisterPlayerRoutes(g *echo.Group) {
	playerGroup := g.Group("/player")

	playerGroup.GET("", listPlayers)
	playerGroup.GET("/stream", streamPlayers)
	playerGroup.GET("/:name/chat", getPlayerChat)
	playerGroup.POST("/:name/message", messagePlayer)
	playerGroup.POST("/:name/kick", kickPlayer)
	playerGroup.POST("/:name/ban", banPlayer)
	playerGroup.POST("/:name/unban", unbanPlayer)
}

var bridgeClient = playerpkg.NewBridgeClientFromEnv()

func listPlayers(c echo.Context) error {
	players, err := bridgeClient.ListPlayers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, players)
}

func streamPlayers(c echo.Context) error {
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastPayload string

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-ticker.C:
			players, err := bridgeClient.ListPlayers(c.Request().Context())
			if err != nil {
				continue
			}
			data, err := json.Marshal(players)
			if err != nil {
				return err
			}
			payload := string(data)
			if payload == lastPayload {
				continue
			}
			lastPayload = payload

			id, _ := shortid.Generate()
			event := sse.Event{
				ID:        []byte(id),
				Data:      []byte(payload),
				Timestamp: time.Now().Unix(),
			}
			if err := event.MarshalTo(w); err != nil {
				return err
			}
			w.Flush()
		}
	}
}

type playerMessageBody struct {
	Message string `json:"message"`
}

func messagePlayer(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	var body playerMessageBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	msg := strings.TrimSpace(body.Message)
	if msg == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "message is required"})
	}

	if storage.GetDB() == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	user, err := playerpkg.LoadUserWithRoles(cl.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	sender := playerpkg.StaffChatSenderLabel(user)
	roleLabel := playerpkg.PrimaryRoleDisplay(user.Roles)
	playerName := c.Param("name")

	if err := bridgeClient.MessagePlayerWithMeta(c.Request().Context(), playerName, msg, sender, roleLabel, cl.UserID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := playerpkg.AppendSupportChatMessage(playerName, cl.UserID, "outgoing", sender, roleLabel, msg, time.Now().UTC()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to persist chat"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "message sent"})
}

func getPlayerChat(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	if storage.GetDB() == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerName := c.Param("name")
	scope := strings.ToLower(strings.TrimSpace(c.QueryParam("scope")))

	var messages []playerpkg.PlayerChatMessage
	var err error
	if scope == "all" {
		if !chatArchiveAllowed(cl) {
			return echo.NewHTTPError(http.StatusForbidden, "archive scope requires admin or manager")
		}
		messages, err = playerpkg.ListSupportChatAllForPlayer(playerName)
	} else {
		messages, err = playerpkg.ListSupportChatForStaff(playerName, cl.UserID)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, messages)
}

func chatArchiveAllowed(cl *access.Claims) bool {
	for _, r := range cl.Roles {
		switch strings.ToLower(strings.TrimSpace(r)) {
		case "admin", "manager":
			return true
		}
	}
	return false
}

func kickPlayer(c echo.Context) error {
	var cmd playerpkg.PlayerCommand
	if err := c.Bind(&cmd); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := bridgeClient.KickPlayer(c.Request().Context(), c.Param("name"), cmd.Reason); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "player kicked"})
}

func banPlayer(c echo.Context) error {
	var cmd playerpkg.PlayerCommand
	if err := c.Bind(&cmd); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := bridgeClient.BanPlayer(c.Request().Context(), c.Param("name"), cmd.Reason); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "player banned"})
}

func unbanPlayer(c echo.Context) error {
	if err := bridgeClient.UnbanPlayer(c.Request().Context(), c.Param("name")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusAccepted, map[string]string{"status": "player unbanned"})
}
