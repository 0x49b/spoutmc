package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"spoutmc/internal/config"
	"spoutmc/internal/docker"
	"spoutmc/internal/git"
	"spoutmc/internal/global"
	"spoutmc/internal/infrastructure"
	"spoutmc/internal/log"
	"spoutmc/internal/watchdog"
	"spoutmc/internal/webserver"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleMain)
var c *echo.Echo
var wd *watchdog.Watchdog

type operation func(ctx context.Context) error

func main() {
	printBanner()

	if err := initializeConfiguration(); err != nil {
		log.HandleError(err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runStartupSequence(ctx); err != nil {
		log.HandleError(err)
		os.Exit(1)
	}

	<-ctx.Done() // wait for shutdown signal
	logger.Info("Shutdown signal received")

	runShutdownSequence()
}

// initializeConfiguration ensures config exists and loads it
func initializeConfiguration() error {
	if err := config.EnsureConfigExists(); err != nil {
		return err
	}

	if err := config.ReadConfiguration(); err != nil {
		return err
	}

	return nil
}

// runStartupSequence executes all startup operations in order
func runStartupSequence(ctx context.Context) error {
	startupOps := getStartupOperations()
	startupOrder := getStartupOrder()

	for _, key := range startupOrder {
		logger.Info(fmt.Sprintf("starting: %s", key))
		if err := startupOps[key](ctx); err != nil {
			return fmt.Errorf("%s failed to start: %w", key, err)
		}
	}

	return nil
}

// runShutdownSequence executes all shutdown operations in order
func runShutdownSequence() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shutdownOps := getShutdownOperations()
	shutdownOrder := getShutdownOrder()

	for _, key := range shutdownOrder {
		logger.Warn(fmt.Sprintf("initiate stopping of: %s", key))
		if err := shutdownOps[key](shutdownCtx); err != nil {
			log.HandleError(err)
		} else {
			logger.Info(fmt.Sprintf("%s shut down gracefully", key))
		}
	}
}

// getStartupOperations returns all startup operations
func getStartupOperations() map[string]operation {
	return map[string]operation{
		"gitSync":         startGitSync,
		"velocityEnvVars": startVelocityEnvVars,
		"infrastructure":  startInfrastructureOp,
		"spoutmc":         startSpoutMCOp,
		"watchdog":        startWatchdogOp,
		"fileWatcher":     startFileWatcherOp,
		"webserver":       startWebserverOp,
	}
}

// getStartupOrder returns the order of startup operations
func getStartupOrder() []string {
	return []string{
		"gitSync",         // Initialize GitOps first (loads config from Git)
		"velocityEnvVars", // Inject Velocity env vars to backend servers
		"infrastructure",  // Start infrastructure containers (database, etc.)
		"spoutmc",         // Then start containers with loaded config
		"watchdog",
		"fileWatcher",
		"webserver",
	}
}

// getShutdownOperations returns all shutdown operations
func getShutdownOperations() map[string]operation {
	return map[string]operation{
		"fileWatcher": func(ctx context.Context) error { return nil },
		"watchdog":    func(ctx context.Context) error { return nil },
		"containers":  func(ctx context.Context) error { return docker.ShutdownContainers(ctx) },
		"webserver":   func(ctx context.Context) error { return webserver.Shutdown(c) },
	}
}

// getShutdownOrder returns the order of shutdown operations
func getShutdownOrder() []string {
	return []string{
		"fileWatcher",
		"watchdog",
		"containers",
		"webserver",
	}
}

// startGitSync initializes GitOps if enabled
func startGitSync(ctx context.Context) error {
	gitLogger := log.GetLogger(log.ModuleGit)
	if config.IsGitOpsEnabled() {
		gitLogger.Info("GitOps is enabled, initializing Git sync")
		if err := git.InitializeGitOps(); err != nil {
			return fmt.Errorf("failed to initialize GitOps: %w", err)
		}
		// Start Git poller in background
		go git.StartGitPoller(ctx)
	} else {
		gitLogger.Info("GitOps is disabled, skipping Git sync")
	}
	return nil
}

// startVelocityEnvVars auto-injects Velocity forwarding environment variables
func startVelocityEnvVars(ctx context.Context) error {
	dockerLogger := log.GetLogger(log.ModuleDocker)
	dockerLogger.Info("Checking Velocity environment variables for backend servers")

	cfg := config.All()
	dataPath := ""
	proxyName := ""

	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	// Find proxy server
	for i := range cfg.Servers {
		if cfg.Servers[i].Proxy {
			proxyName = cfg.Servers[i].Name
			break
		}
	}

	velocitySecret := docker.GetOrGenerateVelocitySecret(dataPath, proxyName)
	updated := config.EnsureVelocityEnvVars(velocitySecret)

	if updated {
		dockerLogger.Info("Velocity env vars injected - backend servers will be recreated with new configuration")
	} else {
		dockerLogger.Info("All backend servers already have Velocity env vars")
	}

	return nil
}

// startInfrastructureOp initializes infrastructure containers
func startInfrastructureOp(ctx context.Context) error {
	infraLogger := log.GetLogger(log.ModuleInfrastructure)
	infraLogger.Info("Initializing infrastructure containers")
	return startInfrastructure(ctx)
}

// startSpoutMCOp starts the SpoutMC server network
func startSpoutMCOp(ctx context.Context) error {
	return startSpoutMC(ctx)
}

// startWatchdogOp starts the watchdog service
func startWatchdogOp(ctx context.Context) error {
	var err error
	wd, err = watchdog.NewWatchdog(15 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to create watchdog: %w", err)
	}

	global.Watchdog = wd
	go wd.Start(ctx)
	return nil
}

// startFileWatcherOp starts the file watcher if GitOps is disabled
func startFileWatcherOp(ctx context.Context) error {
	if !config.IsGitOpsEnabled() {
		go watchdog.StartFileWatcher()
	} else {
		logger.Info("GitOps is enabled, file watcher disabled")
	}
	return nil
}

// startWebserverOp starts the web server
func startWebserverOp(ctx context.Context) error {
	var err error
	c, err = webserver.Start()
	return err
}

func startSpoutMC(ctx context.Context) error {
	serverLogger := log.GetLogger(log.ModuleServer)
	docker.CreateSpoutNetwork(ctx, "spoutnetwork")

	// Step 1: Start all non-proxy servers (game servers and lobby)
	serverLogger.Info("Starting non-proxy servers (lobby and game servers)")
	startNonProxyContainers(ctx)

	// Step 2: Cleanup containers that are not in config
	cleanupContainersNotInConfig(ctx)

	// Step 3: Create/update velocity.toml with all server configurations
	cfg := config.All()
	serverLogger.Info("Creating velocity.toml configuration")
	if err := docker.CreateOrUpdateVelocityToml(&cfg); err != nil {
		serverLogger.Warn("Failed to create velocity.toml on startup", zap.Error(err))
		// Don't fail startup if velocity creation fails
	}

	// Step 4: Start proxy server AFTER velocity.toml is ready
	serverLogger.Info("Starting proxy server with configured velocity.toml")
	startProxyContainer(ctx)

	return nil
}

func cleanupContainersNotInConfig(ctx context.Context) {
	container, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		logger.Error(err.Error())
	}

	if len(container) == 0 {
		return
	}

	for _, c := range container {
		_, err := config.GetServerConfigForContainerName(strings.TrimLeft(c.Names[0], "/"))
		if err != nil {
			err := docker.RemoveContainerById(ctx, c.ID, true)
			if err != nil {
				logger.Error(err.Error())
			}
		}

	}

}

