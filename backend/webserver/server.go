package webserver

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"spoutmc/backend/log"
)

var logger = log.New()

func Start() {

	app := fiber.New(fiber.Config{
		AppName:               "SpoutWebserver",
		DisableStartupMessage: true,
	})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello World!")
	})
	logger.Fatal("Cannot start", zap.Error(app.Listen(":8080")))
}
func Shutdown() error {
	logger.Info("Webserver initiated shutdown procedure")
	return nil
}
