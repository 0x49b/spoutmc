package infrastructure

import (
	"encoding/json"
	"net/http"
	containerpkg "spoutmc/internal/container"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/sse"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
	"go.uber.org/zap"
)

var (
	logger = log.GetLogger(log.ModuleInfrastructure)
)

// RegisterInfrastructureRoutes registers infrastructure-related routes
func RegisterInfrastructureRoutes(g *echo.Group) {
	infra := g.Group("/infrastructure")
	infra.GET("", listInfrastructure)
	infra.GET("/stream", streamInfrastructure)
	infra.GET("/:id", getInfrastructureContainer)
	infra.GET("/:id/stats", getInfrastructureStats)
	infra.GET("/:id/logs", getInfrastructureLogs)
	infra.POST("/:id/restart", restartInfrastructureContainer)
	infra.POST("/:id/stop", stopInfrastructureContainer)
	infra.POST("/:id/start", startInfrastructureContainer)
	infra.GET("/debug/all", debugAllContainers)
}

// InfrastructureContainer represents an infrastructure container response
type InfrastructureContainer struct {
	Summary container.Summary `json:"summary"`
	Type    string            `json:"type"`
}

func listInfrastructure(c echo.Context) error {
	containers, err := docker.GetInfrastructureContainers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get infrastructure containers",
		})
	}

	// Ensure we always return an array, even if empty
	enrichedContainers := make([]InfrastructureContainer, 0, len(containers))
	for _, cont := range containers {
		enrichedContainers = append(enrichedContainers, InfrastructureContainer{
			Summary: cont,
			Type:    determineInfrastructureType(cont.Labels),
		})
	}

	return c.JSON(http.StatusOK, enrichedContainers)
}

func getInfrastructureContainer(c echo.Context) error {
	containerID := c.Param("id")

	// Get detailed container info
	inspectData, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Infrastructure container not found",
		})
	}

	// Get container summary
	containers, err := docker.GetInfrastructureContainers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get infrastructure containers",
		})
	}

	// Find matching container
	var containerSummary container.Summary
	found := false
	for _, cont := range containers {
		if cont.ID == containerID {
			containerSummary = cont
			found = true
			break
		}
	}

	if !found {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Infrastructure container not found",
		})
	}

	enriched := InfrastructureContainer{
		Summary: containerSummary,
		Type:    determineInfrastructureType(containerSummary.Labels),
	}

	// Return enriched container with inspect data
	response := map[string]interface{}{
		"container":   enriched,
		"inspectData": inspectData,
	}

	return c.JSON(http.StatusOK, response)
}

// determineInfrastructureType determines the type of infrastructure container
func determineInfrastructureType(labels map[string]string) string {
	// Check for database label
	if value, exists := labels["io.spout.database"]; exists && value == "true" {
		return "database"
	}

	// Default to unknown
	return "unknown"
}

func debugAllContainers(c echo.Context) error {
	// Get all containers from docker client directly
	cli := docker.GetDockerClient()
	allContainers, err := cli.ContainerList(c.Request().Context(), container.ListOptions{All: true})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list all containers",
		})
	}

	// Return all containers with their labels
	debug := make([]map[string]interface{}, 0)
	for _, cont := range allContainers {
		debug = append(debug, map[string]interface{}{
			"id":     cont.ID[:12],
			"names":  cont.Names,
			"image":  cont.Image,
			"state":  cont.State,
			"labels": cont.Labels,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_containers": len(allContainers),
		"containers":       debug,
	})
}

func restartInfrastructureContainer(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Use shared container action
	if err := containerpkg.RestartContainer(c.Request().Context(), containerID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to restart container",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Container restarted successfully",
		"id":      containerID,
	})
}

func stopInfrastructureContainer(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Use shared container action
	if err := containerpkg.StopContainer(c.Request().Context(), containerID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to stop container",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Container stopped successfully",
		"id":      containerID,
	})
}

func startInfrastructureContainer(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	if err := containerpkg.StartContainer(c.Request().Context(), containerID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to start container",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Container started successfully",
		"id":      containerID,
	})
}

func getInfrastructureStats(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected from infrastructure stats", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			stats, err := docker.GetContainerStats(c.Request().Context(), containerID)
			if err != nil {
				logger.Debug("Could not fetch stats for infrastructure container", zap.String("id", containerID[:12]), zap.Error(err))
				continue
			}

			id, _ := shortid.Generate()
			data, err := json.Marshal(stats)
			if err != nil {
				logger.Error("Error marshalling infrastructure stats", zap.Error(err))
				continue
			}

			event := sse.Event{
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

func getInfrastructureLogs(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	logger.Info("SSE client connected to infrastructure logs", zap.String("ip", c.RealIP()))

	logChan, err := docker.FetchDockerLogs(c.Request().Context(), containerID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such container") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Container not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch container logs",
		})
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected from infrastructure logs", zap.String("ip", c.RealIP()))
			return nil
		case logline, ok := <-logChan:
			if !ok {
				return nil
			}

			id, _ := shortid.Generate()
			event := sse.Event{
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

// InfrastructureContainerWithStats combines container info with real-time stats for SSE stream
type InfrastructureContainerWithStats struct {
	Summary container.Summary `json:"summary"`
	Type    string            `json:"type"`
	Stats   interface{}       `json:"stats,omitempty"`
}

func streamInfrastructure(c echo.Context) error {
	logger.Info("SSE Client connected to infrastructure stream", zap.String("ip", c.RealIP()))

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected from infrastructure stream", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			containers, err := docker.GetInfrastructureContainers(c.Request().Context())

			if err != nil {
				logger.Error("Error fetching infrastructure containers for stream", zap.Error(err))
				continue
			}

			// Enrich containers with type and stats
			enrichedContainers := make([]InfrastructureContainerWithStats, 0, len(containers))
			for _, cont := range containers {
				item := InfrastructureContainerWithStats{
					Summary: cont,
					Type:    determineInfrastructureType(cont.Labels),
				}
				stats, err := docker.GetContainerStats(c.Request().Context(), cont.ID)
				if err != nil {
					logger.Debug("Could not fetch stats for infrastructure container", zap.String("id", cont.ID[:12]), zap.Error(err))
				} else {
					item.Stats = stats
				}
				enrichedContainers = append(enrichedContainers, item)
			}

			id, _ := shortid.Generate()
			data, err := json.Marshal(enrichedContainers)
			if err != nil {
				logger.Error("Error marshalling infrastructure containers", zap.Error(err))
				continue
			}

			event := sse.Event{
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