// startNonProxyContainers starts all game servers and lobby servers (not proxy)
func startNonProxyContainers(ctx context.Context) {
	serverLogger := log.GetLogger(log.ModuleServer)
	cfg := config.All()

	if len(cfg.Servers) == 0 {
		serverLogger.Info("No servers found in configuration - application will start without containers")
		serverLogger.Info("You can add servers via the web UI or by updating your configuration")
		return
	}

	// Get data path from configuration
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	// Start only non-proxy servers
	for _, s := range cfg.Servers {
		if s.Proxy {
			serverLogger.Info(fmt.Sprintf("Skipping proxy server %s (will start after velocity.toml is ready)", s.Name))
			continue
		}

		err := docker.RecreateContainer(ctx, s, dataPath)
		if err != nil {
			if strings.Contains(err.Error(), "Cannot find container") {
				serverLogger.Info(fmt.Sprintf("Container not found, creating new container for %s", s.Name))
				if err := docker.StartContainer(ctx, s, dataPath); err != nil {
					serverLogger.Error(fmt.Sprintf("failed to start %s: %v", s.Name, err))
				}
				continue
			}
			serverLogger.Error(fmt.Sprintf("failed to start %s: %s", s.Name, err.Error()))
		}
	}

	// List started containers
	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		serverLogger.Error("Failed to list containers", zap.Error(err))
		return
	}

	for _, spoutContainer := range containers {
		serverLogger.Info(fmt.Sprintf("Running %s (%s) with %s",
			strings.Trim(spoutContainer.Names[0], "/"),
			spoutContainer.ID[:10],
			spoutContainer.Image))
	}
}

