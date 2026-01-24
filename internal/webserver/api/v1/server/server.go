package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"spoutmc/internal/config"
	containerpkg "spoutmc/internal/container"
	"spoutmc/internal/docker"
	"spoutmc/internal/files"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	serverpkg "spoutmc/internal/server"
	"spoutmc/internal/servercfg"
	"spoutmc/internal/sse"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleAPI)

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
	g.GET("/server/:id/env", getServerEnvHandler)
	g.PUT("/server/:id", updateServerHandler)
	g.GET("/server/:id/stats", getServerStats)
	g.DELETE("/server/:id", deleteServerHandler)

	// Server Actions
	g.POST("/server/:id/start", startServerHandler)
	g.POST("/server/:id/stop", stopServerHandler)
	g.POST("/server/:id/restart", restartServerHandler)
	g.POST("/server/:id/command", executeCommandHandler)

	// Config Files
	g.GET("/server/:id/config/files", listConfigFilesHandler)
	g.GET("/server/:id/config/:filename", getConfigFileHandler)
	g.PUT("/server/:id/config/:filename", updateConfigFileHandler)

	// File Browser
	g.GET("/server/:id/files", listServerFilesHandler)
	g.GET("/server/:id/file", getServerFileHandler)
	g.PUT("/server/:id/file", updateServerFileHandler)

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
	// Get detailed container info for StartedAt
	inspectData, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return err
	}

	// Get container summary for labels and basic info
	containers, err := docker.GetNetworkContainers()
	if err != nil {
		return err
	}

	// Find matching container to get summary and labels
	var containerSummary container.Summary
	var found bool
	for _, cont := range containers {
		if cont.ID == c.Param("id") {
			containerSummary = cont
			found = true
			break
		}
	}

	if !found {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Create enriched response with server type
	enriched := EnrichedContainer{
		Summary: containerSummary,
		Type:    serverpkg.DetermineServerType(containerSummary.Labels),
	}

	if inspectData.State != nil {
		enriched.StartedAt = inspectData.State.StartedAt
	}

	return c.JSON(http.StatusOK, enriched)
}

