package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"spoutmc/internal/config"
	"spoutmc/internal/docker"
	"spoutmc/internal/files"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	serverpkg "spoutmc/internal/server"
	"spoutmc/internal/serverapp"
	"spoutmc/internal/servercfg"
	"spoutmc/internal/utils/sse"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	logger               = log.GetLogger(log.ModuleAPI)
	defaultServerService = serverapp.NewService()
)

func RegisterServerRoutes(g *echo.Group) {
	g.GET("/server", getServers)
	g.POST("/server", addServerHandler)
	g.GET("/server/:id", getServer)
	g.GET("/server/:id/env", getServerEnvHandler)
	g.PUT("/server/:id", updateServerHandler)
	g.GET("/server/:id/stats", getServerStats)
	g.DELETE("/server/:id", deleteServerHandler)

	g.POST("/server/:id/start", startServerHandler)
	g.POST("/server/:id/stop", stopServerHandler)
	g.POST("/server/:id/restart", restartServerHandler)
	g.POST("/server/:id/command", executeCommandHandler)

	g.GET("/server/:id/config/files", listConfigFilesHandler)
	g.GET("/server/:id/config/:filename", getConfigFileHandler)
	g.PUT("/server/:id/config/:filename", updateConfigFileHandler)

	g.GET("/server/:id/files", listServerFilesHandler)
	g.GET("/server/:id/file", getServerFileHandler)
	g.PUT("/server/:id/file", updateServerFileHandler)
	g.PUT("/server/:id/file/binary", updateServerBinaryFileHandler)

	g.GET("/server/stream", streamServers)
	g.GET("/server/:id/logs", getServerLogs)
}

func RegisterServerRoutesWithService(g *echo.Group, service *serverapp.Service) {
	if service != nil {
		defaultServerService = service
	}
	RegisterServerRoutes(g)
}

func getServerStats(c echo.Context) error {
	sseutil.SetupResponse(c)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Info("SSE client disconnected", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			stats, err := defaultServerService.GetContainerStats(c.Request().Context(), c.Param("id"))
			if err != nil {
				return err
			}

			if err := sseutil.WriteJSON(c, stats); err != nil {
				return err
			}
		}
	}
}

func getServerLogs(c echo.Context) error {
	logger.Info("SSE Client connected", zap.String("ip", c.RealIP()))

	sseutil.SetupResponse(c)

	logChan, err := defaultServerService.FetchContainerLogs(c.Request().Context(), c.Param("id"))
	if err != nil {
		logger.Error("Error fetching docker logs", zap.Error(err))
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
			logger.Info("SSE client disconnected", zap.String("ip", c.RealIP()))
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

func getServer(c echo.Context) error {
	enriched, err := defaultServerService.GetServer(c.Request().Context(), c.Param("id"))
	if errors.Is(err, serverapp.ErrServerNotFound) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, enriched)
}

func getServers(c echo.Context) error {
	containers, err := defaultServerService.ListServers(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, containers)
}

type EnrichedContainer = serverapp.EnrichedContainer

type ContainerWithStats = serverapp.ContainerWithStats

type ServerContainerRestartPolicyRequest struct {
	Policy     string `json:"policy,omitempty"`
	MaxRetries *uint  `json:"maxRetries,omitempty"`
}

type ServerRestartPolicyRequest struct {
	Container               *ServerContainerRestartPolicyRequest `json:"container,omitempty"`
	AutoStartOnSpoutmcStart *bool                                `json:"autoStartOnSpoutmcStart,omitempty"`
}

type AddServerRequest struct {
	Name          string                      `json:"name" binding:"required"`
	Image         string                      `json:"image" binding:"required"`
	Port          int                         `json:"port,omitempty"` // Optional - required for proxy, auto-assigned for lobby/game
	Proxy         bool                        `json:"proxy,omitempty"`
	Lobby         bool                        `json:"lobby,omitempty"`
	Env           map[string]string           `json:"env"`
	RestartPolicy *ServerRestartPolicyRequest `json:"restartPolicy,omitempty"`
}

func addServerHandler(c echo.Context) error {
	if config.IsGitOpsEnabled() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Add server is disabled while GitOps is enabled. Add the server manifest to the Git repository instead.",
		})
	}

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

	if req.Proxy && req.Port == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Port is required for proxy servers",
		})
	}

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

	assignedPort := req.Port
	if assignedPort == 0 {
		assignedPort = serverpkg.FindNextAvailablePort()
		logger.Info("Dynamically assigned port",
			zap.String("name", req.Name),
			zap.Int("port", assignedPort))
	}

	defaultEnvVars := serverpkg.GetDefaultEnvVars(req.Proxy, req.Lobby)
	mergedEnv := serverpkg.MergeEnvVars(defaultEnvVars, req.Env)

	logger.Info("Adding new server",
		zap.String("name", req.Name),
		zap.String("image", req.Image),
		zap.Int("port", assignedPort),
		zap.Bool("proxy", req.Proxy),
		zap.Bool("lobby", req.Lobby),
		zap.Any("env", mergedEnv))

	containerPath := "/data" // Default for lobby/game servers
	if req.Proxy {
		containerPath = "/server" // Proxy servers use /server
	}

	containerPort := "25565" // All Minecraft/Velocity servers use 25565 internally

	newServer := models.SpoutServer{
		Name:          req.Name,
		Image:         req.Image,
		Proxy:         req.Proxy,
		Lobby:         req.Lobby,
		Env:           mergedEnv,
		RestartPolicy: mapRestartPolicyRequest(req.RestartPolicy),
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

	dataPath := ""
	existingConfig := config.All()
	if existingConfig.Storage != nil {
		dataPath = existingConfig.Storage.DataPath
	}

	if err := docker.StartContainer(c.Request().Context(), newServer, dataPath); err != nil {
		logger.Error("Failed to start container", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to start container: %v", err),
		})
	}

	if !req.Proxy {
		updatedConfig := config.All()
		if err := docker.SyncVelocityTomlAndRestartProxy(c.Request().Context(), &updatedConfig); err != nil {
			logger.Error("Failed to sync velocity.toml and restart proxy", zap.Error(err))
		}
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Server added successfully",
		"name":    req.Name,
	})
}