// startProxyContainer starts the proxy server after velocity.toml is configured
func startProxyContainer(ctx context.Context) {
	serverLogger := log.GetLogger(log.ModuleServer)
	cfg := config.All()

	if len(cfg.Servers) == 0 {
		serverLogger.Info("No servers configured - skipping proxy startup")
		return
	}

	// Get data path from configuration
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	// Find and start the proxy server
	for _, s := range cfg.Servers {
		if !s.Proxy {
			continue
		}

		serverLogger.Info(fmt.Sprintf("Starting proxy server: %s", s.Name))
		err := docker.RecreateContainer(ctx, s, dataPath)
		if err != nil {
			if strings.Contains(err.Error(), "Cannot find container") {
				serverLogger.Info(fmt.Sprintf("Container not found, creating new container for %s", s.Name))
				if err := docker.StartContainer(ctx, s, dataPath); err != nil {
					serverLogger.Error(fmt.Sprintf("failed to start proxy %s: %v", s.Name, err))
					return
				}
				return
			}
			serverLogger.Error(fmt.Sprintf("failed to start proxy %s: %s", s.Name, err.Error()))
			return
		}

		serverLogger.Info(fmt.Sprintf("Proxy server %s started successfully", s.Name))
		return
	}

	serverLogger.Warn("No proxy server found in configuration")
}

// startInfrastructure initializes and starts infrastructure containers (database, etc.)
func startInfrastructure(ctx context.Context) error {
	infraLogger := log.GetLogger(log.ModuleInfrastructure)
	cfg := config.All()

	// Get data path
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	// Load infrastructure configurations from Git or local config
	var infraContainers []infrastructure.InfrastructureContainer
	var err error
	if config.IsGitOpsEnabled() {
		infraLogger.Info("GitOps is enabled, loading infrastructure from repository")
		repoPath := git.GetLocalRepoPath()
		infraLogger.Info("Repository path", zap.String("path", repoPath))
		infraContainers, err = git.LoadInfrastructureFromRepository(repoPath)
		if err != nil {
			infraLogger.Warn("Failed to load infrastructure from Git", zap.Error(err))
			return nil // Don't fail startup if infrastructure loading fails
		}
		infraLogger.Info("Loaded infrastructure containers from Git", zap.Int("count", len(infraContainers)))
	} else {
		// Load from local config file
		infraLogger.Info("GitOps disabled, loading infrastructure from local config file")
		configPath := infrastructure.GetDefaultInfrastructureConfigPath()
		infraContainers, err = infrastructure.LoadInfrastructureFromLocalConfig(configPath, infraLogger.GetZapLogger())
		if err != nil {
			infraLogger.Warn("Failed to load infrastructure from local config", zap.Error(err))
			return nil // Don't fail startup if infrastructure loading fails
		}
		infraLogger.Info("Loaded infrastructure containers from local config", zap.Int("count", len(infraContainers)))
	}

	// Check if we have any infrastructure to create
	if len(infraContainers) == 0 {
		infraLogger.Info("No infrastructure containers found")
		return nil
	}

	// Generate database passwords if needed
	passwords, wasGenerated, err := infrastructure.GetOrGeneratePasswords(infraContainers, infraLogger.GetZapLogger())
	if err != nil {
		return fmt.Errorf("failed to generate database passwords: %w", err)
	}

	// Print passwords to console if they were newly generated
	if wasGenerated {
		infrastructure.PrintPasswordsToConsole(passwords)
	}

	// Create and start each infrastructure container
	for _, infraConfig := range infraContainers {
		infraLogger.Info("Creating infrastructure container",
			zap.String("name", infraConfig.Name),
			zap.String("type", "database"))

		if err := infrastructure.CreateDatabaseContainer(ctx, infraConfig, dataPath, passwords, infraLogger.GetZapLogger()); err != nil {
			infraLogger.Error("Failed to create infrastructure container",
				zap.String("name", infraConfig.Name),
				zap.Error(err))
			// Don't fail startup, continue with other containers
			continue
		}

		infraLogger.Info("Infrastructure container started successfully",
			zap.String("name", infraConfig.Name))
	}

	return nil
}

func printBanner() {
	fmt.Println()
	fmt.Println("     =()=                                                    ")
	fmt.Println(" ,/'\\_||_           _____                   __  __  _________")
	fmt.Println(" ( (___  `.        / ___/____  ____  __  __/ /_/  |/  / ____/")
	fmt.Println(" `\\./  `=='        \\__ \\/ __ \\/ __ \\/ / / / __/ /|_/ / /     ")
	fmt.Println("        |||       ___/ / /_/ / /_/ / /_/ / /_/ /  / / /___   ")
	fmt.Println("        |||      /____/ .___/\\____/\\__,_/\\__/_/  /_/\\____/   ")
	fmt.Println("        |||          /_/                            0.0.1    ")
	fmt.Println()
}
