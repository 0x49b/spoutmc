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
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo/v4"
	"github.com/pelletier/go-toml/v2"
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
	g.GET("/server/:id/env", getServerEnvHandler)
	g.PUT("/server/:id", updateServerHandler)
	g.GET("/server/:id/stats", getServerStats)
	g.GET("/versions", getVersions)
	g.DELETE("/server/:id", deleteServerHandler)

	// Server Actions
	g.POST("/server/:id/start", startServerHandler)
	g.POST("/server/:id/stop", stopServerHandler)
	g.POST("/server/:id/restart", restartServerHandler)

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

// @Summary Get available Minecraft versions
// @Description Returns a list of available Minecraft versions from configuration
// @Tags server
// @Produce json
// @Success 200 {array} string
// @Failure 500 {object} map[string]string
// @Router /versions [get]
func getVersions(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	cfg := config.All()

	// Return versions from configuration
	versions := cfg.Versions
	if versions == nil {
		versions = []string{} // Return empty array if not configured
	}

	return c.JSON(http.StatusOK, versions)
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
	Port  int               `json:"port,omitempty"` // Optional - required for proxy, auto-assigned for lobby/game
	Proxy bool              `json:"proxy,omitempty"`
	Lobby bool              `json:"lobby,omitempty"`
	Env   map[string]string `json:"env"`
}

// getDefaultEnvVars returns system-managed environment variables for each server type
// These defaults can be overridden by user-provided values
func getDefaultEnvVars(isProxy, isLobby bool) map[string]string {
	if isProxy {
		// Proxy server defaults
		return map[string]string{
			"TYPE": "VELOCITY",
		}
	} else {
		// Lobby and game server defaults
		return map[string]string{
			"EULA":        "TRUE",
			"TYPE":        "PAPER",
			"ONLINE_MODE": "FALSE",
			"GUI":         "FALSE",
			"CONSOLE":     "FALSE",
		}
	}
}

// mergeEnvVars merges default environment variables with user-provided ones
// User-provided values override defaults
func mergeEnvVars(defaults, userProvided map[string]string) map[string]string {
	merged := make(map[string]string)

	// Start with defaults
	for k, v := range defaults {
		merged[k] = v
	}

	// Override with user-provided values
	for k, v := range userProvided {
		merged[k] = v
	}

	return merged
}

// findNextAvailablePort finds the next available port starting from 25566
// Returns the next available port number
func findNextAvailablePort() int {
	existingConfig := config.All()
	usedPorts := make(map[int]bool)

	// Collect all ports currently in use
	for _, server := range existingConfig.Servers {
		for _, portMapping := range server.Ports {
			// Parse host port as integer
			if hostPort := portMapping.HostPort; hostPort != "" {
				var port int
				fmt.Sscanf(hostPort, "%d", &port)
				if port > 0 {
					usedPorts[port] = true
				}
			}
		}
	}

	// Start from 25566 (25565 is typically for proxy) and find first available
	startPort := 25566
	for port := startPort; port <= 65535; port++ {
		if !usedPorts[port] {
			return port
		}
	}

	// Fallback (should never happen unless all ports are used)
	return startPort
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
	existingConfig := config.All()
	if req.Proxy {
		for _, server := range existingConfig.Servers {
			if server.Proxy {
				logger.Warn("Attempt to add duplicate proxy server",
					zap.String("existing", server.Name),
					zap.String("requested", req.Name))
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("A proxy server already exists (%s). Only one proxy is allowed in the network.", server.Name),
				})
			}
		}
	}

	if req.Lobby {
		for _, server := range existingConfig.Servers {
			if server.Lobby {
				logger.Warn("Attempt to add duplicate lobby server",
					zap.String("existing", server.Name),
					zap.String("requested", req.Name))
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("A lobby server already exists (%s). Only one lobby is allowed in the network.", server.Name),
				})
			}
		}
	}

	// Assign port dynamically if not provided (lobby/game servers)
	assignedPort := req.Port
	if assignedPort == 0 {
		assignedPort = findNextAvailablePort()
		logger.Info("Dynamically assigned port",
			zap.String("name", req.Name),
			zap.Int("port", assignedPort))
	}

	// Merge system-managed environment variables with user-provided ones
	defaultEnvVars := getDefaultEnvVars(req.Proxy, req.Lobby)
	mergedEnv := mergeEnvVars(defaultEnvVars, req.Env)

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
				ContainerPort: fmt.Sprintf("%d", assignedPort),
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

	// Get data path from configuration
	dataPath := ""
	existingConfig = config.All()
	if existingConfig.Storage != nil {
		dataPath = existingConfig.Storage.DataPath
	}

	// Start the new container
	docker.StartContainer(newServer, dataPath)

	// Update velocity.toml if this is not a proxy server
	if !req.Proxy {
		if err := updateVelocityTomlAddServer(req.Name, assignedPort, req.Lobby); err != nil {
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

// addServerToGit adds a new server configuration to the git repository
func addServerToGit(server models.SpoutServer) error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git configuration not found")
	}

	// Marshal the server directly (without servers: wrapper)
	yamlData, err := yaml.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	// Write to git repo under /servers directory
	repoPath := gitConfig.LocalPath
	serversDir := filepath.Join(repoPath, "servers")

	// Create servers directory if it doesn't exist
	if err := os.MkdirAll(serversDir, 0755); err != nil {
		return fmt.Errorf("failed to create servers directory: %w", err)
	}

	serverFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", server.Name))

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
	systemEnvVars := getDefaultEnvVars(isProxy, false)

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
		if err := updateServerInGit(currentName, *serverConfig); err != nil {
			logger.Error("Failed to update server in git", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to update server in git: %v", err),
			})
		}
	} else {
		logger.Info("GitOps disabled, updating server in local config")
		if err := updateServerInLocalConfig(currentName, *serverConfig); err != nil {
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
	docker.StartContainer(*serverConfig, dataPath)

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Server updated successfully",
		"name":    newName,
	})
}

