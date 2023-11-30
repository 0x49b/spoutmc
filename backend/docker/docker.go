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

func GetSpoutNetwork() types.NetworkResource {
	networkList, _ := cli.NetworkList(ctx, types.NetworkListOptions{})
	networkName := "spoutnetwork" // todo get this from config
	for _, n := range networkList {
		if networkName == n.Name {
			return n
		}
	}

	return CreateSpoutNetwork(networkName)
}

func getNetworkContainers() ([]types.Container, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("label", "io.spout.network=true")

	list, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: containerFilter})
	if err != nil {
		return []types.Container{}, err
	}

	return list, nil
}

// todo check this and the below function, it's against DRY
func containerExists(containerName string) bool {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, _ := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: containerFilter})
	return len(containerList) > 0
}

func GetContainer(containerName string) types.Container {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, _ := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: containerFilter})
	return containerList[0]
}

func StartContainer(s models.SpoutServer) {
	// Pull Image for Container
	PullImage(s.Image)

	if !containerExists(s.Name) {
		logger.Info(fmt.Sprintf("Creating container %s", s.Name))
		exposedPorts, containerPortBinding := MapExposedPorts(s.Ports)
		spoutNetwork := GetSpoutNetwork()

		spoutContainer, err := cli.ContainerCreate(ctx, &container.Config{
			Image:        s.Image,
			Hostname:     s.Name,
			Env:          MapEnvironmentVariables(s.Env),
			ExposedPorts: exposedPorts,
			Labels:       map[string]string{"io.spout.servername": s.Name, "io.spout.network": "true"},
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
			logger.Info(fmt.Sprintf("container %s created", s.Name))
		}

		if err := cli.ContainerStart(ctx, spoutContainer.ID, types.ContainerStartOptions{}); err != nil {
			logger.Error("Cannot start container", zap.Error(err))
		}
	} else {

		//todo check for configuration switch here if it should restart

		logger.Info(fmt.Sprintf("re/start container %s", s.Name))
		startContainer := GetContainer(s.Name)

		if startContainer.State == "exited" {
			err := cli.ContainerStart(ctx, startContainer.ID, types.ContainerStartOptions{})
			if err != nil {
				logger.Error(err.Error())
			}
		} else {
			err := cli.ContainerRestart(ctx, startContainer.ID, container.StopOptions{})
			if err != nil {
				logger.Error(err.Error())
			}
		}
	}

}

func ShutdownContainers() error {

	containers, err := getNetworkContainers()
	if err != nil {
		return err
	}

	for _, c := range containers {
		logger.Info(fmt.Sprintf("shutting down container %s", c.Names[0]))
		err := cli.ContainerStop(ctx, c.ID, container.StopOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
