package docker

import (
	"go.uber.org/zap"
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
	once.Do(func() {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})

	return cli, err
}

// Create cli client on start of application
func init() {
	_, err := createDockerClient()
	if err != nil {
		logger.Error("docker client not available", zap.Error(err))
	}
}