// @Summary Get list of servers
// @Description Returns a list of servers in the network
// @Tags server
// @Produce json
// @Success 200 {array} interface{}
// @Failure 500 {object} map[string]string
// @Router /server [get]
func getServers(c echo.Context) error {
	containers, err := docker.GetNetworkContainers()
	if err != nil {
		return err
	}

	// Enrich containers with StartedAt timestamp and Type
	enrichedContainers := make([]EnrichedContainer, 0, len(containers))
	for _, container := range containers {
		enriched := EnrichedContainer{
			Summary: container,
			Type:    serverpkg.DetermineServerType(container.Labels),
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
	Type      string `json:"Type,omitempty"`      // Server type: "proxy", "lobby", or "game"
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
	Port  int               `json:"port,omitempty"` // Optional - required for proxy, auto-assigned for lobby/game
	Proxy bool              `json:"proxy,omitempty"`
	Lobby bool              `json:"lobby,omitempty"`
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

	if req.Name == "" || req.Image == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name and image are required fields",
		})
	}

	// Port is required for proxy servers
	if req.Proxy && req.Port == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Port is required for proxy servers",
		})
	}

	// Validate proxy/lobby constraints - only one of each allowed
	if req.Proxy {
		if err := serverpkg.ValidateProxyConstraint(); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
	}

	if req.Lobby {
		if err := serverpkg.ValidateLobbyConstraint(); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
	}

	// Assign port dynamically if not provided (lobby/game servers)
	assignedPort := req.Port
	if assignedPort == 0 {
		assignedPort = serverpkg.FindNextAvailablePort()
		logger.Info("Dynamically assigned port",
			zap.String("name", req.Name),
			zap.Int("port", assignedPort))
	}

	// Merge system-managed environment variables with user-provided ones
	defaultEnvVars := serverpkg.GetDefaultEnvVars(req.Proxy, req.Lobby)
	mergedEnv := serverpkg.MergeEnvVars(defaultEnvVars, req.Env)

	logger.Info("Adding new server",
		zap.String("name", req.Name),
		zap.String("image", req.Image),
		zap.Int("port", assignedPort),
		zap.Bool("proxy", req.Proxy),
		zap.Bool("lobby", req.Lobby),
		zap.Any("env", mergedEnv))

	// Determine default containerpath based on server type
	containerPath := "/data" // Default for lobby/game servers
	if req.Proxy {
		containerPath = "/server" // Proxy servers use /server
	}

	// Configure port mapping
	containerPort := "25565" // All Minecraft/Velocity servers use 25565 internally

	// Create new server model
	newServer := models.SpoutServer{
		Name:  req.Name,
		Image: req.Image,
		Proxy: req.Proxy,
		Lobby: req.Lobby,
		Env:   mergedEnv,
		Ports: []models.SpoutServerPorts{
			{
				HostPort:      fmt.Sprintf("%d", assignedPort),
				ContainerPort: containerPort,
			},
		},
		Volumes: []models.SpoutServerVolumes{
			{
				Containerpath: containerPath,
			},
		},
	}

	// Check if GitOps is enabled
	if config.IsGitOpsEnabled() {
		logger.Info("GitOps enabled, adding server to git repository")
		if err := servercfg.AddServerToGit(newServer); err != nil {
			logger.Error("Failed to add server to git", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to add server to git: %v", err),
			})
		}
	} else {
		logger.Info("GitOps disabled, adding server to local config")
		if err := servercfg.AddServerToLocalConfig(newServer); err != nil {
			logger.Error("Failed to add server to local config", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to add server to local config: %v", err),
			})
		}
	}

	// Get data path from configuration
	dataPath := ""
	existingConfig := config.All()
	if existingConfig.Storage != nil {
		dataPath = existingConfig.Storage.DataPath
	}

	// Start the new container
	if err := docker.StartContainer(newServer, dataPath); err != nil {
		logger.Error("Failed to start container", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to start container: %v", err),
		})
	}

	// Update velocity.toml if this is not a proxy server
	if !req.Proxy {
		// Reload config to get the updated state
		updatedConfig := config.All()
		if err := docker.UpdateVelocityTomlAddServer(&updatedConfig, req.Name, assignedPort, req.Lobby); err != nil {
			logger.Error("Failed to update velocity.toml", zap.Error(err))
			// Don't fail the entire operation, just log the error
		}
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Server added successfully",
		"name":    req.Name,
	})
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

	// Use shared container action
	if err := containerpkg.StartContainer(containerID); err != nil {
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

	// Use shared container action
	if err := containerpkg.StopContainer(containerID); err != nil {
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

	// Use shared container action
	if err := containerpkg.RestartContainer(containerID); err != nil {
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

// ExecuteCommandRequest represents the request body for executing a command
type ExecuteCommandRequest struct {
	Command string `json:"command" binding:"required"`
}

// @Summary Execute a command in a server container
// @Description Executes a Minecraft console command in the server container
// @Tags server
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Param command body ExecuteCommandRequest true "Command to execute"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/command [post]
func executeCommandHandler(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	var req ExecuteCommandRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Command == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Command is required",
		})
	}

	// Execute the command in the container
	ctx := context.Background()
	if err := docker.ExecuteCommand(ctx, containerID, req.Command); err != nil {
		logger.Error("Failed to execute command",
			zap.String("container", containerID[:12]),
			zap.String("command", req.Command),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to execute command: %v", err),
		})
	}

	logger.Info("Command executed successfully",
		zap.String("container", containerID[:12]),
		zap.String("command", req.Command))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Command executed successfully",
	})
}

