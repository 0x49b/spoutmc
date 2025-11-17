package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"spoutmc/internal/config"
	"spoutmc/internal/docker"
	"spoutmc/internal/git"
	"spoutmc/internal/global"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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
	g.POST("/server", addServerHandler)
	g.GET("/server/:id", getServer)
	g.GET("/server/:id/stats", getServerStats)

	// Server Actions
	g.POST("/server/:id/start", startServerHandler)
	g.POST("/server/:id/stop", stopServerHandler)
	g.POST("/server/:id/restart", restartServerHandler)

	//SSE
	g.GET("/server/stream", streamServers)
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

	// Enrich containers with StartedAt timestamp
	enrichedContainers := make([]EnrichedContainer, 0, len(containers))
	for _, container := range containers {
		enriched := EnrichedContainer{
			Summary: container,
		}

		// Get detailed container info to extract StartedAt
		inspectData, err := docker.GetContainerById(container.ID)
		if err == nil && inspectData.State != nil {
			enriched.StartedAt = inspectData.State.StartedAt
		}

		enrichedContainers = append(enrichedContainers, enriched)
	}

	return c.JSON(http.StatusOK, enrichedContainers)
}

// EnrichedContainer combines container summary with additional runtime info
type EnrichedContainer struct {
	container.Summary
	StartedAt string `json:"StartedAt,omitempty"` // ISO 8601 timestamp when container was started
}

// ContainerWithStats combines container info with real-time stats
type ContainerWithStats struct {
	Container EnrichedContainer `json:"container"`
	Stats     interface{}       `json:"stats,omitempty"`
}

// AddServerRequest represents the request body for adding a new server
type AddServerRequest struct {
	Name  string            `json:"name" binding:"required"`
	Image string            `json:"image" binding:"required"`
	Port  int               `json:"port" binding:"required"`
	Env   map[string]string `json:"env"`
}

// @Summary Add a new server
// @Description Creates a new server configuration and deploys it (via GitOps or local config)
// @Tags server
// @Accept json
// @Produce json
// @Param server body AddServerRequest true "Server configuration"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server [post]
func addServerHandler(c echo.Context) error {
	var req AddServerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.Image == "" || req.Port == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name, image, and port are required fields",
		})
	}

	logger.Info("Adding new server", zap.String("name", req.Name), zap.String("image", req.Image))

	// Create new server model
	newServer := models.SpoutServer{
		Name:  req.Name,
		Image: req.Image,
		Env:   req.Env,
		Ports: []models.SpoutServerPorts{
			{
				HostPort:      fmt.Sprintf("%d", req.Port),
				ContainerPort: fmt.Sprintf("%d", req.Port),
			},
		},
		Volumes: []models.SpoutServerVolumes{
			{
				Hostpath:      models.StringSlice{req.Name},
				Containerpath: "/server",
			},
		},
	}

	// Check if GitOps is enabled
	if config.IsGitOpsEnabled() {
		logger.Info("GitOps enabled, adding server to git repository")
		if err := addServerToGit(newServer); err != nil {
			logger.Error("Failed to add server to git", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to add server to git: %v", err),
			})
		}
	} else {
		logger.Info("GitOps disabled, adding server to local config")
		if err := addServerToLocalConfig(newServer); err != nil {
			logger.Error("Failed to add server to local config", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to add server to local config: %v", err),
			})
		}
	}

	// Start the new container
	docker.StartContainer(newServer)

	return c.JSON(http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Server added successfully",
		"name":    req.Name,
	})
}

// addServerToGit adds a new server configuration to the git repository
func addServerToGit(server models.SpoutServer) error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git configuration not found")
	}

	// Create YAML content for the new server
	serverConfig := models.SpoutConfiguration{
		Servers: []models.SpoutServer{server},
	}

	yamlData, err := yaml.Marshal(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	// Write to git repo
	repoPath := gitConfig.LocalPath
	serverFilePath := filepath.Join(repoPath, fmt.Sprintf("%s.yaml", server.Name))

	if err := os.WriteFile(serverFilePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write server config file: %w", err)
	}

	// Commit and push changes
	if err := git.CommitAndPushChanges(repoPath, fmt.Sprintf("Add server: %s", server.Name)); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	logger.Info("Server config added to git repository", zap.String("file", serverFilePath))
	return nil
}