// updateServerInGit updates a server configuration in the git repository
func updateServerInGit(oldName string, server models.SpoutServer) error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git configuration not found")
	}

	repoPath := gitConfig.LocalPath
	serversDir := filepath.Join(repoPath, "servers")

	// If name changed, remove old file
	if oldName != server.Name {
		oldFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", oldName))
		if err := os.Remove(oldFilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old server config file: %w", err)
		}
	}

	// Marshal the updated server
	yamlData, err := yaml.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	// Write to git repo
	serverFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", server.Name))
	if err := os.WriteFile(serverFilePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write server config file: %w", err)
	}

	// Commit and push changes
	if err := git.CommitAndPushChanges(repoPath, fmt.Sprintf("Update server: %s", server.Name)); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	logger.Info("Server config updated in git repository", zap.String("file", serverFilePath))
	return nil
}

// updateServerInLocalConfig updates a server in the local spoutmc.yaml file
func updateServerInLocalConfig(oldName string, server models.SpoutServer) error {
	// Get current configuration
	currentConfig := config.All()

	// Find and update the server
	found := false
	for i, s := range currentConfig.Servers {
		if s.Name == oldName {
			currentConfig.Servers[i] = server
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("server %s not found in configuration", oldName)
	}

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

	logger.Info("Server updated in local config", zap.String("path", configPath))
	return nil
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
		if err := updateVelocityTomlRemoveServer(serverName); err != nil {
			logger.Error("Failed to update velocity.toml", zap.Error(err))
			// Don't fail the entire operation, just log the error
		}
	}

	// Exclude from watchdog monitoring
	if global.Watchdog != nil {
		global.Watchdog.Exclude(containerID)
		logger.Info("Container excluded from watchdog monitoring", zap.String("id", containerID[:12]))
	}

	// Stop and remove container
	logger.Info("Stopping and removing container", zap.String("name", serverName))
	if err := docker.StopAndRemoveContainerById(containerID); err != nil {
		logger.Error("Failed to stop and remove container", zap.Error(err))
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
		if err := removeServerFromGit(serverName); err != nil {
			logger.Error("Failed to remove server from git", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to remove server from git: %v", err),
			})
		}
	} else {
		logger.Info("GitOps disabled, removing server from local config")
		if err := removeServerFromLocalConfig(serverName); err != nil {
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

// removeServerFromGit removes a server configuration from the git repository
func removeServerFromGit(serverName string) error {
	gitConfig := config.GetGitConfig()
	if gitConfig == nil {
		return fmt.Errorf("git configuration not found")
	}

	// Remove the server file
	repoPath := gitConfig.LocalPath
	serversDir := filepath.Join(repoPath, "servers")
	serverFilePath := filepath.Join(serversDir, fmt.Sprintf("%s.yaml", serverName))

	if err := os.Remove(serverFilePath); err != nil {
		return fmt.Errorf("failed to remove server config file: %w", err)
	}

	// Commit and push changes
	if err := git.CommitAndPushChanges(repoPath, fmt.Sprintf("Remove server: %s", serverName)); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	logger.Info("Server config removed from git repository", zap.String("file", serverFilePath))
	return nil
}

// removeServerFromLocalConfig removes a server from the local spoutmc.yaml file
func removeServerFromLocalConfig(serverName string) error {
	// Get current configuration
	currentConfig := config.All()

	// Find and remove the server
	newServers := make([]models.SpoutServer, 0)
	found := false
	for _, server := range currentConfig.Servers {
		if server.Name != serverName {
			newServers = append(newServers, server)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("server %s not found in configuration", serverName)
	}

	// Update servers list
	currentConfig.Servers = newServers

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

	logger.Info("Server removed from local config", zap.String("path", configPath))
	return nil
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

// VelocityConfig represents the structure of velocity.toml
type VelocityConfig struct {
	Servers     map[string]string      `toml:"servers"`
	ForcedHosts map[string]interface{} `toml:"forced-hosts"`
	RawConfig   map[string]interface{} `toml:",inline"`
}

// findProxyServer finds the proxy server container
func findProxyServer() (*container.Summary, error) {
	containers, err := docker.GetNetworkContainers()
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	for _, c := range containers {
		// Get detailed container info to check environment variables
		containerInfo, err := docker.GetContainerById(c.ID)
		if err != nil {
			continue
		}

		// Check if this is a proxy server
		for _, env := range containerInfo.Config.Env {
			if len(env) > 5 && env[:5] == "TYPE=" && env[5:] == "VELOCITY" {
				return &c, nil
			}
		}
	}

	return nil, fmt.Errorf("no proxy server found")
}

// updateVelocityTomlAddServer adds a server to velocity.toml
func updateVelocityTomlAddServer(serverName string, serverPort int, isLobby bool) error {
	// Find proxy server
	proxyContainer, err := findProxyServer()
	if err != nil {
		logger.Warn("No proxy server found, skipping velocity.toml update", zap.Error(err))
		return nil // Don't fail the operation if no proxy exists
	}

	// Get proxy container name
	proxyName := proxyContainer.Names[0]
	if len(proxyName) > 0 && proxyName[0] == '/' {
		proxyName = proxyName[1:] // Remove leading slash
	}

	// Get data path
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}
	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		dataPath = wd
	}

	// Path to velocity.toml
	velocityTomlPath := filepath.Join(dataPath, proxyName, "server", "velocity.toml")

	// Read current velocity.toml
	data, err := os.ReadFile(velocityTomlPath)
	if err != nil {
		return fmt.Errorf("failed to read velocity.toml: %w", err)
	}

	// Parse TOML
	var velocityConfig map[string]interface{}
	if err := toml.Unmarshal(data, &velocityConfig); err != nil {
		return fmt.Errorf("failed to parse velocity.toml: %w", err)
	}

	// Get or create servers section
	servers, ok := velocityConfig["servers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
		velocityConfig["servers"] = servers
	}

	// Add server (using container name as hostname since they're on same Docker network)
	servers[serverName] = fmt.Sprintf("%s:%d", serverName, serverPort)

	// Get or create forced-hosts section
	forcedHosts, ok := velocityConfig["forced-hosts"].(map[string]interface{})
	if !ok {
		forcedHosts = make(map[string]interface{})
		velocityConfig["forced-hosts"] = forcedHosts
	}

	// Add to forced-hosts
	forcedHosts[serverName+".local"] = []string{serverName}

	// If this is a lobby server, add to try list
	if isLobby {
		tryList, ok := forcedHosts["try"].([]interface{})
		if !ok {
			tryList = []interface{}{}
		}
		// Add lobby to try list if not already there
		found := false
		for _, item := range tryList {
			if str, ok := item.(string); ok && str == serverName {
				found = true
				break
			}
		}
		if !found {
			tryList = append(tryList, serverName)
		}
		forcedHosts["try"] = tryList
	}

	// Marshal back to TOML
	updatedData, err := toml.Marshal(velocityConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal velocity.toml: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(velocityTomlPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write velocity.toml: %w", err)
	}

	logger.Info("Added server to velocity.toml",
		zap.String("server", serverName),
		zap.Int("port", serverPort),
		zap.Bool("lobby", isLobby))

	// Restart proxy server
	docker.RestartContainerById(proxyContainer.ID)
	logger.Info("Proxy server restarted", zap.String("proxy", proxyName))

	return nil
}

// updateVelocityTomlRemoveServer removes a server from velocity.toml
func updateVelocityTomlRemoveServer(serverName string) error {
	// Find proxy server
	proxyContainer, err := findProxyServer()
	if err != nil {
		logger.Warn("No proxy server found, skipping velocity.toml update", zap.Error(err))
		return nil // Don't fail the operation if no proxy exists
	}

	// Get proxy container name
	proxyName := proxyContainer.Names[0]
	if len(proxyName) > 0 && proxyName[0] == '/' {
		proxyName = proxyName[1:] // Remove leading slash
	}

	// Get data path
	cfg := config.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}
	if dataPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		dataPath = wd
	}

	// Path to velocity.toml
	velocityTomlPath := filepath.Join(dataPath, proxyName, "server", "velocity.toml")

	// Read current velocity.toml
	data, err := os.ReadFile(velocityTomlPath)
	if err != nil {
		return fmt.Errorf("failed to read velocity.toml: %w", err)
	}

	// Parse TOML
	var velocityConfig map[string]interface{}
	if err := toml.Unmarshal(data, &velocityConfig); err != nil {
		return fmt.Errorf("failed to parse velocity.toml: %w", err)
	}

	// Remove from servers section
	if servers, ok := velocityConfig["servers"].(map[string]interface{}); ok {
		delete(servers, serverName)
	}

	// Remove from forced-hosts section
	if forcedHosts, ok := velocityConfig["forced-hosts"].(map[string]interface{}); ok {
		delete(forcedHosts, serverName+".local")

		// Remove from try list if present
		if tryList, ok := forcedHosts["try"].([]interface{}); ok {
			newTryList := []interface{}{}
			for _, item := range tryList {
				if str, ok := item.(string); ok && str != serverName {
					newTryList = append(newTryList, str)
				}
			}
			forcedHosts["try"] = newTryList
		}
	}

	// Marshal back to TOML
	updatedData, err := toml.Marshal(velocityConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal velocity.toml: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(velocityTomlPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write velocity.toml: %w", err)
	}

	logger.Info("Removed server from velocity.toml", zap.String("server", serverName))

	// Restart proxy server
	docker.RestartContainerById(proxyContainer.ID)
	logger.Info("Proxy server restarted", zap.String("proxy", proxyName))

	return nil
}

// FileNode represents a file or directory in the file tree
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"isDir"`
	Size     int64       `json:"size,omitempty"`
	ModTime  string      `json:"modTime,omitempty"`
	Children []*FileNode `json:"children,omitempty"`
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

		fileTree, err := buildFileTree(volumePath, volumePath, true)
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

// buildFileTree recursively builds a tree of files and directories
// isRoot indicates if this is the root directory (should not be excluded)
func buildFileTree(basePath, currentPath string, isRoot bool) (*FileNode, error) {
	info, err := os.Stat(currentPath)
	if err != nil {
		return nil, err
	}

	// Check if this file/folder should be excluded (but not the root)
	if !isRoot && shouldExclude(info.Name()) {
		logger.Debug("Excluding file/folder",
			zap.String("name", info.Name()),
			zap.String("path", currentPath))
		return nil, fmt.Errorf("excluded by pattern")
	}

	// Get relative path for the node
	relPath, err := filepath.Rel(basePath, currentPath)
	if err != nil {
		relPath = currentPath
	}
	if relPath == "." {
		relPath = ""
	}

	node := &FileNode{
		Name:    info.Name(),
		Path:    relPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
	}

	// If it's a directory, read its contents
	if info.IsDir() {
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return node, nil // Return directory node even if we can't read it
		}

		node.Children = make([]*FileNode, 0)
		for _, entry := range entries {
			childPath := filepath.Join(currentPath, entry.Name())
			childNode, err := buildFileTree(basePath, childPath, false)
			if err != nil {
				logger.Debug("Skipping child",
					zap.String("name", entry.Name()),
					zap.Error(err))
				continue // Skip files we can't read or are excluded
			}
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}

// shouldExclude checks if a file or folder name matches any exclusion pattern
func shouldExclude(name string) bool {
	cfg := config.All()

	// If no files config or no patterns, don't exclude anything
	if cfg.Files == nil {
		logger.Debug("Files config is nil")
		return false
	}

	if len(cfg.Files.ExcludePatterns) == 0 {
		logger.Debug("No exclusion patterns configured")
		return false
	}

	// Log loaded patterns (only once per check to avoid spam)
	logger.Debug("Checking exclusion patterns",
		zap.Int("pattern_count", len(cfg.Files.ExcludePatterns)),
		zap.String("checking_name", name))

	// Check against each pattern
	for _, pattern := range cfg.Files.ExcludePatterns {
		// Support both glob patterns and exact matches
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			logger.Debug("Pattern match error",
				zap.String("pattern", pattern),
				zap.String("name", name),
				zap.Error(err))
			// If pattern is invalid, try exact match
			if pattern == name {
				logger.Debug("Exact match found",
					zap.String("pattern", pattern),
					zap.String("name", name))
				return true
			}
			continue
		}
		if matched {
			logger.Debug("Pattern matched",
				zap.String("pattern", pattern),
				zap.String("name", name))
			return true
		}
	}

	return false
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
	if !filepath.HasPrefix(fullPath, serverDir) {
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
	if !filepath.HasPrefix(fullPath, serverDir) {
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
