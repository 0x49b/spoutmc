package watchdog

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/plugins"
	"spoutmc/internal/utils"
	"time"
)

var containerIds = []string{}

// Todo this has to be refatored with a client used for all different docker operations, currently against DRY principle
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var logger = log.GetLogger()
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

func ExcludeFromWatchdog(containerId string) {
	containerIds = append(containerIds, containerId)
	logger.Debug(fmt.Sprintf("[WatchDog] added %s", containerId))
}

func IncludeToWatchdog(containerId string) {
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

				logger.Debug(fmt.Sprintf("[WatchDog] Server %s in State %s", containerInfo.Config.Hostname, containerInfo.State.Status))

				// States: Can be one of "created", "running", "paused", "restarting", "removing", "exited", or "dead"
				if containerInfo.State.Status != "running" {

					// only restart container if not stoppeb by user
					if !utils.CheckInStringSlice(containerIds, containerInfo.ID) {
						logger.Warn(fmt.Sprintf("[WatchDog] detected container %s in state %s", containerInfo.Config.Hostname, containerInfo.State.Status))

						switch containerInfo.State.Status {
						case "exited":
						case "dead":
							startContainer(containerInfo.ID, containerInfo.Config.Hostname)
							break
						case "paused":
						}

						if containerInfo.State.Status == "exited" || containerInfo.State.Status == "dead" {
							startContainer(containerInfo.ID, containerInfo.Config.Hostname)
						}
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

func startContainer(containerId string, containerName string) {
	logger.Info(fmt.Sprintf("[WatchDog] try starting container %s", containerName))

	//checkForServerTapPlugin(containerId)

	err := cli.ContainerStart(ctx, containerId, container.StartOptions{})
	if err != nil {
		logger.Error("[WatchDog] Could not start container !!!")
	}
	logger.Info(fmt.Sprintf("[WatchDog] started container %s", containerName))
}

func checkForServerTapPlugin(containerId string) {

	c, _ := docker.GetContainerById(containerId)
	proxy, _ := docker.GetProxyContainer()

	// Do this plugin check only if it's not the proxy container
	if c.ID != proxy.ID {
		p, _ := plugins.CheckForServerTap(c.Mounts[0].Source)

		if !p {
			plugins.DownloadServerTap(c.Mounts[0].Source)
		}
	}
}
