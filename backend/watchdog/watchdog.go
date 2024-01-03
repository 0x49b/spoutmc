package watchdog

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	"spoutmc/backend/utils"
	"time"
)

var containerIds = []string{}

// Todo this has to be refatored with a client used for all different docker operations, currently against DRY principle
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var logger = log.New()
var stopped = false

func Start() {
	stopped = false
	runWatchdog()
}

func Shutdown() error {
	logger.Info("Should now end watchdog")
	stopped = true
	return nil
}

func AddToWatchdog(containerId string) {
	containerIds = append(containerIds, containerId)
	logger.Debug(fmt.Sprintf("[WatchDog] added %s", containerId))
}

func RemoveFromWatchdog(containerId string) {
	containerIds = utils.Remove(containerIds, containerId)
	logger.Debug(fmt.Sprintf("[WatchDog] removed %s", containerId))
}

func runWatchdog() {
loop:
	for {
		if !stopped {
			networkContainer, err := docker.GetNetworkContainers()
			if err != nil {
				logger.Error("[Watchdog] Cannot find any Containers")
			}

			for _, container := range networkContainer {
				containerInfo, err := cli.ContainerInspect(ctx, container.ID)
				if err != nil {
					logger.Error("", zap.Error(err))
				}

				logger.Debug(fmt.Sprintf("[WatchDog] Container %s in State %s", containerInfo.Name, containerInfo.State.Status))

				// States: Can be one of "created", "running", "paused", "restarting", "removing", "exited", or "dead"
				if containerInfo.State.Status != "running" {

					logger.Error(fmt.Sprintf("[WatchDog] detected container %s in state %s", containerInfo.Name, containerInfo.State.Status))

					switch containerInfo.State.Status {
					case "exited":
					case "dead":
						startContainer(containerInfo.ID)
						break
					case "paused":
					}

					if containerInfo.State.Status == "exited" || containerInfo.State.Status == "dead" {
						startContainer(containerInfo.ID)
					}
				}
			}
		} else {
			break loop
		}

		// Sleep Time before Watchdog checks container
		time.Sleep(15 * time.Second)

	}
}

func startContainer(containerId string) {
	logger.Info(fmt.Sprintf("[WatchDog] try starting container %s", containerId))
	err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{})
	if err != nil {
		logger.Error("[WatchDog] Could not start container !!!")
	}
	logger.Info(fmt.Sprintf("[WatchDog] started container %s", containerId))
}
