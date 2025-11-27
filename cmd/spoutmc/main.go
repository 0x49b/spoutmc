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
	"spoutmc/internal/storage"
	"spoutmc/internal/watchdog"
	"spoutmc/internal/webserver"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var logger = log.GetLogger()
var c *echo.Echo
var wd *watchdog.Watchdog

type operation func(ctx context.Context) error

func main() {
	printBanner()

	// Ensure config exists and EULA is accepted
	if err := config.EnsureConfigExists(); err != nil {
		log.HandleError(err)
		os.Exit(1)
	}

	err := config.ReadConfiguration()
	if err != nil {
		log.HandleError(err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupOps := map[string]operation{
		"gitSync": func(ctx context.Context) error {
			// Only initialize if GitOps is enabled
			if config.IsGitOpsEnabled() {
				logger.Info("🗄️ GitOps is enabled, initializing Git sync")
				if err := git.InitializeGitOps(); err != nil {
					return fmt.Errorf("🗄️ failed to initialize GitOps: %w", err)
				}
				// Start Git poller in background
				go git.StartGitPoller(ctx)
			} else {
				logger.Info("🗄️ GitOps is disabled, skipping Git sync")
			}
			return nil
		},
		"velocityEnvVars": func(ctx context.Context) error {
			// Auto-inject Velocity forwarding environment variables to backend servers
			logger.Info("🔐 Checking Velocity environment variables for backend servers")

			// Get or generate Velocity secret
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

			// Inject missing env vars
			updated := config.EnsureVelocityEnvVars(velocitySecret)

			if updated {
				logger.Info("✅ Velocity env vars injected - backend servers will be recreated with new configuration")
			} else {
				logger.Info("✓ All backend servers already have Velocity env vars")
			}

			return nil
		},
		"spoutmc": func(ctx context.Context) error {
			err = startSpoutMC()
			return err
		},
		"webserver": func(ctx context.Context) error {
			c, err = webserver.Start()
			return err
		},
		"database": func(ctx context.Context) error {
			err = storage.InitDB()
			return err
		},
		"watchdog": func(ctx context.Context) error {
			wd, err = watchdog.NewWatchdog(15 * time.Second)
			if err != nil {
				log.HandleError(fmt.Errorf("failed to create watchdog: %w", err))
				return err
			}

			global.Watchdog = wd

			go wd.Start(ctx)
			return nil
		},
		"fileWatcher": func(ctx context.Context) error {
			// Only start file watcher if GitOps is disabled
			if !config.IsGitOpsEnabled() {
				go watchdog.StartFileWatcher()
			} else {
				logger.Info("GitOps is enabled, file watcher disabled")
			}
			return nil
		},
		"infrastructure": func(ctx context.Context) error {
			// Initialize infrastructure containers (database, etc.)
			logger.Info("🏗️ Initializing infrastructure containers")
			return startInfrastructure()
		},
	}

	startupOrder := []string{
		//database
		"gitSync",         // Initialize GitOps first (loads config from Git)
		"velocityEnvVars", // Inject Velocity env vars to backend servers
		"infrastructure",  // Start infrastructure containers (database, etc.)
		"spoutmc",         // Then start containers with loaded config
		"watchdog",
		"fileWatcher",
		"webserver",
	}

	for _, key := range startupOrder {
		logger.Info(fmt.Sprintf("⚔️ starting: %s", key))
		if err := startupOps[key](ctx); err != nil {
			log.HandleError(fmt.Errorf("%s failed to start: %w", key, err))
			os.Exit(1)
		}
	}

	<-ctx.Done() // wait for shutdown signal
	logger.Info("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shutdownOps := map[string]operation{
		"fileWatcher": func(ctx context.Context) error {
			logger.Info(fmt.Sprintf("📁 fileWatcher will stop via context cancel"))
			return nil
		},
		"watchdog": func(ctx context.Context) error {
			logger.Info(fmt.Sprintf("🐺 watchdog will stop via context cancel"))
			return nil
		},
		"containers": func(ctx context.Context) error {
			return docker.ShutdownContainers()
		},
		"webserver": func(ctx context.Context) error {
			return webserver.Shutdown(c)
		},
	}

	shutdownOrder := []string{
		"fileWatcher",
		"watchdog",
		"containers",
		"webserver",
	}

	for _, key := range shutdownOrder {
		logger.Warn(fmt.Sprintf("⚔️ initiate stopping of: %s", key))
		if err := shutdownOps[key](shutdownCtx); err != nil {
			log.HandleError(err)
		} else {
			logger.Info(fmt.Sprintf("⚔️ %s shut down gracefully", key))
		}
	}
}

func startSpoutMC() error {
	docker.CreateSpoutNetwork("spoutnetwork")

	// Step 1: Start all non-proxy servers (game servers and lobby)
	logger.Info("🎮 Starting non-proxy servers (lobby and game servers)")
	startNonProxyContainers()

	// Step 2: Cleanup containers that are not in config
	cleanupContainersNotInConfig()

	// Step 3: Create/update velocity.toml with all server configurations
	cfg := config.All()
	logger.Info("📝 Creating velocity.toml configuration")
	if err := docker.CreateOrUpdateVelocityToml(&cfg); err != nil {
		logger.Warn("Failed to create velocity.toml on startup", zap.Error(err))
		// Don't fail startup if velocity creation fails
	}

	// Step 4: Start proxy server AFTER velocity.toml is ready
	logger.Info("🚀 Starting proxy server with configured velocity.toml")
	startProxyContainer()

	return nil
}

func cleanupContainersNotInConfig() {
	container, err := docker.GetNetworkContainers()
	if err != nil {
		logger.Error(err.Error())
	}

	if len(container) == 0 {
		return
	}

	for _, c := range container {
		_, err := config.GetServerConfigForContainerName(strings.TrimLeft(c.Names[0], "/"))
		if err != nil {
			err := docker.RemoveContainerById(c.ID, true)
			if err != nil {
				logger.Error(err.Error())
			}
		}

	}

}

// startNonProxyContainers starts all game servers and lobby servers (not proxy)
func startNonProxyContainers() {
	cfg := config.All()

	if len(cfg.Servers) == 0 {
		panic("spoutmc: no servers found in Configuration")
	}

	// Get data path from configuration
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	// Start only non-proxy servers
	for _, s := range cfg.Servers {
		if s.Proxy {
			logger.Info(fmt.Sprintf("⏭️ Skipping proxy server %s (will start after velocity.toml is ready)", s.Name))
			continue
		}

		err := docker.RecreateContainer(s, dataPath)
		if err != nil {
			if strings.Contains(err.Error(), "Cannot find container") {
				logger.Info(fmt.Sprintf("Container not found, creating new container for %s", s.Name))
				docker.StartContainer(s, dataPath)
				continue
			}
			logger.Error(fmt.Sprintf("❌ failed to start %s: %s", s.Name, err.Error()))
		}
	}

	// List started containers
	containers, err := docker.GetNetworkContainers()
	if err != nil {
		logger.Error("Failed to list containers", zap.Error(err))
		return
	}

	for _, spoutContainer := range containers {
		logger.Info(fmt.Sprintf("⛏️ Running %s (%s) with %s",
			strings.Trim(spoutContainer.Names[0], "/"),
			spoutContainer.ID[:10],
			spoutContainer.Image))
	}
}

// startProxyContainer starts the proxy server after velocity.toml is configured
func startProxyContainer() {
	cfg := config.All()

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

		logger.Info(fmt.Sprintf("🚀 Starting proxy server: %s", s.Name))
		err := docker.RecreateContainer(s, dataPath)
		if err != nil {
			if strings.Contains(err.Error(), "Cannot find container") {
				logger.Info(fmt.Sprintf("Container not found, creating new container for %s", s.Name))
				docker.StartContainer(s, dataPath)
				return
			}
			logger.Error(fmt.Sprintf("❌ failed to start proxy %s: %s", s.Name, err.Error()))
			return
		}

		logger.Info(fmt.Sprintf("✅ Proxy server %s started successfully", s.Name))
		return
	}

	logger.Warn("⚠️ No proxy server found in configuration")
}

// startInfrastructure initializes and starts infrastructure containers (database, etc.)
func startInfrastructure() error {
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
		logger.Info("🏗️ GitOps is enabled, loading infrastructure from repository")
		repoPath := git.GetLocalRepoPath()
		logger.Info("🏗️ Repository path", zap.String("path", repoPath))
		infraContainers, err = git.LoadInfrastructureFromRepository(repoPath)
		if err != nil {
			logger.Warn("Failed to load infrastructure from Git", zap.Error(err))
			return nil // Don't fail startup if infrastructure loading fails
		}
		logger.Info("🏗️ Loaded infrastructure containers from Git", zap.Int("count", len(infraContainers)))
	} else {
		// Load from local config file
		logger.Info("🏗️ GitOps disabled, loading infrastructure from local config file")
		configPath := infrastructure.GetDefaultInfrastructureConfigPath()
		infraContainers, err = infrastructure.LoadInfrastructureFromLocalConfig(configPath, logger)
		if err != nil {
			logger.Warn("Failed to load infrastructure from local config", zap.Error(err))
			return nil // Don't fail startup if infrastructure loading fails
		}
		logger.Info("🏗️ Loaded infrastructure containers from local config", zap.Int("count", len(infraContainers)))
	}

	// Check if we have any infrastructure to create
	if len(infraContainers) == 0 {
		logger.Info("🏗️ No infrastructure containers found")
		return nil
	}

	// Generate database passwords if needed
	passwords, wasGenerated, err := infrastructure.GetOrGeneratePasswords(infraContainers, logger)
	if err != nil {
		return fmt.Errorf("failed to generate database passwords: %w", err)
	}

	// Print passwords to console if they were newly generated
	if wasGenerated {
		infrastructure.PrintPasswordsToConsole(passwords)
	}

	// Create and start each infrastructure container
	for _, infraConfig := range infraContainers {
		logger.Info("Creating infrastructure container",
			zap.String("name", infraConfig.Name),
			zap.String("type", "database"))

		if err := infrastructure.CreateDatabaseContainer(infraConfig, dataPath, passwords, logger); err != nil {
			logger.Error("Failed to create infrastructure container",
				zap.String("name", infraConfig.Name),
				zap.Error(err))
			// Don't fail startup, continue with other containers
			continue
		}

		logger.Info("✅ Infrastructure container started successfully",
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