func startServerHandler(c echo.Context) error {
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

func stopServerHandler(c echo.Context) error {
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

func restartServerHandler(c echo.Context) error {
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

type ExecuteCommandRequest struct {
	Command string `json:"command" binding:"required"`
}

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

	if err := docker.ExecuteCommand(c.Request().Context(), containerID, req.Command); err != nil {
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

type UpdateServerRequest struct {
	Name             string                      `json:"name,omitempty"`
	Env              map[string]string           `json:"env,omitempty"`
	RestartPolicy    *ServerRestartPolicyRequest `json:"restartPolicy,omitempty"`
	ApplyImmediately *bool                       `json:"applyImmediately,omitempty"`
}

func mapRestartPolicyRequest(req *ServerRestartPolicyRequest) *models.SpoutServerRestartPolicy {
	if req == nil {
		return nil
	}

	policy := &models.SpoutServerRestartPolicy{}
	if req.AutoStartOnSpoutmcStart != nil {
		autoStart := *req.AutoStartOnSpoutmcStart
		policy.AutoStartOnSpoutmcStart = &autoStart
	}

	if req.Container != nil {
		containerPolicy := &models.SpoutServerContainerRestartPolicy{
			Policy: models.DockerRestartPolicyName(req.Container.Policy),
		}

		if req.Container.MaxRetries != nil {
			maxRetries := *req.Container.MaxRetries
			containerPolicy.MaxRetries = &maxRetries
		}
		policy.Container = containerPolicy
	}

	if policy.AutoStartOnSpoutmcStart == nil && policy.Container == nil {
		return nil
	}
	return policy
}

func getServerEnvHandler(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	isProxy := false
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" && env[5:] == "VELOCITY" {
			isProxy = true
			break
		}
	}

	systemEnvVars := serverpkg.GetDefaultEnvVars(isProxy, false)

	envVars := make(map[string]string)
	for _, envStr := range containerInfo.Config.Env {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			if _, isSystemManaged := systemEnvVars[key]; !isSystemManaged {
				envVars[key] = value
			}
		}
	}
	return c.JSON(http.StatusOK, envVars)
}

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

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	currentName := containerInfo.Name
	if len(currentName) > 0 && currentName[0] == '/' {
		currentName = currentName[1:] // Remove leading slash
	}

	newName := currentName
	if req.Name != "" {
		newName = req.Name
	}

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

	if req.Name != "" && req.Name != currentName {
		serverConfig.Name = req.Name
	}
	if req.Env != nil {
		if serverConfig.Env == nil {
			serverConfig.Env = make(map[string]string)
		}
		for k, v := range req.Env {
			serverConfig.Env[k] = v
		}
	}
	if req.RestartPolicy != nil {
		serverConfig.RestartPolicy = mapRestartPolicyRequest(req.RestartPolicy)
	}

	applyImmediately := true
	if req.ApplyImmediately != nil {
		applyImmediately = *req.ApplyImmediately
	}
	if !applyImmediately && req.Name != "" && req.Name != currentName {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Renaming a server requires applyImmediately=true",
		})
	}

	if applyImmediately {
		logger.Info("Stopping and removing old container", zap.String("name", currentName))
		if err := docker.StopAndRemoveContainerById(c.Request().Context(), containerID); err != nil {
			logger.Error("Failed to stop and remove container", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to remove container: %v", err),
			})
		}

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
	}

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

	if !applyImmediately {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Server configuration updated without container restart",
			"name":    newName,
		})
	}

	dataPath := ""
	cfg = config.All()
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	if err := docker.StartContainer(c.Request().Context(), *serverConfig, dataPath); err != nil {
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

func deleteServerHandler(c echo.Context) error {
	if config.IsGitOpsEnabled() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Remove server is disabled while GitOps is enabled. Delete the server manifest from the Git repository instead.",
		})
	}

	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	removeData := true
	if removeDataParam := c.QueryParam("removeData"); removeDataParam == "false" {
		removeData = false
	}

	logger.Info("Deleting server",
		zap.String("id", containerID[:12]),
		zap.Bool("removeData", removeData))

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	isProxy := false
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" && env[5:] == "VELOCITY" {
			isProxy = true
			break
		}
	}

	if err := docker.StopContainerWithWatchdog(c.Request().Context(), containerID); err != nil {
		logger.Error("Failed to stop container", zap.Error(err))
	}

	logger.Info("Removing container", zap.String("name", serverName))
	if err := docker.RemoveContainerById(c.Request().Context(), containerID, false); err != nil {
		logger.Error("Failed to remove container", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to remove container: %v", err),
		})
	}

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
			}
		}
	}

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

	if !isProxy {
		cfg := config.All()
		if err := docker.SyncVelocityTomlAndRestartProxy(c.Request().Context(), &cfg); err != nil {
			logger.Error("Failed to sync velocity.toml and restart proxy", zap.Error(err))
		}
	}

	logger.Info("Server deleted successfully", zap.String("name", serverName))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Server deleted successfully",
		"name":    serverName,
	})
}

