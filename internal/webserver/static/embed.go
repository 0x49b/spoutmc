package static

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetDistFS returns the embedded filesystem with the dist/ prefix removed
func GetDistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
