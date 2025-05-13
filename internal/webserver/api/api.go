package api

import (
	"github.com/labstack/echo/v4"
	"spoutmc/internal/webserver/api/v1"
)

func RegisterAPI(e *echo.Echo) {
	api := e.Group("/api")
	v1.RegisterV1(api)
}
