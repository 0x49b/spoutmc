package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"spoutmc/internal/log"
	"spoutmc/internal/webserver/api"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"go.uber.org/zap"
)

var logger = log.GetLogger()

func Start() (*echo.Echo, error) {

	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	e.Pre(middleware.RemoveTrailingSlash())

	//e.Use(log.CreateZapLoggerMiddleware(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
	}))
	e.Use(middleware.Secure())
	//e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request", zap.String("URI", v.URI), zap.Int("status", v.Status), zap.String("method", v.Method), zap.Duration("latency", v.Latency))
			return nil
		},
	}))

	//swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Register API routes
	api.RegisterAPI(e)

	ln, err := net.Listen("tcp", ":3000")
	if err != nil {
		return nil, fmt.Errorf("❌ failed to bind to port: %w", err)
	}
	logger.Info("🤵🏻‍♂️ webserver started on http://localhost:3000")

	go func() {
		if err := e.Server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Fatal("❌ shutting down webserver", zap.Error(err))
		}
	}()

	err = writeRoutes(e)
	if err != nil {
		return nil, err
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("🤵🏻‍♂️ Error during webserver shutdown", zap.Error(err))
	} else {
		logger.Info("🤵🏻‍♂️ Webserver shutdown complete")
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
	err = os.WriteFile("routes.json", data, 0644)
	if err != nil {
		return err
	}
	return nil
}
