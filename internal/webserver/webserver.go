package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"spoutmc/internal/log"
	"spoutmc/internal/webserver/api"
	"spoutmc/internal/webserver/static"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleWebserver)

// WriteRoutesOnStart controls whether docs/routes.json is written at startup.
// Default is true for local development and can be overridden at build-time.
// Example: -X spoutmc/internal/webserver.WriteRoutesOnStart=false
var WriteRoutesOnStart = "true"

// serveEmbeddedFiles serves embedded frontend files with SPA fallback
func serveEmbeddedFiles(fsys fs.FS) echo.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))

	return func(c echo.Context) error {
		path := c.Request().URL.Path

		// Try to open the requested file
		f, err := fsys.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			// File exists, serve it
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}

		// File doesn't exist, serve index.html for SPA routing
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

	//e.Use(log.CreateZapLoggerMiddleware(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.Use(middleware.Secure())
	//e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request", zap.String("URI", v.URI), zap.Int("status", v.Status), zap.String("method", v.Method), zap.Duration("latency", v.Latency))
			return nil
		},
	}))

	// Serve embedded frontend
	if distFS, err := static.GetDistFS(); err == nil {
		logger.Info("🎨 Serving embedded frontend from binary")

		// Serve static assets (JS, CSS, images)
		e.GET("/assets/*", echo.WrapHandler(
			http.FileServer(http.FS(distFS)),
		))

		// Catch-all route for SPA routing (serves index.html for non-existent paths)
		e.GET("/*", serveEmbeddedFiles(distFS))
	} else {
		logger.Warn("⚠️ Frontend assets not embedded, running in API-only mode", zap.Error(err))
	}

	// Register API routes (these take precedence due to Echo's router priority)
	api.RegisterAPI(e)

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

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("Error during webserver shutdown", zap.Error(err))
	} else {
		logger.Info("Webserver shutdown complete")
	}

	return e, nil
}

func Shutdown(e *echo.Echo) error {
	err := e.Shutdown(context.Background())
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
	// Runtime override for local debugging:
	// SPOUTMC_WRITE_ROUTES=true|false
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
