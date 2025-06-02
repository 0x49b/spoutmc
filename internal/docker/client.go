package docker

import (
	"errors"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"spoutmc/internal/log"
	"sync"

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

func GetDockerClient() (*client.Client, error) {
	return createDockerClient()
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
