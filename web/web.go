package web

import (
	"embed"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"net/http"
	"spoutmc/backend/log"
)

var (
	logger = log.New()

	//go:embed all:dist
	embedDirStatic embed.FS
)

func GetEmbedFS() embed.FS {
	return embedDirStatic
}

func RegisterFrontend(app *fiber.App) {

	app.Use("/", filesystem.New(filesystem.Config{
		Root:  http.FS(embedDirStatic),
		Index: "index.html",
	}))

}
