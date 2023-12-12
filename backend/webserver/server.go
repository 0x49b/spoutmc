package webserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"go.uber.org/zap"
	"net/http"
	"spoutmc/backend/log"
	"spoutmc/backend/webserver/routes"
	"spoutmc/web"
)

var logger = log.New()

func Start() {

	app := fiber.New(fiber.Config{
		AppName:               "SpoutWebserver",
		DisableStartupMessage: true,
	})

	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(web.GetEmbedFS()),
		PathPrefix: "static",
		Browse:     true,
	}))

	api := app.Group("/api")
	v1 := api.Group("/v1")
	routes.SetupContainerRoutes(v1)

	logger.Fatal("Cannot start", zap.Error(app.Listen(":8080")))
}

func Shutdown() error {
	logger.Info("Webserver initiated shutdown procedure")
	return nil
}
