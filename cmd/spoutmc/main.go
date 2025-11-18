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
	"spoutmc/internal/log"
	"spoutmc/internal/storage"
	"spoutmc/internal/watchdog"
	"spoutmc/internal/webserver"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

var logger = log.GetLogger()
var c *echo.Echo
var wd *watchdog.Watchdog

type operation func(ctx context.Context) error

func main() {
	printBanner()

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
	}

	startupOrder := []string{
		//database
		"gitSync", // Initialize GitOps first (loads config from Git)
		"spoutmc", // Then start containers with loaded config
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
	startContainers()
	cleanupContainersNotInConfig()
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

func startContainers() {

	cfg := config.All()

	if len(cfg.Servers) == 0 {
		panic("spoutmc: no servers found in Configuration")
	}

	// Get data path from configuration
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}

	for _, s := range cfg.Servers {
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

	containers, err := docker.GetNetworkContainers()

	if err != nil {
		panic(err)
	}

	for _, spoutContainer := range containers {
		logger.Info(fmt.Sprintf("⛏️ Running %s (%s) with %s",
			strings.Trim(spoutContainer.Names[0], "/"),
			spoutContainer.ID[:10],
			spoutContainer.Image))
	}

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
