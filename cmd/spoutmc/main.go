package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"spoutmc/internal/docker"
	"spoutmc/internal/global"
	"spoutmc/internal/kubernetes"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"
	"spoutmc/internal/watchdog"
	"spoutmc/internal/webserver"
	"strings"
	"syscall"
	"time"
)

var spoutConfiguration models.SpoutConfiguration
var logger = log.GetLogger()
var c *echo.Echo
var err error
var wd *watchdog.Watchdog

type operation func(ctx context.Context) error

func main() {
	printBanner()
	logger.Info("Starting SpoutNetwork")

	/*if !isDockerRunning() {
		log.HandleError(errors.New("docker runtime is not running. Cannot start SpoutMC"))
		os.Exit(1)
	}*/

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupOps := map[string]operation{
		"spoutmc": func(ctx context.Context) error {
			err = startSpoutMC()
			return nil
		},
		"webserver": func(ctx context.Context) error {
			c, err = webserver.Start()
			return nil
		},
		"database": func(ctx context.Context) error {
			err = storage.InitDB()
			return nil
		},
		"kubernetes": func(ctx context.Context) error {
			return kubernetes.StartKubeClient()
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
	}

	/**
	original order:
	"kubernetes",
	"database",
	"spoutmc",
	"watchdog",
	"webserver",
	*/
	startupOrder := []string{
		"kubernetes",
		"database",
		"webserver",
	}

	for _, key := range startupOrder {
		logger.Info(fmt.Sprintf("starting: %s", key))
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
		"watchdog",
		"webserver",
		"containers",
	}

	for _, key := range shutdownOrder {
		logger.Warn(fmt.Sprintf("initiate stopping of: %s", key))
		if err := shutdownOps[key](shutdownCtx); err != nil {
			log.HandleError(err)
		} else {
			logger.Info(fmt.Sprintf("%s shut down gracefully", key))
		}
	}
}

func isDockerRunning() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

func startSpoutMC() error {
	err := readConfiguration()
	if err != nil {
		log.HandleError(err)
		return err
	}

	docker.CreateSpoutNetwork("spoutnetwork") // todo get this from config
	startContainers()
	return nil
}

func readConfiguration() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, "config", "spout-servers.json")
	logger.Debug(path)
	jsonFile, err := os.Open(path)
	if err != nil {
		return err
	}
	logger.Info("Successfully opened configuration file")

	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &spoutConfiguration)
	if err != nil {
		return err
	}

	return nil
}

func readServersToStart() (models.SpoutConfiguration, error) {
	wd, err := os.Getwd()
	if err != nil {
		return models.SpoutConfiguration{}, err
	}
	path := filepath.Join(wd, "config", "spout-servers.json")

	jsonFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}

	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(jsonFile)

	byteValue, _ := io.ReadAll(jsonFile)
	var spoutServers models.SpoutConfiguration
	err = json.Unmarshal(byteValue, &spoutServers)
	if err != nil {
		return models.SpoutConfiguration{}, err
	}
	return spoutServers, nil
}

func startContainers() {

	logger.Info("Starting Containers")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	spoutServers, err := readServersToStart()

	if err != nil {
		panic(err)
	}

	for _, s := range spoutServers.Servers {
		docker.StartContainer(s)
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, spoutContainer := range containers {
		logger.Info(fmt.Sprintf("🚀 Running %s (%s) with %s", strings.Trim(spoutContainer.Names[0], "/"), spoutContainer.ID[:10], spoutContainer.Image))

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