// UpdateServerRequest represents the request body for updating a server
type UpdateServerRequest struct {
	Name string            `json:"name,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
}

// @Summary Get server environment variables
// @Description Returns environment variables for a server (excluding system-managed ones)
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /server/{id}/env [get]
func getServerEnvHandler(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Get container info
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Determine if this is a proxy or lobby/game server
	isProxy := false
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" && env[5:] == "VELOCITY" {
			isProxy = true
			break
		}
	}

	// Get system-managed env vars for this server type
	systemEnvVars := serverpkg.GetDefaultEnvVars(isProxy, false)

	// Extract env vars from container
	envVars := make(map[string]string)
	for _, envStr := range containerInfo.Config.Env {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Skip system-managed env vars
			if _, isSystemManaged := systemEnvVars[key]; !isSystemManaged {
				envVars[key] = value
			}
		}
	}

	return c.JSON(http.StatusOK, envVars)
}

// @Summary Update a server
// @Description Updates server configuration (name, environment variables)
// @Tags server
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Param server body UpdateServerRequest true "Server update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id} [put]
func updateServerHandler(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	var req UpdateServerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get container info to find current server name
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name
	currentName := containerInfo.Name
	if len(currentName) > 0 && currentName[0] == '/' {
		currentName = currentName[1:] // Remove leading slash
	}

	newName := currentName
	if req.Name != "" {
		newName = req.Name
	}

	// Get current configuration
	cfg := config.All()
	var serverConfig *models.SpoutServer
	for i, server := range cfg.Servers {
		if server.Name == currentName {
			serverConfig = &cfg.Servers[i]
			break
		}
	}

	if serverConfig == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found in configuration",
		})
	}

	// Update server configuration
	if req.Name != "" && req.Name != currentName {
		serverConfig.Name = req.Name
	}
	if req.Env != nil {
		// Merge with existing env vars
		if serverConfig.Env == nil {
			serverConfig.Env = make(map[string]string)
		}
		for k, v := range req.Env {
			serverConfig.Env[k] = v
		}
	}

	// Stop and remove old container
	logger.Info("Stopping and removing old container", zap.String("name", currentName))
	if err := docker.StopAndRemoveContainerById(containerID); err != nil {
		logger.Error("Failed to stop and remove container", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to remove container: %v", err),
		})
	}

	// If name changed, rename data directory
	if newName != currentName {
		dataPath := ""
		if cfg.Storage != nil {
			dataPath = cfg.Storage.DataPath
		}
		if dataPath != "" {
			oldPath := filepath.Join(dataPath, currentName)
			newPath := filepath.Join(dataPath, newName)
			if _, err := os.Stat(oldPath); err == nil {
				if err := os.Rename(oldPath, newPath); err != nil {
					logger.Warn("Failed to rename server data directory",
						zap.String("oldPath", oldPath),
						zap.String("newPath", newPath),
						zap.Error(err))
				}
			}
		}
	}

	// Update configuration file (GitOps or local)
	if config.IsGitOpsEnabled() {
		logger.Info("GitOps enabled, updating server in git repository")
		if err := servercfg.UpdateServerInGit(currentName, *serverConfig); err != nil {
			logger.Error("Failed to update server in git", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to update server in git: %v", err),
			})
		}
	} else {
		logger.Info("GitOps disabled, updating server in local config")
		if err := servercfg.UpdateServerInLocalConfig(currentName, *serverConfig); err != nil {
			logger.Error("Failed to update server in local config", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to update server in local config: %v", err),
			})
		}
	}

	// Get data path from configuration
	dataPath := ""
	cfg = config.All()
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	// Start the updated container
	if err := docker.StartContainer(*serverConfig, dataPath); err != nil {
		logger.Error("Failed to start updated container", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to start container: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Server updated successfully",
		"name":    newName,
	})
}

// @Summary Delete a server
// @Description Stops, removes container, deletes data (optional), and removes from config
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Param removeData query boolean false "Remove server data directory" default(true)
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id} [delete]
func deleteServerHandler(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Get removeData query parameter (default: true)
	removeData := true
	if removeDataParam := c.QueryParam("removeData"); removeDataParam == "false" {
		removeData = false
	}

	logger.Info("Deleting server",
		zap.String("id", containerID[:12]),
		zap.Bool("removeData", removeData))

	// Get container details to find server name
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name from container
	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	// Check if this is a proxy server
	isProxy := false
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" && env[5:] == "VELOCITY" {
			isProxy = true
			break
		}
	}

	// Update velocity.toml if this is not a proxy server
	if !isProxy {
		cfg := config.All()
		if err := docker.UpdateVelocityTomlRemoveServer(&cfg, serverName); err != nil {
			logger.Error("Failed to update velocity.toml", zap.Error(err))
			// Don't fail the entire operation, just log the error
		}
	}

	// Stop the container (this also excludes from watchdog)
	if err := containerpkg.StopContainer(containerID); err != nil {
		logger.Error("Failed to stop container", zap.Error(err))
		// Continue with removal even if stop fails
	}

	// Remove container (without removing volumes - handled separately below)
	logger.Info("Removing container", zap.String("name", serverName))
	if err := docker.RemoveContainerById(containerID, false); err != nil {
		logger.Error("Failed to remove container", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to remove container: %v", err),
		})
	}

	// Remove data directory if requested
	if removeData {
		cfg := config.All()
		dataPath := ""
		if cfg.Storage != nil {
			dataPath = cfg.Storage.DataPath
		}

		if dataPath != "" {
			serverDataPath := filepath.Join(dataPath, serverName)
			logger.Info("Removing server data directory",
				zap.String("path", serverDataPath))

			if err := os.RemoveAll(serverDataPath); err != nil {
				logger.Warn("Failed to remove server data directory",
					zap.String("path", serverDataPath),
					zap.Error(err))
				// Don't fail the entire operation if data removal fails
			}
		}
	}

	// Remove from configuration (GitOps or local)
	if config.IsGitOpsEnabled() {
		logger.Info("GitOps enabled, removing server from git repository")
		if err := servercfg.RemoveServerFromGit(serverName); err != nil {
			logger.Error("Failed to remove server from git", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to remove server from git: %v", err),
			})
		}
	} else {
		logger.Info("GitOps disabled, removing server from local config")
		if err := servercfg.RemoveServerFromLocalConfig(serverName); err != nil {
			logger.Error("Failed to remove server from local config", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to remove server from local config: %v", err),
			})
		}
	}

	logger.Info("Server deleted successfully", zap.String("name", serverName))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Server deleted successfully",
		"name":    serverName,
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
			containers, err := docker.GetNetworkContainers()

			if err != nil {
				logger.Error("Error fetching containers for stream", zap.Error(err))
				continue
			}

			// Enrich containers with stats and StartedAt
			enrichedContainers := make([]ContainerWithStats, 0, len(containers))
			for _, container := range containers {
				// Create enriched container with StartedAt and Type
				enrichedContainer := EnrichedContainer{
					Summary: container,
					Type:    serverpkg.DetermineServerType(container.Labels),
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

// @Summary List available config files
// @Description Returns list of editable config files for a server (server.properties, spigot.yml)
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/config/files [get]
func listConfigFilesHandler(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Get container info to find server name and type
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name
	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	// Get server type from environment variables
	serverType := ""
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" {
			serverType = env[5:]
			break
		}
	}

	// Only PAPER and VELOCITY servers have editable config files
	if serverType != "PAPER" && serverType != "VELOCITY" {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"files": []string{},
		})
	}

	// Get data path from configuration
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to determine data path",
			})
		}
		dataPath = wd
	}

	// Build path to server data directory based on server type
	var serverDataPath string
	var configFiles []string

	if serverType == "PAPER" {
		// PAPER servers mount at /data
		serverDataPath = filepath.Join(dataPath, serverName, "data")
		configFiles = []string{"server.properties", "spigot.yml"}
	} else if serverType == "VELOCITY" {
		// VELOCITY proxy servers mount at /server
		serverDataPath = filepath.Join(dataPath, serverName, "server")
		configFiles = []string{"velocity.toml"}
	}

	// Check which config files exist
	availableFiles := []string{}
	for _, filename := range configFiles {
		filePath := filepath.Join(serverDataPath, filename)
		if _, err := os.Stat(filePath); err == nil {
			availableFiles = append(availableFiles, filename)
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"files": availableFiles,
	})
}

// @Summary Get config file content
// @Description Returns the content of a server config file
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Param filename path string true "Config filename (server.properties or spigot.yml)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/config/{filename} [get]
func getConfigFileHandler(c echo.Context) error {
	containerID := c.Param("id")
	filename := c.Param("filename")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Validate filename
	if filename != "server.properties" && filename != "spigot.yml" && filename != "velocity.toml" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid config file. Only server.properties, spigot.yml, and velocity.toml are supported",
		})
	}

	// Get container info to find server name and type
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name
	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	// Get server type from environment variables
	serverType := ""
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" {
			serverType = env[5:]
			break
		}
	}

	// Get data path from configuration
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to determine data path",
			})
		}
		dataPath = wd
	}

	// Build path to config file based on server type
	var serverDataPath string
	if serverType == "VELOCITY" {
		// VELOCITY proxy servers mount at /server
		serverDataPath = filepath.Join(dataPath, serverName, "server")
	} else {
		// PAPER servers mount at /data
		serverDataPath = filepath.Join(dataPath, serverName, "data")
	}
	filePath := filepath.Join(serverDataPath, filename)

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error("Failed to read config file",
			zap.String("file", filePath),
			zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Config file not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"filename": filename,
		"content":  string(content),
	})
}

// @Summary Update config file content
// @Description Updates the content of a server config file
// @Tags server
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Param filename path string true "Config filename (server.properties or spigot.yml)"
// @Param body body map[string]string true "File content"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/config/{filename} [put]
func updateConfigFileHandler(c echo.Context) error {
	containerID := c.Param("id")
	filename := c.Param("filename")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Validate filename
	if filename != "server.properties" && filename != "spigot.yml" && filename != "velocity.toml" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid config file. Only server.properties, spigot.yml, and velocity.toml are supported",
		})
	}

	// Parse request body
	var reqBody struct {
		Content string `json:"content"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if reqBody.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Content is required",
		})
	}

	// Get container info to find server name and type
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name
	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	// Get server type from environment variables
	serverType := ""
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" {
			serverType = env[5:]
			break
		}
	}

	// Get data path from configuration
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to determine data path",
			})
		}
		dataPath = wd
	}

	// Build path to config file based on server type
	var serverDataPath string
	if serverType == "VELOCITY" {
		// VELOCITY proxy servers mount at /server
		serverDataPath = filepath.Join(dataPath, serverName, "server")
	} else {
		// PAPER servers mount at /data
		serverDataPath = filepath.Join(dataPath, serverName, "data")
	}
	filePath := filepath.Join(serverDataPath, filename)

	// Create backup of existing file
	backupPath := filePath + ".backup"
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Rename(filePath, backupPath); err != nil {
			logger.Error("Failed to create backup of config file",
				zap.String("file", filePath),
				zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create backup of config file",
			})
		}
	}

	// Write new content
	if err := os.WriteFile(filePath, []byte(reqBody.Content), 0644); err != nil {
		logger.Error("Failed to write config file",
			zap.String("file", filePath),
			zap.Error(err))

		// Restore backup if write failed
		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, filePath)
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to write config file",
		})
	}

	// Remove backup after successful write
	os.Remove(backupPath)

	logger.Info("Config file updated successfully",
		zap.String("server", serverName),
		zap.String("file", filename))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Config file updated successfully",
	})
}

