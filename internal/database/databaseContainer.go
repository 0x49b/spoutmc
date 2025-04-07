package database

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io/ioutil"
	"os"
	"path/filepath"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/utils"
	"time"
)

// Example docker command
//docker run --name some-mysql -e MYSQL_ROOT_PASSWORD=my-secret-pw -d mysql:tag

// Todo this has to be refatored with a client used for all different docker operations, currently against DRY principle
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var databaseContainerName = "spoutdatabase"
var databaseContainerImage = "mysql"
var logger = log.GetLogger()

// Todo remove this one, only to check
type Product struct {
	gorm.Model
	Code  string
	Price uint
}

func Start() {
	docker.PullImage(databaseContainerImage)
	containerId := ""
	if checkHasDatabaseContainer() {
		containerId = restartDatabaseContainer()
	} else {
		containerId = createDatabaseContainer()
	}
	go connectAndMigrate(containerId)
}

func connectAndMigrate(containerId string) {
	for {
		// Get container information
		containerInfo, err := cli.ContainerInspect(ctx, containerId)
		if err != nil {
			logger.Error("", zap.Error(err))
		}

		// Check if the container is in the "Running" state
		logger.Info(fmt.Sprintf("Database is %s. Connect and Migrate now", containerInfo.State.Status))

		if containerInfo.State.Status == "running" {
			time.Sleep(15 * time.Second) // Todo need here a timeout to let container fully start --> check for container status healthy or other
			ConnectDBThenMigrate()
			break
		}

		// Sleep for a short duration before checking again
		time.Sleep(1 * time.Second)
	}
}

func checkHasDatabaseContainer() bool {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", databaseContainerName)

	opts := container.ListOptions{All: true, Filters: containerFilter}
	containerList, err := cli.ContainerList(ctx, opts)
	if err != nil {
		return false
	}
	if len(containerList) > 0 {
		return true
	}
	return false
}

func restartDatabaseContainer() string {
	databaseContainer, err := docker.GetContainer(databaseContainerName)
	if err != nil {
		logger.Error(err.Error())
	}

	if databaseContainer.State == "exited" {
		err := cli.ContainerStart(ctx, databaseContainer.ID, container.StartOptions{})
		if err != nil {
			logger.Error(err.Error())
		}
	} else {
		err := cli.ContainerRestart(ctx, databaseContainer.ID, container.StopOptions{})
		if err != nil {
			logger.Error(err.Error())
		}
	}

	return databaseContainer.ID
}

func writeDBPassword(password string) string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	path := filepath.Join(wd, ".msql_password")

	file, err := os.Create(path)
	if err != nil {
		logger.Error("", zap.Error(err))
	}
	defer file.Close() // Ensure the file is closed when the function returns.

	// Data to be written to the file
	data := []byte(password)

	// Write data to the file
	_, err = file.Write(data)
	if err != nil {
		logger.Error("", zap.Error(err))
	}
	logger.Info(fmt.Sprintf("Find your DB Password in %s. Do not forget to delete the file from the filesystem", path))
	return path
}

func GetDbPasswordIfExists() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	path := filepath.Join(wd, ".msql_password")

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil

}

func createDatabaseContainer() string {
	logger.Info(fmt.Sprintf("Creating database container %s", databaseContainerName))
	rootPassword, err := GetDbPasswordIfExists()
	if err != nil {
		rootPassword = utils.RandomString(25)
		writeDBPassword(rootPassword)
	}

	exposedPorts, containerPortBinding := docker.MapExposedPorts(models.SpoutServerPorts{
		HostPort:      "3306",
		ContainerPort: "3306",
	})
	spoutNetwork := docker.GetSpoutNetwork()
	containerLabels := map[string]string{
		"io.spout.servername": databaseContainerName,
		"io.spout.network":    "true",
	}

	// Todo change password to MYSQL_PASSWORD_FILE and mount that file
	databaseContainer, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        databaseContainerImage,
		Hostname:     databaseContainerName,
		Env:          []string{fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", rootPassword), fmt.Sprintf("MYSQL_PASSWORD=%s", rootPassword), "MYSQL_USER=spoutdbuser", "MYSQL_DATABASE=spout"},
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

	if err := cli.ContainerStart(ctx, databaseContainer.ID, container.StartOptions{}); err != nil {
		logger.Error("Cannot start database container", zap.Error(err))
	}

	return databaseContainer.ID

}

func Shutdown() error {

	databaseContainer, err := docker.GetContainer(databaseContainerName)
	if err != nil {
		logger.Error(err.Error())
	}

	err = cli.ContainerStop(ctx, databaseContainer.ID, container.StopOptions{})
	if err != nil {
		logger.Error(err.Error())
	}

	return nil
}
