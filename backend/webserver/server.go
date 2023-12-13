package webserver

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"spoutmc/backend/config"
	"spoutmc/backend/dbcontext"
	"spoutmc/backend/log"
	v1Container "spoutmc/backend/webserver/api/v1"
	"spoutmc/web"
	"time"
)

var logger = log.New()

func Start() *echo.Echo {

	logger.Info("Starting Webserver")

	conf := config.New(os.Getenv("3000"), os.Getenv("ENV"))

	e := echo.New()
	e.HideBanner = true
	app := conf.Bootstrap()

	e.Pre(middleware.RemoveTrailingSlash())

	e.Use(middleware.CORS())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{Timeout: 10 * time.Second}))
	e.Use(middleware.Secure())
	e.Use(middleware.Recover())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request", zap.String("URI", v.URI), zap.Int("status", v.Status), zap.String("method", v.Method), zap.Duration("latency", v.Latency))
			return nil
		},
	}))

	// Frontend Handler
	registerHandler(e, app.Db)

	apiGroup := e.Group("/api")
	v1 := apiGroup.Group("/v1")
	v1Container.RegisterContainerAPI(v1)

	go func() {
		logger.Info("Webserver started")
		if err := e.Start(":3000"); err != nil && err != http.ErrServerClosed {
			logger.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal("", zap.Error(err))
	}

	return e
}

func registerHandler(r *echo.Echo, db *dbcontext.DB) {
	web.RegisterHandlers(r)
}

func Shutdown(e *echo.Echo) error {
	err := e.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}
