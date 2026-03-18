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
	playerGroup.POST("/:name/message", messagePlayer)
	playerGroup.POST("/:name/kick", kickPlayer)
	playerGroup.POST("/:name/ban", banPlayer)
}

func listPlayers(c echo.Context) error {
	tracker := playerpkg.GetTracker()
	tracker.EnsureStarted()
	return c.JSON(http.StatusOK, tracker.List())
}

func streamPlayers(c echo.Context) error {
	tracker := playerpkg.GetTracker()
	tracker.EnsureStarted()

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	events, unsubscribe := tracker.Subscribe()
	defer unsubscribe()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case players, ok := <-events:
			if !ok {
				return nil
			}

			data, err := json.Marshal(players)
			if err != nil {
				return err
			}

			id, _ := shortid.Generate()
			event := sse.Event{
				ID:        []byte(id),
				Data:      data,
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

	tracker := playerpkg.GetTracker()
	tracker.EnsureStarted()
	if err := tracker.MessagePlayer(c.Request().Context(), c.Param("name"), cmd.Message); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "message sent"})
}

func kickPlayer(c echo.Context) error {
	var cmd playerpkg.PlayerCommand
	if err := c.Bind(&cmd); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	tracker := playerpkg.GetTracker()
	tracker.EnsureStarted()
	if err := tracker.KickPlayer(c.Request().Context(), c.Param("name"), cmd.Reason); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "player kicked"})
}

func banPlayer(c echo.Context) error {
	var cmd playerpkg.PlayerCommand
	if err := c.Bind(&cmd); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	tracker := playerpkg.GetTracker()
	tracker.EnsureStarted()
	if err := tracker.BanPlayer(c.Request().Context(), c.Param("name"), cmd.Reason); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "player banned"})
}
