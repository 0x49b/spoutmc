package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"io"
	"slices"
	"spoutmc/backend/log"
	"spoutmc/backend/models"
)

// ALways run Docker commands in Background Context
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var logger = log.New()

func PullImage(imageName string) {

	logger.Info("Pulling/Checking image ", zap.String("imageName", imageName))
	pull, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return
	}
	defer pull.Close()
	if _, err := io.ReadAll(pull); err != nil {
		logger.Error("Cannot pull image", zap.Error(err))
	}
}

func CreateSpoutNetwork(networkName string) types.NetworkResource {

	networkList, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return types.NetworkResource{}
	}

	var availableNetworks []string

	for _, n := range networkList {
		availableNetworks = append(availableNetworks, n.Name)
	}

	if !slices.Contains(availableNetworks, networkName) {
		spoutNetwork, err := cli.NetworkCreate(ctx, networkName, types.NetworkCreate{Driver: "bridge"})
		if err != nil {
			logger.Error("Cannot create network", zap.Error(err))
		}
		return types.NetworkResource{ID: spoutNetwork.ID, Name: networkName}
	} else {
		for _, n := range networkList {
			if networkName == n.Name {
				return n
			}
		}
	}

	return types.NetworkResource{}
}

func getSpoutNetwork() types.NetworkResource {
	networkList, _ := cli.NetworkList(ctx, types.NetworkListOptions{})
	networkName := "spoutnetwork" // todo get this from config
	for _, n := range networkList {
		if networkName == n.Name {
			return n
		}
	}

	return CreateSpoutNetwork(networkName)
}

func containerExists(containerName string) bool {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, _ := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: containerFilter})
	return len(containerList) > 0
}

func getContainer(containerName string) types.Container {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, _ := cli.ContainerList(ctx, types.ContainerListOptions{Filters: containerFilter})
	return containerList[0]
}

func StartContainer(s models.SpoutServer) {
	// Pull Image for Container
	PullImage(s.Image)

	if !containerExists(s.Name) {
		exposedPorts, containerPortBinding := MapExposedPorts(s.Ports)
		spoutNetwork := getSpoutNetwork()

		spoutContainer, err := cli.ContainerCreate(ctx, &container.Config{
			Image:        s.Image,
			Hostname:     s.Name,
			Env:          MapEnvironmentVariables(s.Env),
			ExposedPorts: exposedPorts,
		}, &container.HostConfig{
			Binds:        MapVolumeBindings(s.Volumes),
			PortBindings: containerPortBinding,
		}, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{spoutNetwork.ID: {NetworkID: spoutNetwork.ID}}},
			nil, s.Name)
		if err != nil {
			panic(err)
		}

		statusCh, errCh := cli.ContainerWait(ctx, spoutContainer.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case <-statusCh:
		}

		if err := cli.ContainerStart(ctx, spoutContainer.ID, types.ContainerStartOptions{}); err != nil {
			logger.Error("Cannot start container", zap.Error(err))
		}
	} else {

		//todo check for configuration switch here if it should restart

		logger.Info(fmt.Sprintf("restart container %s", s.Name))
		restartContainer := getContainer(s.Name)
		err := cli.ContainerRestart(ctx, restartContainer.ID, container.StopOptions{})
		if err != nil {
			zap.Error(err)
		}
	}

}
