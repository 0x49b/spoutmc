package v1

import (
	"net/http"
	"spoutmc/internal/webserver/api/v1/auth"
	"spoutmc/internal/webserver/api/v1/git"
	"spoutmc/internal/webserver/api/v1/host"
	"spoutmc/internal/webserver/api/v1/infrastructure"
	"spoutmc/internal/webserver/api/v1/player"
	"spoutmc/internal/webserver/api/v1/role"
	"spoutmc/internal/webserver/api/v1/server"
	"spoutmc/internal/webserver/api/v1/setup"
	"spoutmc/internal/webserver/api/v1/user"
	"spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
)

func RegisterV1(g *echo.Group) {
	v1 := g.Group("/v1")
	v1.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	// Public routes (no JWT required)
	auth.RegisterAuthRoutes(v1)
	setup.RegisterSetupRoutes(v1)
	player.RegisterPlayerChatIngestRoute(v1)

	// Protected routes (JWT required)
	protected := v1.Group("", middleware.JWT)
	auth.RegisterAuthVerifyRoute(protected)
	server.RegisterServerRoutes(protected)
	user.RegisterUserRoutes(protected)
	role.RegisterRoleRoutes(protected)
	player.RegisterPlayerRoutes(protected)
	host.RegisterHostRoutes(protected)
	git.RegisterGitRoutes(protected)
	infrastructure.RegisterInfrastructureRoutes(protected)
}
