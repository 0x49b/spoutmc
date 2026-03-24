package api

import (
	"spoutmc/internal/webserver/api/v1"

	"github.com/labstack/echo/v4"
)

func RegisterAPI(e *echo.Echo) {
	RegisterAPIWithModules(e, v1.Modules{})
}

func RegisterAPIWithModules(e *echo.Echo, modules v1.Modules) {
	api := e.Group("/api")
	v1.RegisterV1WithModules(api, modules)
}
