package webserver

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"os"
	"spoutmc/backend/config"
	"spoutmc/backend/dbcontext"
	"spoutmc/backend/log"
	"spoutmc/web"
	"time"
)

var logger = log.New()

func Start() *echo.Echo {
	conf := config.New(os.Getenv("PORT"), os.Getenv("ENV"))

	e := echo.New()
	e.HideBanner = true
	app := conf.Bootstrap()

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

	registerHandler(e, app.Db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	return e
}

func registerHandler(r *echo.Echo, db *dbcontext.DB) {
	web.RegisterHandlers(r)
}

func ShutdownServer(e *echo.Echo) error {
	err := e.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}
