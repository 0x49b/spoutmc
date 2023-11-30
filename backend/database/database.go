package database

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	"spoutmc/backend/models"
	"spoutmc/backend/utils"
)

// Example docker command
//docker run --name some-mysql -e MYSQL_ROOT_PASSWORD=my-secret-pw -d mysql:tag

// Todo this has to be refatored with a client used for all different docker operations, currently against DRY principle
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var databaseContainerName = "spoutdatabase"
var databaseContainerImage = "mysql"
var logger = log.New()

func Start() {

	docker.PullImage(databaseContainerImage)

	if checkHasDatabaseContainer() {
		restartDatabaseContainer()
	} else {
		createDatabaseContainer()
	}
}

func checkHasDatabaseContainer() bool {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", databaseContainerName)

	opts := types.ContainerListOptions{All: true, Filters: containerFilter}
	containerList, err := cli.ContainerList(ctx, opts)
	if err != nil {
		return false
	}
	if len(containerList) > 0 {
		return true
	}
	return false
}

func restartDatabaseContainer() {
	databaseContainer := docker.GetContainer(databaseContainerName)

	if databaseContainer.State == "exited" {
		err := cli.ContainerStart(ctx, databaseContainer.ID, types.ContainerStartOptions{})
		if err != nil {
			logger.Error(err.Error())
		}
	} else {
		err := cli.ContainerRestart(ctx, databaseContainer.ID, container.StopOptions{})
		if err != nil {
			logger.Error(err.Error())
		}
	}
}

func writeDBPassword(password string) {
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	path := filepath.Join(wd, ".msql_password")

	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed when the function returns.

	// Data to be written to the file
	data := []byte(password)

	// Write data to the file
	_, err = file.Write(data)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	logger.Info(fmt.Sprintf("Find your DB Password in %s. Do not forget to delete the file from the filesystem", path))
}

func createDatabaseContainer() {
	logger.Info(fmt.Sprintf("Creating database container %s"))
	rootPassword := utils.RandomString(25)
	writeDBPassword(rootPassword)

	exposedPorts, containerPortBinding := docker.MapExposedPorts(models.SpoutServerPorts{
		HostPort:      "3306",
		ContainerPort: "3306",
	})
	spoutNetwork := docker.GetSpoutNetwork()
	containerLabels := map[string]string{
		"io.spout.servername": databaseContainerName,
		"io.spout.network":    "true",
	}

	databaseContainer, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        databaseContainerImage,
		Hostname:     databaseContainerName,
		Env:          []string{fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", rootPassword)},
		ExposedPorts: exposedPorts,
		Labels:       containerLabels,
	}, &container.HostConfig{
		PortBindings: containerPortBinding,
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{spoutNetwork.ID: {NetworkID: spoutNetwork.ID}}},
		nil, databaseContainerName)
	if err != nil {
		logger.Error("", zap.Error(err))
	}

	if err := cli.ContainerStart(ctx, databaseContainer.ID, types.ContainerStartOptions{}); err != nil {
		logger.Error("Cannot start database container", zap.Error(err))
	}

}

func Shutdown() error {

	databaseContainer := docker.GetContainer(databaseContainerName)

	err := cli.ContainerStop(ctx, databaseContainer.ID, container.StopOptions{})
	if err != nil {
		logger.Error(err.Error())
	}

	return nil
}
