package sseutil

import (
	"encoding/json"
	"net/http"
	"spoutmc/internal/sse"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
)

// SetupResponse configures SSE headers once per request.
func SetupResponse(c echo.Context) {
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// WriteJSON writes a JSON payload as SSE event and flushes.
func WriteJSON(c echo.Context, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return WriteBytes(c, data)
}

// WriteBytes writes a raw byte payload as SSE event and flushes.
func WriteBytes(c echo.Context, data []byte) error {
	id, _ := shortid.Generate()
	event := sse.Event{
		ID:        []byte(id),
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	if err := event.MarshalTo(c.Response()); err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}

// JSONError is a tiny helper to keep SSE handlers consistent.
func JSONError(c echo.Context, status int, msg string) error {
	return c.JSON(status, map[string]string{"error": msg})
}

// IsClientClosed reports if a request context is done.
func IsClientClosed(c echo.Context) bool {
	return c.Request().Context().Err() != nil
}

// NoContent is exported to avoid importing net/http in callers.
const NoContent = http.StatusNoContent
