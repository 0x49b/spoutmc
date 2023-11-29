package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"spoutmc/internal/config"
	"spoutmc/pkg/dbcontext"
	"spoutmc/pkg/log"
	"spoutmc/web"
	"time"
)

var spoutConfiguration SpoutConfiguration
var logger = log.New()

func main() {
	printBanner()
	logger.Info("Starting SpoutNetwork")
	startSpout()
}

func startSpout() {
	err := readConfiguration()
	if err != nil {
		fmt.Println("Cannot open/read configuration")
		panic(err)
	}

	startWebServer()
}

func readConfiguration() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, "internal", "config", "spout-servers.json")

	fmt.Println(path)

	jsonFile, err := os.Open(path)
	if err != nil {
		return err
	}
	fmt.Println("Successfully opened configuration file")

	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &spoutConfiguration)
	if err != nil {
		return err
	}

	return nil
}

func startWebServer() {
	conf := config.New(os.Getenv("PORT"), os.Getenv("ENV"))

	e := echo.New()
	e.HideBanner = true
	app := conf.Bootstrap()

	e.Use(middleware.CORS())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{Timeout: 10 * time.Second}))
	e.Use(middleware.Secure())
	e.Use(middleware.Recover())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request", zap.String("URI", v.URI), zap.Int("status", v.Status), zap.String("method", v.Method), zap.Duration("latency", v.Latency))
			return nil
		},
	}))
	/*	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method} ${uri} ${status} ${latency_human} ${error}\n",
	}))*/
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // 20 request/sec rate limit

	registerHandler(e, app.Db)

	// Graceful shutdown
	go func() {
		if err := e.Start(":" + app.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Error(err)
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

}

func readServersToStart() (SpoutConfiguration, error) {
	wd, err := os.Getwd()
	if err != nil {
		return SpoutConfiguration{}, err
	}
	path := filepath.Join(wd, "internal", "config", "spout-servers.json")

	fmt.Println(path)

	jsonFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully opened spout-servers.json")

	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {

		}
	}(jsonFile)

	byteValue, _ := io.ReadAll(jsonFile)
	var spoutServers SpoutConfiguration
	err = json.Unmarshal(byteValue, &spoutServers)
	if err != nil {
		return SpoutConfiguration{}, err
	}
	return spoutServers, nil
}

func mapEnvironmentVariables(s SpoutServerEnv) []string {
	var containerEnv []string

	if s.Eula != "" {
		containerEnv = append(containerEnv, "EULA="+s.Eula)
	}
	if s.Type != "" {
		containerEnv = append(containerEnv, "TYPE="+s.Type)
	}
	if s.OnlineMode != "" {
		containerEnv = append(containerEnv, "ONLINE_MODE="+s.OnlineMode)
	}
	if s.EnforceSecureProfile != "" {
		containerEnv = append(containerEnv, "ENFORCE_SECURE_PROFILE="+s.EnforceSecureProfile)
	}
	if s.MaxMemory != "" {
		containerEnv = append(containerEnv, "MAX_MEMORY="+s.MaxMemory)
	}
	if s.Gui != "" {
		containerEnv = append(containerEnv, "GUI="+s.Gui)
	}
	if s.Console != "" {
		containerEnv = append(containerEnv, "CONSOLE="+s.Console)
	}
	if s.LogTimestamp != "" {
		containerEnv = append(containerEnv, "LOG_TIMESTAMP="+s.LogTimestamp)
	}
	if s.Tz != "" {
		containerEnv = append(containerEnv, "TZ="+s.Tz)
	}

	return containerEnv
}

func mapVolumeBindings(volumes []SpoutServerVolumes) []string {
	var spoutVolumes []string
	for _, v := range volumes {
		spoutVolumes = append(spoutVolumes, v.Hostpath+":"+v.Containerpath)
	}
	return spoutVolumes
}

func createSpoutNetwork() types.NetworkCreateResponse {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	spoutNetwork, err := cli.NetworkCreate(ctx, "spoutnetwork", types.NetworkCreate{Driver: "bridge"})
	if err != nil {
		panic(err)
	}

	return spoutNetwork
}

func mapExposedPorts(p SpoutServerPorts) (nat.PortSet, nat.PortMap) {
	var exposedPorts nat.PortSet
	var hostBinding nat.PortBinding
	var containerPortBinding nat.PortMap

	if (SpoutServerPorts{}) != p {

		exposedPorts = map[nat.Port]struct{}{nat.Port(p.ContainerPort + "/tcp"): {}}
		hostBinding = nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: p.HostPort,
		}
		containerPortBinding = nat.PortMap{
			nat.Port(p.ContainerPort + "/tcp"): []nat.PortBinding{hostBinding},
		}
	}

	return exposedPorts, containerPortBinding

}

func startContainers() {

	fmt.Println("Starting Containers")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	spoutServers, err := readServersToStart()
	spoutNetwork := createSpoutNetwork()

	if err != nil {
		panic(err)
	}

	for _, s := range spoutServers.Servers {
		fmt.Println(s.Name)

		fmt.Println("TESTING PORtS")
		fmt.Println(s.Ports)

		exposedPorts, containerPortBinding := mapExposedPorts(s.Ports)

		spoutContainer, err := cli.ContainerCreate(ctx, &container.Config{
			Image:        s.Image,
			Hostname:     s.Name,
			Env:          mapEnvironmentVariables(s.Env),
			ExposedPorts: exposedPorts,
		}, &container.HostConfig{
			Binds:        mapVolumeBindings(s.Volumes),
			PortBindings: containerPortBinding,
		}, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{spoutNetwork.ID: {NetworkID: spoutNetwork.ID}}},
			nil, s.Name)
		if err != nil {
			panic(err)
		}

		statusCh, errCh := cli.ContainerWait(ctx, spoutContainer.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case <-statusCh:
		}

		if err := cli.ContainerStart(ctx, spoutContainer.ID, types.ContainerStartOptions{}); err != nil {
			panic(err)
		}

		out, err := cli.ContainerLogs(ctx, spoutContainer.ID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			panic(err)
		}

		defer out.Close()
		stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}

}

func registerHandler(r *echo.Echo, db *dbcontext.DB) {
	web.RegisterHandlers(r)
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
