package v1

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func RegisterV1(g *echo.Group) {
	v1 := g.Group("/v1")
	v1.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	RegisterServerRoutes(v1)
}
