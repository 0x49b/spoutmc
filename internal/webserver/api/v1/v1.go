package v1

import (
	"net/http"
	"spoutmc/internal/infrastructureapp"
	realtimews "spoutmc/internal/realtime/ws"
	"spoutmc/internal/serverapp"
	"spoutmc/internal/webserver/api/v1/auth"
	"spoutmc/internal/webserver/api/v1/git"
	"spoutmc/internal/webserver/api/v1/host"
	"spoutmc/internal/webserver/api/v1/infrastructure"
	"spoutmc/internal/webserver/api/v1/minime"
	"spoutmc/internal/webserver/api/v1/notification"
	"spoutmc/internal/webserver/api/v1/permission"
	"spoutmc/internal/webserver/api/v1/player"
	"spoutmc/internal/webserver/api/v1/plugin"
	"spoutmc/internal/webserver/api/v1/role"
	"spoutmc/internal/webserver/api/v1/server"
	"spoutmc/internal/webserver/api/v1/setup"
	"spoutmc/internal/webserver/api/v1/user"
	wsapi "spoutmc/internal/webserver/api/v1/ws"
	"spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
)

type Modules struct {
	ServerService *serverapp.Service
	InfraService  *infrastructureapp.Service
	WSService     *realtimews.Service
}

func RegisterV1(g *echo.Group) {
	RegisterV1WithModules(g, Modules{})
}

func RegisterV1WithModules(g *echo.Group, modules Modules) {
	v1 := g.Group("/v1")
	v1.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	player.RegisterPlayerChatIngestRoute(v1)

	auth.RegisterAuthRoutes(v1)
	setup.RegisterSetupRoutes(v1)
	minime.RegisterMinimeRoutes(v1)
	git.RegisterGitWebhookRoute(v1)

	protected := v1.Group("", middleware.JWT)
	serverService := modules.ServerService
	if serverService == nil {
		serverService = serverapp.NewService()
	}
	infraService := modules.InfraService
	if infraService == nil {
		infraService = infrastructureapp.NewService()
	}
	wsService := modules.WSService
	if wsService == nil {
		wsService = realtimews.NewService()
	}

	auth.RegisterAuthVerifyRoute(protected)
	server.RegisterServerRoutesWithService(protected, serverService)
	user.RegisterUserRoutes(protected)
	role.RegisterRoleRoutes(protected)
	permission.RegisterPermissionRoutes(protected)
	notification.RegisterNotificationRoutes(protected)
	player.RegisterPlayerRoutes(protected)
	host.RegisterHostRoutes(protected)
	git.RegisterGitRoutes(protected)
	infrastructure.RegisterInfrastructureRoutesWithService(protected, infraService)
	plugin.RegisterPluginRoutes(protected)
	wsapi.RegisterWSRoutesWithService(protected, wsService)
}