func streamServers(c echo.Context) error {
	logger.Debug("SSE Client connected to server stream", zap.String("ip", c.RealIP()))
	sseutil.SetupResponse(c)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			logger.Debug("SSE client disconnected from server stream", zap.String("ip", c.RealIP()))
			return nil
		case <-ticker.C:
			snapshot, err := defaultServerService.StreamSnapshot(c.Request().Context())
			if err != nil {
				if serverapp.IsContextCanceled(err, c.Request().Context()) {
					return nil
				}
				logger.Error("Error fetching containers for stream", zap.Error(err))
				continue
			}
			if err := sseutil.WriteJSON(c, snapshot); err != nil {
				return err
			}
		}
	}
}

func listConfigFilesHandler(c echo.Context) error {
	containerID := c.Param("id")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	serverType := ""
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" {
			serverType = env[5:]
			break
		}
	}

	if serverType != "PAPER" && serverType != "VELOCITY" {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"files": []string{},
		})
	}

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

	var serverDataPath string
	var configFiles []string

	if serverType == "PAPER" {
		serverDataPath = filepath.Join(dataPath, serverName, "data")
		configFiles = []string{"server.properties", "spigot.yml"}
	} else if serverType == "VELOCITY" {
		serverDataPath = filepath.Join(dataPath, serverName, "server")
		configFiles = []string{"velocity.toml"}
	}

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

func getConfigFileHandler(c echo.Context) error {
	containerID := c.Param("id")
	filename := c.Param("filename")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	if filename != "server.properties" && filename != "spigot.yml" && filename != "velocity.toml" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid config file. Only server.properties, spigot.yml, and velocity.toml are supported",
		})
	}

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	serverType := ""
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" {
			serverType = env[5:]
			break
		}
	}

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

	var serverDataPath string
	if serverType == "VELOCITY" {
		serverDataPath = filepath.Join(dataPath, serverName, "server")
	} else {
		serverDataPath = filepath.Join(dataPath, serverName, "data")
	}
	filePath := filepath.Join(serverDataPath, filename)

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