// @Summary List server files
// @Description Returns a tree of files and directories in server volumes
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/files [get]
func listServerFilesHandler(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	// Get container info to find server name
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name
	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	// Get data path from configuration
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to determine data path",
			})
		}
		dataPath = wd
	}

	// Get server volumes from configuration
	var serverVolumes []models.SpoutServerVolumes
	for _, server := range cfg.Servers {
		if server.Name == serverName {
			serverVolumes = server.Volumes
			break
		}
	}

	// Build file tree for each volume
	volumes := make([]map[string]interface{}, 0)
	for _, volume := range serverVolumes {
		volumePath := filepath.Join(dataPath, serverName, volume.Containerpath)

		// Check if path exists
		if _, err := os.Stat(volumePath); os.IsNotExist(err) {
			continue
		}

		fileTree, err := files.BuildFileTree(volumePath, volumePath, true)
		if err != nil {
			logger.Error("Failed to build file tree", zap.Error(err))
			continue
		}

		volumes = append(volumes, map[string]interface{}{
			"containerPath": volume.Containerpath,
			"hostPath":      volumePath,
			"files":         fileTree,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"volumes": volumes,
	})
}

// @Summary Get server file content
// @Description Returns the content of a file in server volumes
// @Tags server
// @Produce json
// @Param id path string true "Container ID"
// @Param path query string true "Relative file path within volume"
// @Param volume query string false "Container volume path (default: first volume)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/file [get]
func getServerFileHandler(c echo.Context) error {
	containerID := c.Param("id")
	filePath := c.QueryParam("path")
	volumePath := c.QueryParam("volume")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	if filePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "File path is required",
		})
	}

	// Get container info to find server name
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name
	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	// Get data path from configuration
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to determine data path",
			})
		}
		dataPath = wd
	}

	// Get server configuration to find volumes
	var containerPath string
	for _, server := range cfg.Servers {
		if server.Name == serverName {
			if volumePath != "" {
				// Use specified volume
				containerPath = volumePath
			} else if len(server.Volumes) > 0 {
				// Use first volume
				containerPath = server.Volumes[0].Containerpath
			}
			break
		}
	}

	if containerPath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No volumes found for server",
		})
	}

	// Build full file path
	fullPath := filepath.Join(dataPath, serverName, containerPath, filePath)

	// Security check: ensure path is within server directory
	serverDir := filepath.Join(dataPath, serverName)
	if !strings.HasPrefix(fullPath, serverDir+string(filepath.Separator)) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid file path",
		})
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		logger.Error("Failed to read file",
			zap.String("file", fullPath),
			zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "File not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"path":    filePath,
		"content": string(content),
	})
}

