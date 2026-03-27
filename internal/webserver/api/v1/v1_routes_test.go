package v1

import (
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestChatIngestRouteRegistered(t *testing.T) {
	e := echo.New()
	api := e.Group("/api")
	RegisterV1WithModules(api, Modules{})

	for _, r := range e.Routes() {
		if strings.Contains(r.Path, "chat-ingest") {
			if r.Path != "/api/v1/player/chat-ingest" {
				t.Fatalf("unexpected chat-ingest path: %s", r.Path)
			}
			return
		}
	}
	t.Fatal("expected POST /api/v1/player/chat-ingest to be registered")
}
