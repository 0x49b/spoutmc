package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
	"go.uber.org/zap"
	"io"
	"net/http"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"sync"
	"time"
)

var lock = sync.Mutex{}
var logger = log.GetLogger()

type Event struct {
	ID        []byte
	Data      []byte
	Event     []byte
	Retry     []byte
	Comment   []byte
	Timestamp int64
}

// RegisterServerRoutes registers routes related to the /server endpoint
func RegisterServerRoutes(g *echo.Group) {
	// REST
	g.GET("/server", getServers)
	g.GET("/server/:id", getServer)
	g.GET("/server/:id/stats", getServerStats)

	//SSE
	g.GET("/server/:id/logs", getServerLogs)
}

func getServerStats(c echo.Context) error {

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			container, err := docker.GetContainerStats(c.Param("id"))
			if err != nil {
				return err
			}

			id, _ := shortid.Generate()
			data, err := json.Marshal(container)
			if err != nil {
				return err
			}

			event := Event{
				ID:        []byte(id),
				Data:      data,
				Timestamp: time.Now().Unix(),
			}
			if err = event.MarshalTo(w); err != nil {
				return err
			}
			w.Flush()
		}
	}

}

func getServerLogs(c echo.Context) error {
	logger.Info("SSE Client connected", zap.String("ip", c.RealIP()))

	ctx := context.Background()
	logChan, err := docker.FetchDockerLogs(ctx, c.Param("id"))
	if err != nil {
		logger.Error("Error fetching docker logs", zap.Error(err))
		return err
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected", zap.String("ip", c.RealIP()))
			return nil
		default:
			for logline := range logChan {
				id, _ := shortid.Generate()
				event := Event{
					ID:        []byte(id),
					Data:      []byte(logline),
					Timestamp: time.Now().Unix(),
				}
				if err := event.MarshalTo(w); err != nil {
					return err
				}
				w.Flush()
			}
		}
	}

}

func getServer(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	container, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, container)
}

func getServers(c echo.Context) error {

	lock.Lock()
	defer lock.Unlock()

	containers, err := docker.GetNetworkContainers()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, containers)
}

func (ev *Event) MarshalTo(w io.Writer) error {
	// Marshalling part is taken from: https://github.com/r3labs/sse/blob/c6d5381ee3ca63828b321c16baa008fd6c0b4564/http.go#L16
	if len(ev.Data) == 0 && len(ev.Comment) == 0 {
		return nil
	}

	if len(ev.Data) > 0 {
		if _, err := fmt.Fprintf(w, "id: %s\n", ev.ID); err != nil {
			return err
		}

		sd := bytes.Split(ev.Data, []byte("\n"))
		for i := range sd {
			if _, err := fmt.Fprintf(w, "data: %s\n", sd[i]); err != nil {
				return err
			}
		}

		if len(ev.Event) > 0 {
			if _, err := fmt.Fprintf(w, "event: %s\n", ev.Event); err != nil {
				return err
			}
		}

		if len(ev.Retry) > 0 {
			if _, err := fmt.Fprintf(w, "retry: %s\n", ev.Retry); err != nil {
				return err
			}
		}
	}

	if len(ev.Comment) > 0 {
		if _, err := fmt.Fprintf(w, ": %s\n", ev.Comment); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}

	return nil
}
