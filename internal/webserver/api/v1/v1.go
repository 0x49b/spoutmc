package v1

import (
	"net/http"
	"spoutmc/internal/webserver/api/v1/host"
	"spoutmc/internal/webserver/api/v1/server"
	"spoutmc/internal/webserver/api/v1/user"

	"github.com/labstack/echo/v4"
)

func RegisterV1(g *echo.Group) {
	v1 := g.Group("/v1")
	v1.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	server.RegisterServerRoutes(v1)
	user.RegisterUserRoutes(v1)
	host.RegisterHostRoutes(v1)
}
