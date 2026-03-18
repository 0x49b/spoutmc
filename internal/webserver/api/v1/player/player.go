package player

import (
	"encoding/json"
	"net/http"
	playerpkg "spoutmc/internal/player"
	"spoutmc/internal/sse"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
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

func messagePlayer(c echo.Context) error {
	var cmd playerpkg.PlayerCommand
	if err := c.Bind(&cmd); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := bridgeClient.MessagePlayerWithMeta(c.Request().Context(), c.Param("name"), cmd.Message, cmd.Sender, cmd.Role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "message sent"})
}

func getPlayerChat(c echo.Context) error {
	messages, err := bridgeClient.GetPlayerChat(c.Request().Context(), c.Param("name"))
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, messages)
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