func updateConfigFileHandler(c echo.Context) error {
	containerID := c.Param("id")
	filename := c.Param("filename")

	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	if filename != "server.properties" && filename != "spigot.yml" && filename != "velocity.toml" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid config file. Only server.properties, spigot.yml, and velocity.toml are supported",
		})
	}

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

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

	serverType := ""
	for _, env := range containerInfo.Config.Env {
		if len(env) > 5 && env[:5] == "TYPE=" {
			serverType = env[5:]
			break
		}
	}

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

	var serverDataPath string
	if serverType == "VELOCITY" {
		serverDataPath = filepath.Join(dataPath, serverName, "server")
	} else {
		serverDataPath = filepath.Join(dataPath, serverName, "data")
	}
	filePath := filepath.Join(serverDataPath, filename)

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

	if err := os.WriteFile(filePath, []byte(reqBody.Content), 0644); err != nil {
		logger.Error("Failed to write config file",
			zap.String("file", filePath),
			zap.Error(err))

		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, filePath)
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to write config file",
		})
	}

	os.Remove(backupPath)

	logger.Info("Config file updated successfully",
		zap.String("server", serverName),
		zap.String("file", filename))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Config file updated successfully",
	})
}

func listServerFilesHandler(c echo.Context) error {
	containerID := c.Param("id")
	if containerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container ID is required",
		})
	}

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

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

	var serverVolumes []models.SpoutServerVolumes
	for _, server := range cfg.Servers {
		if server.Name == serverName {
			serverVolumes = server.Volumes
			break
		}
	}

	volumes := make([]map[string]interface{}, 0)
	for _, volume := range serverVolumes {
		volumePath := filepath.Join(dataPath, serverName, volume.Containerpath)

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

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

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

	var containerPath string
	for _, server := range cfg.Servers {
		if server.Name == serverName {
			if volumePath != "" {
				containerPath = volumePath
			} else if len(server.Volumes) > 0 {
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

	fullPath := filepath.Join(dataPath, serverName, containerPath, filePath)

	serverDir := filepath.Join(dataPath, serverName)
	rel, err := filepath.Rel(serverDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid file path",
		})
	}

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

	var reqBody struct {
		Content string `json:"content"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

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

	var containerPath string
	for _, server := range cfg.Servers {
		if server.Name == serverName {
			if volumePath != "" {
				containerPath = volumePath
			} else if len(server.Volumes) > 0 {
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

	fullPath := filepath.Join(dataPath, serverName, containerPath, filePath)

	serverDir := filepath.Join(dataPath, serverName)
	rel, err := filepath.Rel(serverDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid file path",
		})
	}

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

	if err := os.WriteFile(fullPath, []byte(reqBody.Content), 0644); err != nil {
		logger.Error("Failed to write file",
			zap.String("file", fullPath),
			zap.Error(err))

		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, fullPath)
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to write file",
		})
	}

	os.Remove(backupPath)

	logger.Info("File updated successfully",
		zap.String("server", serverName),
		zap.String("file", filePath))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "File updated successfully",
	})
}

func updateServerBinaryFileHandler(c echo.Context) error {
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

	var reqBody struct {
		ContentBase64 string `json:"contentBase64"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}
	if reqBody.ContentBase64 == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "contentBase64 is required",
		})
	}

	contentBytes, err := base64.StdEncoding.DecodeString(reqBody.ContentBase64)
	if err != nil {
		contentBytes, err = base64.RawStdEncoding.DecodeString(reqBody.ContentBase64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "contentBase64 is not valid base64 data",
			})
		}
	}

	containerInfo, err := docker.GetContainerById(c.Request().Context(), containerID)
	if err != nil {
		logger.Error("Failed to get container info", zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Container not found",
		})
	}

	serverName := containerInfo.Name
	if len(serverName) > 0 && serverName[0] == '/' {
		serverName = serverName[1:] // Remove leading slash
	}

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

	var containerPath string
	for _, server := range cfg.Servers {
		if server.Name == serverName {
			if volumePath != "" {
				containerPath = volumePath
			} else if len(server.Volumes) > 0 {
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

	fullPath := filepath.Join(dataPath, serverName, containerPath, filePath)

	serverDir := filepath.Join(dataPath, serverName)
	rel, err := filepath.Rel(serverDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid file path",
		})
	}

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

	if err := os.WriteFile(fullPath, contentBytes, 0644); err != nil {
		logger.Error("Failed to write binary file",
			zap.String("file", fullPath),
			zap.Error(err))

		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, fullPath)
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to write binary file",
		})
	}

	os.Remove(backupPath)

	logger.Info("Binary file updated successfully",
		zap.String("server", serverName),
		zap.String("file", filePath))

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Binary file updated successfully",
	})
}