// @Summary Update server file content
// @Description Updates the content of a file in server volumes
// @Tags server
// @Accept json
// @Produce json
// @Param id path string true "Container ID"
// @Param path query string true "Relative file path within volume"
// @Param volume query string false "Container volume path (default: first volume)"
// @Param body body map[string]string true "File content"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /server/{id}/file [put]
func updateServerFileHandler(c echo.Context) error {
	containerID := c.Param("id")
	filePath := c.QueryParam("path")
	volumePath := c.QueryParam("volume")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	if filePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "File path is required",
		})
	}

	// Parse request body
	var reqBody struct {
		Content string `json:"content"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get container info to find server name
	containerInfo, err := docker.GetContainerById(containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	// Get server name
	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	// Get data path from configuration
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to determine data path",
			})
		}
		dataPath = wd
	}

	// Get server configuration to find volumes
	var containerPath string
	for _, server := range cfg.Servers {
		if server.Name == serverName {
			if volumePath != "" {
				// Use specified volume
				containerPath = volumePath
			} else if len(server.Volumes) > 0 {
				// Use first volume
				containerPath = server.Volumes[0].Containerpath
			}
			break
		}
	}

	if containerPath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No volumes found for server",
		})
	}

	// Build full file path
	fullPath := filepath.Join(dataPath, serverName, containerPath, filePath)

	// Security check: ensure path is within server directory
	serverDir := filepath.Join(dataPath, serverName)
	if !strings.HasPrefix(fullPath, serverDir+string(filepath.Separator)) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid file path",
		})
	}

	// Create backup of existing file
	backupPath := fullPath + ".backup"
	if _, err := os.Stat(fullPath); err == nil {
		if err := os.Rename(fullPath, backupPath); err != nil {
			logger.Error("Failed to create backup of file",
				zap.String("file", fullPath),
				zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create backup of file",
			})
		}
	}

	// Write new content
	if err := os.WriteFile(fullPath, []byte(reqBody.Content), 0644); err != nil {
		logger.Error("Failed to write file",
			zap.String("file", fullPath),
			zap.Error(err))

		// Restore backup if write failed
		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, fullPath)
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to write file",
		})
	}

	// Remove backup after successful write
	os.Remove(backupPath)

	logger.Info("File updated successfully",
		zap.String("server", serverName),
		zap.String("file", filePath))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "File updated successfully",
	})
}
