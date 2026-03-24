package ws

import (
	"net/http"
	"os"
	realtimews "spoutmc/internal/realtime/ws"
	"spoutmc/internal/webserver/guards"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

var defaultWSService = realtimews.NewService()

type WSService interface {
	HandleConnection(ctx echo.Context, conn *websocket.Conn, containerID string, userID uint) error
}

type serviceAdapter struct {
	service *realtimews.Service
}

func (a serviceAdapter) HandleConnection(ctx echo.Context, conn *websocket.Conn, containerID string, userID uint) error {
	return a.service.HandleConnection(ctx.Request().Context(), conn, containerID, userID)
}

func RegisterWSRoutes(g *echo.Group) {
	grp := g.Group("/ws")
	grp.GET("/server/:id", handleServerSocket)
}

func RegisterWSRoutesWithService(g *echo.Group, service *realtimews.Service) {
	if service != nil {
		defaultWSService = service
	}
	RegisterWSRoutes(g)
}

func handleServerSocket(c echo.Context) error {
	if !isServerWSFeatureEnabled() {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "WebSocket realtime is disabled",
		})
	}

	claims, err := guards.RequireClaims(c)
	if err != nil {
		return err
	}

	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	adapter := serviceAdapter{service: defaultWSService}
	websocket.Handler(func(conn *websocket.Conn) {
		_ = adapter.HandleConnection(c, conn, containerID, claims.UserID)
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func isServerWSFeatureEnabled() bool {
	flagValue := strings.TrimSpace(strings.ToLower(os.Getenv("ENABLE_SERVER_WS")))
	if flagValue == "" {
		return true
	}
	switch flagValue {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
