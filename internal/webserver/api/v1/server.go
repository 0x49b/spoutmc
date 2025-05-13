package v1

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// RegisterServerRoutes registers routes related to the /server endpoint
func RegisterServerRoutes(g *echo.Group) {
	g.GET("/server", func(c echo.Context) error {
		return c.String(http.StatusOK, "server")
	})
}
