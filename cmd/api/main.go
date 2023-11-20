package main

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"os/signal"
	"spoutmc/internal/config"
	"spoutmc/pkg/dbcontext"
	"spoutmc/pkg/log"
	"spoutmc/web"
	"time"
)

func main() {

	conf := config.New(os.Getenv("PORT"), os.Getenv("ENV"))
	l := log.New()
	e := echo.New()
	e.HideBanner = true
	app := conf.Bootstrap()

	e.Use(middleware.CORS())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{Timeout: 10 * time.Second}))
	e.Use(middleware.Secure())
	e.Use(middleware.Recover())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method} ${uri} ${status} ${latency_human} ${error}\n",
	}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // 20 request/sec rate limit

	registerHandler(e, l, app.Db)

	// Graceful shutdown
	go func() {
		if err := e.Start(":" + app.Port); err != nil && err != http.ErrServerClosed {
			e.Logger.Error(err)
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

}

func registerHandler(r *echo.Echo, l log.Logger, db *dbcontext.DB) {
	web.RegisterHandlers(r)

}