// addServerToLocalConfig adds a new server to the local spoutmc.yaml file
func addServerToLocalConfig(server models.SpoutServer) error {
	// Get current configuration
	currentConfig := config.All()

	// Add new server
	currentConfig.Servers = append(currentConfig.Servers, server)

	// Marshal to YAML
	yamlData, err := yaml.Marshal(currentConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Write to config file (try both .yaml and .yml)
	configPaths := []string{
		filepath.Join(wd, "config", "spoutmc.yaml"),
		filepath.Join(wd, "config", "spoutmc.yml"),
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		// Default to .yaml if neither exists
		configPath = configPaths[0]
	}

	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Reload configuration
	if err := config.ReadConfiguration(); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	logger.Info("Server added to local config", zap.String("path", configPath))
	return nil
}

// @Summary Start a server
// @Description Starts a stopped container and includes it in watchdog monitoring
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/start [post]
func startServerHandler(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	logger.Info("Starting container", zap.String("id", containerID[:12]))

	// Start the container
	docker.StartContainerById(containerID)

	// Include in watchdog monitoring
	if global.Watchdog != nil {
		global.Watchdog.Include(containerID)
		logger.Info("Container included in watchdog monitoring", zap.String("id", containerID[:12]))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Container started successfully",
		"id":      containerID,
	})
}

// @Summary Stop a server
// @Description Stops a running container and excludes it from watchdog monitoring
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/stop [post]
func stopServerHandler(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	logger.Info("Stopping container", zap.String("id", containerID[:12]))

	// Exclude from watchdog monitoring BEFORE stopping
	// This prevents the watchdog from immediately restarting it
	if global.Watchdog != nil {
		global.Watchdog.Exclude(containerID)
		logger.Info("Container excluded from watchdog monitoring", zap.String("id", containerID[:12]))
	}

	// Stop the container
	docker.StopContainerById(containerID)

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Container stopped successfully",
		"id":      containerID,
	})
}

// @Summary Restart a server
// @Description Restarts a container (remains in watchdog monitoring)
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/restart [post]
func restartServerHandler(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	logger.Info("Restarting container", zap.String("id", containerID[:12]))

	// Restart the container
	docker.RestartContainerById(containerID)

	// Ensure it's included in watchdog monitoring after restart
	if global.Watchdog != nil {
		global.Watchdog.Include(containerID)
		logger.Info("Container ensured in watchdog monitoring", zap.String("id", containerID[:12]))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Container restarted successfully",
		"id":      containerID,
	})
}

// @Summary Stream server list updates
// @Description Server-Sent Events (SSE) for real-time server list updates with stats
// @Tags server
// @Produce text/event-stream
// @Success 200 {string} string "Stream of server list updates"
// @Failure 500 {object} map[string]string
// @Router /server/stream [get]
func streamServers(c echo.Context) error {
	logger.Info("SSE Client connected to server stream", zap.String("ip", c.RealIP()))

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected from server stream", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			lock.Lock()
			containers, err := docker.GetNetworkContainers()
			lock.Unlock()

			if err != nil {
				logger.Error("Error fetching containers for stream", zap.Error(err))
				continue
			}

			// Enrich containers with stats and StartedAt
			enrichedContainers := make([]ContainerWithStats, 0, len(containers))
			for _, container := range containers {
				// Create enriched container with StartedAt
				enrichedContainer := EnrichedContainer{
					Summary: container,
				}

				// Get detailed container info to extract StartedAt
				inspectData, err := docker.GetContainerById(container.ID)
				if err == nil && inspectData.State != nil {
					enrichedContainer.StartedAt = inspectData.State.StartedAt
				}

				containerData := ContainerWithStats{
					Container: enrichedContainer,
				}

				// Try to fetch stats for this container (non-blocking)
				stats, err := docker.GetContainerStats(container.ID)
				if err != nil {
					// Log but don't fail the whole stream
					logger.Debug("Could not fetch stats for container",
						zap.String("id", container.ID[:12]),
						zap.Error(err))
				} else {
					containerData.Stats = stats
				}

				enrichedContainers = append(enrichedContainers, containerData)
			}

			id, _ := shortid.Generate()
			data, err := json.Marshal(enrichedContainers)
			if err != nil {
				logger.Error("Error marshalling containers", zap.Error(err))
				continue
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
