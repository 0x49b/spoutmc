package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"spoutmc/internal/docker"
	"spoutmc/internal/global"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"
	"spoutmc/internal/watchdog"
	"spoutmc/internal/webserver"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

var logger = log.GetLogger()
var c *echo.Echo
var wd *watchdog.Watchdog
var spoutConfiguration models.SpoutConfiguration

type operation func(ctx context.Context) error

func main() {
	printBanner()
	err := readConfiguration()
	if err != nil {
		log.HandleError(err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupOps := map[string]operation{
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
			go watchdog.StartFileWatcher()
			return nil
		},
	}

	startupOrder := []string{
		//database
		"spoutmc",
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
	return nil
}

func readConfiguration() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	candidates := []string{
		filepath.Join(wd, "config", "spoutmc.yaml"),
		filepath.Join(wd, "config", "spoutmc.yml"),
	}

	var data []byte
	var usedPath string
	for _, candidate := range candidates {
		if _, statErr := os.Stat(candidate); statErr == nil {
			usedPath = candidate
			data, err = os.ReadFile(candidate)
			if err != nil {
				return err
			}
			break
		}
	}

	if usedPath == "" {
		return fmt.Errorf("no config file found (looked for spout-servers.yaml/.yml)")
	}

	if err := yaml.Unmarshal(data, &spoutConfiguration); err != nil {
		return err
	}

	return nil
}

func startContainers() {

	if len(spoutConfiguration.Servers) == 0 {
		panic("spoutmc: no servers found in Configuration")
	}

	for _, s := range spoutConfiguration.Servers {
		docker.StartContainer(s)
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
