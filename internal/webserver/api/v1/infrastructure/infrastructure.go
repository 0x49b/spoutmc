package infrastructure

import (
	"errors"
	"net/http"
	"spoutmc/internal/docker"
	"spoutmc/internal/infrastructureapp"
	"spoutmc/internal/log"
	"spoutmc/internal/utils/sse"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	logger              = log.GetLogger(log.ModuleInfrastructure)
	defaultInfraService = infrastructureapp.NewService()
)

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

func RegisterInfrastructureRoutesWithService(g *echo.Group, service *infrastructureapp.Service) {
	if service != nil {
		defaultInfraService = service
	}
	RegisterInfrastructureRoutes(g)
}

type InfrastructureContainer struct {
	Summary container.Summary `json:"summary"`
	Type    string            `json:"type"`
}

func listInfrastructure(c echo.Context) error {
	containers, err := defaultInfraService.ListContainers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get infrastructure containers",
		})
	}
	return c.JSON(http.StatusOK, containers)
}

func getInfrastructureContainer(c echo.Context) error {
	containerID := c.Param("id")
	enriched, inspectData, err := defaultInfraService.GetContainer(c.Request().Context(), containerID)
	if errors.Is(err, infrastructureapp.ErrInfrastructureNotFound) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Infrastructure container not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get infrastructure containers",
		})
	}

	response := map[string]interface{}{
		"container":   enriched,
		"inspectData": inspectData,
	}

	return c.JSON(http.StatusOK, response)
}

func debugAllContainers(c echo.Context) error {
	cli := docker.GetDockerClient()
	allContainers, err := cli.ContainerList(c.Request().Context(), container.ListOptions{All: true})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list all containers",
		})
	}

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

	if err := docker.RestartContainerWithWatchdog(c.Request().Context(), containerID); err != nil {
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

	if err := docker.StopContainerWithWatchdog(c.Request().Context(), containerID); err != nil {
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

	if err := docker.StartContainerWithWatchdog(c.Request().Context(), containerID); err != nil {
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

	sseutil.SetupResponse(c)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected from infrastructure stats", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			stats, err := defaultInfraService.GetContainerStats(c.Request().Context(), containerID)
			if err != nil {
				logger.Debug("Could not fetch stats for infrastructure container", zap.String("id", containerID[:12]), zap.Error(err))
				continue
			}
			if err := sseutil.WriteJSON(c, stats); err != nil {
				return err
			}
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

	sseutil.SetupResponse(c)

	logChan, err := defaultInfraService.FetchContainerLogs(c.Request().Context(), containerID)
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

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected from infrastructure logs", zap.String("ip", c.RealIP()))
			return nil
		case logline, ok := <-logChan:
			if !ok {
				return nil
			}

			if err := sseutil.WriteBytes(c, []byte(logline)); err != nil {
				return err
			}
		}
	}
}

type InfrastructureContainerWithStats struct {
	Summary container.Summary `json:"summary"`
	Type    string            `json:"type"`
	Stats   interface{}       `json:"stats,omitempty"`
}

func streamInfrastructure(c echo.Context) error {
	logger.Info("SSE Client connected to infrastructure stream", zap.String("ip", c.RealIP()))
	sseutil.SetupResponse(c)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected from infrastructure stream", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			containers, err := defaultInfraService.StreamSnapshot(c.Request().Context())

			if err != nil {
				logger.Error("Error fetching infrastructure containers for stream", zap.Error(err))
				continue
			}
			if err := sseutil.WriteJSON(c, containers); err != nil {
				return err
			}
		}
	}
}
