package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/watchdog"
	"spoutmc/internal/webserver"
	"syscall"
	"time"
)

var spoutConfiguration models.SpoutConfiguration
var logger = log.GetLogger()

type operation func(ctx context.Context) error

func main() {
	printBanner()
	logger.Info("Starting SpoutNetwork")

	if !isDockerRunning() {
		log.HandleError(errors.New("docker runtime is not running. Cannot start SpoutMC"))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go watchdog.Start()
	go startSpout()

	c := webserver.Start()

	<-ctx.Done() // wait for shutdown signal
	logger.Info("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shutdownOps := map[string]operation{
		"containers": func(ctx context.Context) error {
			return docker.ShutdownContainers()
		},
		"webserver": func(ctx context.Context) error {
			return webserver.Shutdown(c)
		},
		"watchdog": func(ctx context.Context) error {
			return watchdog.Shutdown()
		},
	}

	for key, op := range shutdownOps {
		logger.Info(fmt.Sprintf("cleaning up: %s", key))
		if err := op(shutdownCtx); err != nil {
			log.HandleError(err)
		} else {
			logger.Info(fmt.Sprintf("%s was shut down gracefully", key))
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

func startSpout() {
	err := readConfiguration()
	if err != nil {
		log.HandleError(err)
		os.Exit(1)
	}

	docker.CreateSpoutNetwork("spoutnetwork") // todo get this from config
	startContainers()                         // Todo only do a restart if really needed. On Start of SpoutMC, the WatchDog checks for exited containers and restarts them [label needed]
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
		logger.Info(fmt.Sprintf("Running spoutContainer %s", spoutContainer.Names[0]), zap.String("image", spoutContainer.Image), zap.String("containerShortId", spoutContainer.ID[:10]))
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
