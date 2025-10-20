package api

import (
	"spoutmc/internal/webserver/api/v1"

	"github.com/labstack/echo/v4"
)

func RegisterAPI(e *echo.Echo) {
	api := e.Group("/api")
	v1.RegisterV1(api)
}
