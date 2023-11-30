package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	"spoutmc/backend/models"
	"spoutmc/backend/webserver"
	"sync"
	"syscall"
	"time"
)

var spoutConfiguration models.SpoutConfiguration
var logger = log.New()

func main() {
	printBanner()
	logger.Info("Starting SpoutNetwork")
	go startSpout()
	e := webserver.Start()

	wait := registerShutdown(context.Background(), 30*time.Second, map[string]operation{
		"containers": func(ctx context.Context) error {
			return docker.ShutdownContainers()
		},
		"webserver": func(ctx context.Context) error {
			return webserver.ShutdownServer(e)
		},
	})
	<-wait
}

func startSpout() {
	err := readConfiguration()
	if err != nil {
		logger.Error("Cannot open/read configuration", zap.Error(err))
		os.Exit(1)
	}

	startContainers()
}

func readConfiguration() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, "backend", "config", "spout-servers.json")

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
	path := filepath.Join(wd, "backend", "config", "spout-servers.json")

	jsonFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}

	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {

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

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		logger.Info(fmt.Sprintf("Running container %s", container.Names[0]), zap.String("image", container.Image), zap.String("containerShortId", container.ID[:10]))
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

type operation func(ctx context.Context) error

func registerShutdown(ctx context.Context, timeout time.Duration, ops map[string]operation) <-chan struct{} {
	wait := make(chan struct{})
	go func() {
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		<-s

		logger.Info("shutting down initiated")
		timeoutFunc := time.AfterFunc(timeout, func() {
			logger.Info(fmt.Sprintf("timeout %d ms has been elapsed, force exit", timeout.Milliseconds()))
			os.Exit(0)
		})

		defer timeoutFunc.Stop()

		var wg sync.WaitGroup
		for key, op := range ops {
			wg.Add(1)
			innerOp := op
			innerKey := key
			go func() {
				defer wg.Done()

				logger.Info(fmt.Sprintf("cleaning up: %s", innerKey))
				if err := innerOp(ctx); err != nil {
					logger.Error(err.Error())
					return
				}

				logger.Info(fmt.Sprintf("%s was shutdown gracefully", innerKey))
			}()
		}

		wg.Wait()

		close(wait)
	}()

	return wait
}
