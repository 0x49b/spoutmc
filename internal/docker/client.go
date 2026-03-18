package docker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"spoutmc/internal/log"
	"sync"

	"github.com/docker/docker/client"
)

var (
	cli     *client.Client
	initErr error
	once    sync.Once
)

// createDockerClient creates the Docker Client as singleton
func createDockerClient() (*client.Client, error) {
	once.Do(func() {
		if !isDockerRunning() {
			log.HandleError(errors.New("🐳 docker or podman runtime is not running. Cannot start SpoutMC"))
			os.Exit(1)
		}

		// Set DOCKER_HOST for rootless podman if it exists and DOCKER_HOST is not set
		if os.Getenv("DOCKER_HOST") == "" {
			uid := os.Getuid()
			if uid != 0 { // Rootless
				podmanSock := fmt.Sprintf("/run/user/%d/podman/podman.sock", uid)
				if _, err := os.Stat(podmanSock); err == nil {
					os.Setenv("DOCKER_HOST", "unix://"+podmanSock)
				}
			} else { // Rootful
				podmanSock := "/run/podman/podman.sock"
				if _, err := os.Stat(podmanSock); err == nil {
					os.Setenv("DOCKER_HOST", "unix://"+podmanSock)
				}
			}
		}

		cli, initErr = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})
	return cli, initErr
}

func GetDockerClient() *client.Client {
	dockerClient, err := createDockerClient()
	if err != nil {
		log.HandleError(fmt.Errorf("cannot create docker client: %w", err))
		os.Exit(1)
	}

	return dockerClient
}

func isDockerRunning() bool {
	// First check podman
	cmd := exec.Command("podman", "version")
	err := cmd.Run()
	if err == nil {
		return true
	}
	// Fall back to docker
	cmd = exec.Command("docker", "version")
	err = cmd.Run()
	if err == nil {
		return true
	}
	return false
}

// Create cli client on start of application
func init() {
	_, err := createDockerClient()
	if err != nil {
		log.HandleError(fmt.Errorf("docker client not available: %w", err))
	}
}
