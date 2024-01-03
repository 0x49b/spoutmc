package docker

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"io"
	"os"
	"spoutmc/backend/log"
	"spoutmc/backend/models"
)

// ALways run Docker commands in Background Context
var ctx = context.Background()
var cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

var logger = log.New()

func StreamLogsFromContainer(containerName string) {

	cid, err := GetContainer(containerName)
	if err != nil {
		return
	}

	i, err := cli.ContainerLogs(context.Background(), cid.ID, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
		Follow:     true,
		Tail:       "40",
	})
	if err != nil {
		logger.Error("", zap.Error(err))
	}
	hdr := make([]byte, 8)
	for {
		_, err := i.Read(hdr)
		if err != nil {
			logger.Error("", zap.Error(err))
		}
		var w io.Writer
		switch hdr[0] {
		case 1:
			w = os.Stdout
		default:
			w = os.Stderr
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, err = i.Read(dat)
		fmt.Fprint(w, string(dat))
	}

}

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

func GetNetworkContainers() ([]types.Container, error) {
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

func GetContainer(containerName string) (types.Container, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: containerFilter})

	if err != nil {
		return types.Container{}, err
	}

	if len(containerList) < 1 {
		return types.Container{}, errors.New(fmt.Sprintf("Cannot find container for name %s", containerName))
	}

	return containerList[0], nil
}

func GetContainerById(containerId string) (types.ContainerJSON, error) {
	requestedContainer, err := cli.ContainerInspect(ctx, containerId)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return requestedContainer, nil
}

func getHostNetworkId() (types.NetworkResource, error) {
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return types.NetworkResource{}, err
	}

	for _, n := range networks {
		if n.Name == "bridge" {
			return n, nil
		}
	}

	return types.NetworkResource{}, nil
}

func StartContainer(s models.SpoutServer) {

	// Pull Image for Container
	PullImage(s.Image)

	if !containerExists(s.Name) {
		logger.Info(fmt.Sprintf("Creating container %s", s.Name))
		exposedPorts, containerPortBinding := MapExposedPorts(s.Ports)
		spoutNetwork := GetSpoutNetwork()
		hostNetwork, err := getHostNetworkId()
		if err != nil {
			logger.Error("", zap.Error(err))
		}

		endpoints := make(map[string]*network.EndpointSettings, 2)
		endpoints[spoutNetwork.ID] = &network.EndpointSettings{EndpointID: spoutNetwork.ID}
		endpoints[hostNetwork.ID] = &network.EndpointSettings{EndpointID: hostNetwork.ID}

		var containerLabels map[string]string
		containerLabels = make(map[string]string)

		containerLabels["io.spout.servername"] = s.Name
		containerLabels["io.spout.network"] = "true"

		if s.Proxy {
			containerLabels["io.spout.proxy"] = "true"
		}
		if s.Lobby {
			containerLabels["io.spout.lobby"] = "true"
		}

		spoutContainer, err := cli.ContainerCreate(ctx, &container.Config{
			Tty:          true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        s.Image,
			Hostname:     s.Name,
			Env:          MapEnvironmentVariables(s.Env),
			ExposedPorts: exposedPorts,
			Labels:       containerLabels,
		}, &container.HostConfig{
			Binds:        MapVolumeBindings(s.Volumes),
			PortBindings: containerPortBinding,
		}, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{spoutNetwork.ID: {NetworkID: spoutNetwork.ID}}},
			nil, s.Name)
		if err != nil {
			logger.Error("Error creating container", zap.Error(err))
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
		startContainer, err := GetContainer(s.Name)
		if err != nil {
			logger.Error(err.Error())
		}

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

	containers, err := GetNetworkContainers()
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

func StopContainerById(containerId string) {
	if err := cli.ContainerStop(ctx, containerId, container.StopOptions{}); err != nil {
		logger.Error("Cannot stop container", zap.Error(err))
	}

}

func StartContainerById(containerId string) {
	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		logger.Error("Cannot start container", zap.Error(err))
	}
}

func RestartContainerById(containerId string) {
	if err := cli.ContainerRestart(ctx, containerId, container.StopOptions{}); err != nil {
		logger.Error("Cannot start container", zap.Error(err))
	}
}

func filterForContainerLabel(label string) (types.Container, error) {
	networkContainer, err := GetNetworkContainers()
	if err != nil {
		return types.Container{}, err
	}

	for _, nc := range networkContainer {
		containerDetails, err := cli.ContainerInspect(ctx, nc.ID)
		if err != nil {
			return types.Container{}, err
		}
		_, check := containerDetails.Config.Labels[label]

		if check {
			return nc, nil
		}
	}
	return types.Container{}, errors.New(fmt.Sprintf("no Container found for label %s", label))
}

func GetProxyContainer() (types.Container, error) {
	proxyContainer, err := filterForContainerLabel("io.spout.proxy")
	if err != nil {
		return types.Container{}, nil
	}
	return proxyContainer, nil
}

func GetLobbyContainer() (types.Container, error) {
	lobbyContainer, err := filterForContainerLabel("io.spout.lobby")
	if err != nil {
		return types.Container{}, nil
	}
	return lobbyContainer, nil
}
