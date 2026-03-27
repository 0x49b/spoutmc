package static

import (
	"embed"
	"io/fs"
)

var distFS embed.FS

func GetDistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
