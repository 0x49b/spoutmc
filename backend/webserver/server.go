package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
	"net/http"
	"os"
	"os/signal"
	"spoutmc/backend/config"
	"spoutmc/backend/dbcontext"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	v1Container "spoutmc/backend/webserver/api/v1"
	"spoutmc/web"
	"time"
)

var logger = log.CreateLogger()

func hello(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			// Read
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				if err.Error() == "EOF" {
					c.Logger().Info("Client disconnected gracefully")
				} else {
					c.Logger().Error("WebSocket read error", zap.Error(err))
				}
				break // Exit the loop if an error occurs
			}

			fmt.Printf("%s\n", msg)

			if msg == "server" {
				containerList, err := docker.GetNetworkContainers()
				if err != nil {
					logger.Error("Cannot load containerlist", zap.Error(err))
				}

				containerListJson, err := json.Marshal(containerList)
				err = websocket.Message.Send(ws, containerListJson)
			}

			// Write
			err = websocket.Message.Send(ws, "Hello, Client!")

			if err != nil {
				c.Logger().Error("WebSocket write error", zap.Error(err))
				break // Exit the loop if writing fails
			}

		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func Start() *echo.Echo {

	logger.Info("Starting Webserver")

	conf := config.New(os.Getenv("3000"), os.Getenv("ENV"))

	e := echo.New()

	e.HideBanner = true
	e.HidePort = true
	app := conf.Bootstrap()

	e.Pre(middleware.RemoveTrailingSlash())

	//e.Use(log.CreateZapLoggerMiddleware(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:4200"},
	}))
	e.Use(middleware.Secure())
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

	// Frontend deliver Page
	registerHandler(e, app.Db)

	// Frontend Handler REST based
	apiGroup := e.Group("/api")
	v1 := apiGroup.Group("/v1")
	v1Container.RegisterContainerAPI(v1)

	// FrontendHandler WS based
	/*wsGroup := e.Group("/ws")
	v1ws := wsGroup.Group("/v1")
	v1Ws.RegisterWS(v1ws)*/

	e.GET("ws", hello)

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

	writeRoutes(e)

	return e
}

func registerHandler(r *echo.Echo, db *dbcontext.DB) {
	web.RegisterHandlers(r)
}

func writeRoutes(e *echo.Echo) {
	data, err := json.MarshalIndent(e.Routes(), "", "  ")
	if err != nil {
		logger.Error("json marshalling error", zap.Error(err))
	}

	err = os.WriteFile("routes.json", data, 0644)
	if err != nil {
		logger.Error("writing error", zap.Error(err))
	}
}

func Shutdown(e *echo.Echo) error {
	err := e.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}
