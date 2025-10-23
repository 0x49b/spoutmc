package docker

import (
	"errors"
	"os"
	"os/exec"
	"spoutmc/internal/log"
	"sync"

	"go.uber.org/zap"

	"github.com/docker/docker/client"
)

var (
	cli  *client.Client
	once sync.Once
)

// createDockerClient creates the Docker Client as singleton
func createDockerClient() (*client.Client, error) {
	var err error

	if !isDockerRunning() {
		log.HandleError(errors.New("🐳 docker runtime is not running. Cannot start SpoutMC"))
		os.Exit(1)
	}
	once.Do(func() {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})
	return cli, err
}

func GetDockerClient() *client.Client {

	dockerClient, err := createDockerClient()
	if err != nil {
		logger.Error("Cannot create docker client", zap.Error(err))
		os.Exit(1)
	}

	return dockerClient
}

func isDockerRunning() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

// Create cli client on start of application
func init() {
	_, err := createDockerClient()
	if err != nil {
		logger.Error("docker client not available", zap.Error(err))
	}
}
