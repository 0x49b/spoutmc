package watchdog

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"spoutmc/backend/log"
	"time"
)

var containerIds = []string{}

// Todo this has to be refatored with a client used for all different docker operations, currently against DRY principle
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var logger = log.New()

func Start() {
	runWatchdog()
}

func Shutdown() error {
	logger.Info("Should no end watchdog")
	return nil
}

func AddToWatchdog(containerId string) {
	containerIds = append(containerIds, containerId)
	logger.Info(fmt.Sprintf("[WatchDog] added %s", containerId))
}

func runWatchdog() {

	for {

		for _, container := range containerIds {
			containerInfo, err := cli.ContainerInspect(ctx, container)
			if err != nil {
				logger.Error("", zap.Error(err))
			}

			logger.Info(fmt.Sprintf("[WatchDog] Container %s in State %s", containerInfo.Name, containerInfo.State.Status))
			if containerInfo.State.Status != "running" {
				// Todo REstart container here if crashed
				logger.Error(fmt.Sprintf("[WatchDog] PANIC container %s in State %s", containerInfo.Name, containerInfo.State.Status))
			}

		}

		// Sleep Time before Watchdog checks container
		time.Sleep(5 * time.Second)
	}

}
