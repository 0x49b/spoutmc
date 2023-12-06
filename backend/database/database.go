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
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"os"
	"path/filepath"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	"spoutmc/backend/models"
	"spoutmc/backend/utils"
	"spoutmc/backend/watchdog"
	"time"
)

// Example docker command
//docker run --name some-mysql -e MYSQL_ROOT_PASSWORD=my-secret-pw -d mysql:tag

// Todo this has to be refatored with a client used for all different docker operations, currently against DRY principle
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var databaseContainerName = "spoutdatabase"
var databaseContainerImage = "mysql"
var logger = log.New()

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

	watchdog.AddToWatchdog(containerId)

	go migrateDatabase(containerId)

}

func migrateDatabase(containerId string) {
	for {
		// Get container information
		containerInfo, err := cli.ContainerInspect(ctx, containerId)
		if err != nil {
			logger.Error("", zap.Error(err))
		}

		// Check if the container is in the "Running" state
		logger.Info(fmt.Sprintf("Database Status %s", containerInfo.State.Status))

		if containerInfo.State.Status == "running" {
			time.Sleep(5 * time.Second)        // Todo need here a
			pw, err := getDbPasswordIfExists() // Todo extend this wit configurtion properties or ENV Var
			if err != nil {
				logger.Error("Cannot get DatabasePassword from File")
			}

			dsn := fmt.Sprintf("spoutdbuser:%s@tcp(127.0.0.1:3306)/spout?charset=utf8mb4&parseTime=True&loc=Local", pw)
			db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
			if err != nil {
				logger.Error("", zap.Error(err))
			}
			// Migrate the schema
			err = db.AutoMigrate(&Product{})
			if err != nil {
				logger.Error("", zap.Error(err))
			}
			logger.Info("Applied migrations to Database")
			break
		}

		// Sleep for a short duration before checking again
		time.Sleep(1 * time.Second)
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

func restartDatabaseContainer() string {
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

func getDbPasswordIfExists() (string, error) {
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
	rootPassword, err := getDbPasswordIfExists()
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

	if err := cli.ContainerStart(ctx, databaseContainer.ID, types.ContainerStartOptions{}); err != nil {
		logger.Error("Cannot start database container", zap.Error(err))
	}

	return databaseContainer.ID

}

func Shutdown() error {

	databaseContainer := docker.GetContainer(databaseContainerName)

	err := cli.ContainerStop(ctx, databaseContainer.ID, container.StopOptions{})
	if err != nil {
		logger.Error(err.Error())
	}

	return nil
}
