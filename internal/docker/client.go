package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"sync"
	"time"

	"github.com/docker/docker/client"
)

var (
	cli     *client.Client
	initErr error
	once    sync.Once
)

func createDockerClient() (*client.Client, error) {
	once.Do(func() {
		if os.Getenv("DOCKER_HOST") == "" {
			uid := os.Getuid()
			xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")

			candidates := []string{
				fmt.Sprintf("/run/user/%d/podman/podman.sock", uid),
				"/run/podman/podman.sock",
			}
			if xdgRuntimeDir != "" {
				candidates = append(candidates, filepath.Join(xdgRuntimeDir, "podman", "podman.sock"))
			}

			for _, sock := range candidates {
				if _, err := os.Stat(sock); err == nil {
					os.Setenv("DOCKER_HOST", "unix://"+sock)
					break
				}
			}
		}

		cli, initErr = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

		if initErr != nil || cli == nil {
			if initErr == nil {
				initErr = errors.New("docker client is nil after initialization")
			}
			log.HandleError(fmt.Errorf("cannot create docker client: %w", initErr))
			os.Exit(1)
		}

		start := time.Now()
		for attempt := 0; time.Since(start) < 25*time.Second; attempt++ {
			attemptCtx, attemptCancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, pingErr := cli.Ping(attemptCtx)
			attemptCancel()

			initErr = pingErr

			if pingErr == nil {
				break
			}
		}

		if initErr != nil {
			log.HandleError(errors.New("🐳 docker/podman runtime is not reachable (ping failed after retries). Cannot start SpoutMC"))
			os.Exit(1)
		}
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

func init() {
	_, err := createDockerClient()
	if err != nil {
		log.HandleError(fmt.Errorf("docker client not available: %w", err))
	}
}
