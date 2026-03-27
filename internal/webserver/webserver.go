package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"spoutmc/internal/infrastructureapp"
	"spoutmc/internal/log"
	realtimews "spoutmc/internal/realtime/ws"
	"spoutmc/internal/serverapp"
	"spoutmc/internal/webserver/api"
	"spoutmc/internal/webserver/api/v1"
	"spoutmc/internal/webserver/static"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleWebserver)

var WriteRoutesOnStart = "true"

func serveEmbeddedFiles(fsys fs.FS) echo.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))

	return func(c echo.Context) error {
		path := c.Request().URL.Path

		f, err := fsys.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}

		c.Request().URL.Path = "/"
		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

func Start() (*echo.Echo, error) {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	e.Pre(middleware.RemoveTrailingSlash())

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.Use(middleware.Secure())
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      100,
			Burst:     100,
			ExpiresIn: 3 * time.Minute,
		}),
	}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Debug("request", zap.String("URI", v.URI), zap.Int("status", v.Status), zap.String("method", v.Method), zap.Duration("latency", v.Latency))
			return nil
		},
	}))

	if distFS, err := static.GetDistFS(); err == nil {
		logger.Info("🎨 Serving embedded frontend from binary")

		e.GET("/assets/*", echo.WrapHandler(
			http.FileServer(http.FS(distFS)),
		))

		e.GET("/*", serveEmbeddedFiles(distFS))
	} else {
		logger.Warn("⚠️ Frontend assets not embedded, running in API-only mode", zap.Error(err))
	}

	modules := v1.Modules{
		ServerService: serverapp.NewService(),
		InfraService:  infrastructureapp.NewService(),
		WSService:     realtimews.NewService(),
	}
	api.RegisterAPIWithModules(e, modules)

	ln, err := net.Listen("tcp", ":3000")
	if err != nil {
		return nil, fmt.Errorf("❌ failed to bind to port: %w", err)
	}
	logger.Info("webserver started on http://localhost:3000")

	go func() {
		if err := e.Server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Fatal("❌ shutting down webserver", zap.Error(err))
		}
	}()

	if shouldWriteRoutes() {
		err = writeRoutes(e)
		if err != nil {
			return nil, err
		}
	}

	return e, nil
}

func Shutdown(ctx context.Context, e *echo.Echo) error {
	if e == nil {
		return nil
	}

	err := e.Shutdown(ctx)
	if err != nil {
		return err
	}
	return nil
}

func writeRoutes(e *echo.Echo) error {
	data, err := json.MarshalIndent(e.Routes(), "", "  ")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(cwd, "docs", "routes.json"), data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func shouldWriteRoutes() bool {
	if envValue := os.Getenv("SPOUTMC_WRITE_ROUTES"); envValue != "" {
		parsed, err := strconv.ParseBool(strings.TrimSpace(envValue))
		if err == nil {
			return parsed
		}
		logger.Warn("Invalid SPOUTMC_WRITE_ROUTES value, using build default",
			zap.String("value", envValue),
			zap.Error(err))
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(WriteRoutesOnStart))
	if err != nil {
		logger.Warn("Invalid WriteRoutesOnStart build value, defaulting to true",
			zap.String("value", WriteRoutesOnStart),
			zap.Error(err))
		return true
	}

	return parsed
}
