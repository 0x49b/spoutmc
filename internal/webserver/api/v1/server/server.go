package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
	"go.uber.org/zap"
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

// RegisterServerRoutes registers container/server-related API endpoints.
//
// @Tags server
// @Router /server [get]
// @Router /server/{id} [get]
// @Router /server/{id}/stats [get]
// @Router /server/{id}/logs [get]
// @Produce json
func RegisterServerRoutes(g *echo.Group) {
	// REST
	g.GET("/server", getServers)
	g.GET("/server/:id", getServer)
	g.GET("/server/:id/stats", getServerStats)

	//SSE
	g.GET("/server/:id/logs", getServerLogs)
}

// @Summary Get real-time container stats
// @Description Server-Sent Events (SSE) for real-time container statistics
// @Tags server
// @Produce text/event-stream
// @Param id path string true "Container ID"
// @Success 200 {string} string "Stream of container stats"
// @Failure 500 {object} map[string]string
// @Router /server/{id}/stats [get]
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

// @Summary Stream container logs
// @Description Server-Sent Events (SSE) for container logs
// @Tags server
// @Produce text/event-stream
// @Param id path string true "Container ID"
// @Success 200 {string} string "Stream of container logs"
// @Failure 500 {object} map[string]string
// @Router /server/{id}/logs [get]
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

// @Summary Get server details
// @Description Retrieve information about a specific Docker container
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} interface{}
// @Failure 500 {object} map[string]string
// @Router /server/{id} [get]
func getServer(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	container, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, container)
}

// @Summary Get list of servers
// @Description Returns a list of servers in the network
// @Tags server
// @Produce json
// @Success 200 {array} interface{}
// @Failure 500 {object} map[string]string
// @Router /server [get]
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
