package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

// ALways run Docker commands in Background Context
var ctx = context.Background()
var logger = log.GetLogger(log.ModuleDocker)

func PullImage(imageName string) {

	logger.Info("Pulling ", zap.String("imageName", imageName))
	pull, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return
	}
	defer func(pull io.ReadCloser) {
		err := pull.Close()
		if err != nil {
			logger.Error("Cannot close image pull", zap.Error(err))
		}
	}(pull)
	if _, err := io.ReadAll(pull); err != nil {
		logger.Error("Cannot pull image", zap.Error(err))
	}

}

func GetNetworkContainers() ([]container.Summary, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("label", "io.spout.network=true")
	list, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: containerFilter})
	if err != nil {
		return []container.Summary{}, err
	}

	// Filter out infrastructure containers
	serverContainers := make([]container.Summary, 0)
	for _, c := range list {
		// Skip if this is an infrastructure container
		if value, exists := c.Labels["io.spout.infrastructure"]; exists && value == "true" {
			continue
		}
		serverContainers = append(serverContainers, c)
	}

	return serverContainers, nil
}

// todo check this and the below function, it's against DRY
func containerExists(containerName string) bool {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, _ := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: containerFilter})
	return len(containerList) > 0
}

func GetContainer(containerName string) (container.Summary, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: containerFilter})

	if err != nil {
		return container.Summary{}, err
	}

	if len(containerList) < 1 {
		return container.Summary{}, errors.New(fmt.Sprintf("Cannot find container for name %s", containerName))
	}

	return containerList[0], nil
}

func GetContainerById(containerId string) (container.InspectResponse, error) {
	requestedContainer, err := cli.ContainerInspect(ctx, containerId)
	if err != nil {
		return container.InspectResponse{}, err
	}
	return requestedContainer, nil
}

func GetContainerStats(containerId string) (container.StatsResponse, error) {

	stats, err := cli.ContainerStats(context.Background(), containerId, false)
	if err != nil {
		return container.StatsResponse{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("Cannot close container stats", zap.Error(err))
		}
	}(stats.Body)
	var statsResponse container.StatsResponse
	decoder := json.NewDecoder(stats.Body)
	if err := decoder.Decode(&statsResponse); err != nil && err != io.EOF {
		return container.StatsResponse{}, err
	}
	return statsResponse, nil
}

func getHostNetworkId() (network.Inspect, error) {
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return network.Inspect{}, err
	}

	for _, n := range networks {
		if n.Name == "bridge" {
			return n, nil
		}
	}

	return network.Inspect{}, nil
}

func StartContainer(s models.SpoutServer, dataPath string) error {

	// Check if image exists, if not pull it
	_, err := cli.ImageInspect(context.Background(), s.Image)
	if err != nil {
		// Pull Image for Server
		PullImage(s.Image)
	}

	// Pre-create volume directories with proper user ownership
	// This prevents Docker from creating them as root on Linux
	if err := ensureVolumeDirectoriesExist(s.Volumes, dataPath, s.Name); err != nil {
		logger.Error("Failed to create volume directories", zap.Error(err))
		return fmt.Errorf("failed to create volume directories: %w", err)
	}

	if !containerExists(s.Name) {
		logger.Info(fmt.Sprintf("Creating container %s", s.Name))
		exposedPorts, containerPortBinding := nat.PortSet{}, nat.PortMap{}
		if s.Proxy {
			exposedPorts, containerPortBinding = MapExposedPorts(s.Ports)
		}

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

		envVars := MapEnvironmentVariables(s.Env)

		// Prepare volume bindings
		binds := MapVolumeBindings(s.Volumes, dataPath, s.Name)

		// For proxy servers, mount the forwarding.secret file
		// The itzg/mc-proxy image copies files from /config to /server on startup
		if s.Proxy {
			// Ensure the forwarding.secret file exists BEFORE mounting it
			// If we try to mount a non-existent file, Docker creates it as a directory
			secretSourcePath := filepath.Join(dataPath, s.Name, "server", "forwarding.secret")
			if err := ensureForwardingSecret(secretSourcePath); err != nil {
				logger.Warn("Failed to create forwarding.secret file before mounting",
					zap.Error(err),
					zap.String("path", secretSourcePath))
				// Continue anyway - the file might be created by velocity.toml sync
			}

			secretTargetPath := "/config/forwarding.secret"
			binds = append(binds, fmt.Sprintf("%s:%s:ro", secretSourcePath, secretTargetPath))
			logger.Info("Mounting Velocity forwarding secret",
				zap.String("source", secretSourcePath),
				zap.String("target", secretTargetPath))
		}

		spoutContainer, err := cli.ContainerCreate(ctx, &container.Config{
			Tty:          true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        s.Image,
			Hostname:     s.Name,
			Env:          envVars,
			ExposedPorts: exposedPorts,
			Labels:       containerLabels,
		}, &container.HostConfig{
			Binds:        binds,
			PortBindings: containerPortBinding,
		}, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{spoutNetwork.ID: {NetworkID: spoutNetwork.ID}}},
			nil, s.Name)
		if err != nil {
			logger.Error("Error creating container", zap.Error(err))
			return fmt.Errorf("failed to create container: %w", err)
		}

		statusCh, errCh := cli.ContainerWait(ctx, spoutContainer.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("container wait error: %w", err)
			}
		case <-statusCh:
			logger.Info(fmt.Sprintf("container %s created", s.Name))
		}

		if err := cli.ContainerStart(ctx, spoutContainer.ID, container.StartOptions{}); err != nil {
			logger.Error("Cannot start container", zap.Error(err))
			return fmt.Errorf("failed to start container: %w", err)
		}
	} else {
		startContainer, err := GetContainer(s.Name)
		if err != nil {
			logger.Error(err.Error())
			return fmt.Errorf("failed to get container: %w", err)
		}

		if startContainer.State == "exited" {
			err := cli.ContainerStart(ctx, startContainer.ID, container.StartOptions{})
			logger.Info(fmt.Sprintf("⛏️ start container %s", s.Name))
			if err != nil {
				logger.Error(err.Error())
				return fmt.Errorf("failed to start existing container: %w", err)
			}
		} else {
			err := cli.ContainerRestart(ctx, startContainer.ID, container.StopOptions{})
			logger.Info(fmt.Sprintf("⛏️ restart container %s", s.Name))
			if err != nil {
				logger.Error(err.Error())
				return fmt.Errorf("failed to restart container: %w", err)
			}
		}
	}

	return nil
}

func ShutdownContainers() error {
	containers, err := GetNetworkContainers()
	if err != nil {
		return err
	}

	for _, c := range containers {
		logger.Info(fmt.Sprintf("⛏️ shutting down container %s (%s)",
			strings.Trim(c.Names[0], "/"),
			c.ID[:10]))
		err := cli.ContainerStop(ctx, c.ID, container.StopOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func StopContainerById(containerId string) {
	if err := cli.ContainerStop(ctx, containerId, container.StopOptions{Signal: "SIGKILL"}); err != nil {
		logger.Error("Cannot stop container", zap.Error(err))
	}

}

func StartContainerById(containerId string) {
	if err := cli.ContainerStart(ctx, containerId, container.StartOptions{}); err != nil {
		logger.Error("Cannot start container", zap.Error(err))
	}
}

func RestartContainerById(containerId string) {
	if err := cli.ContainerRestart(ctx, containerId, container.StopOptions{}); err != nil {
		logger.Error("Cannot start container", zap.Error(err))
	}
}

func GetProxyContainer() (container.Summary, error) {
	proxyContainer, err := filterForContainerLabel("io.spout.proxy")
	if err != nil {
		return container.Summary{}, err
	}
	return proxyContainer, nil
}

func GetLobbyContainer() (container.Summary, error) {
	lobbyContainer, err := filterForContainerLabel("io.spout.lobby")
	if err != nil {
		return container.Summary{}, err
	}
	return lobbyContainer, nil
}

func GetProxyVolumeMount() string {
	proxyContainer, err := GetProxyContainer()
	if err != nil {
		return ""
	}
	proxyContainerInspect, err := cli.ContainerInspect(ctx, proxyContainer.ID)
	if err != nil {
		return ""
	}
	return proxyContainerInspect.Mounts[0].Source
}

func GetProxyConfigFilePath() string {
	return filepath.Join(GetProxyVolumeMount(), "velocity.toml")
}

func filterForContainerLabel(label string) (container.Summary, error) {
	networkContainer, err := GetNetworkContainers()
	if err != nil {
		return container.Summary{}, err
	}

	for _, nc := range networkContainer {
		containerDetails, err := cli.ContainerInspect(ctx, nc.ID)
		if err != nil {
			return container.Summary{}, err
		}
		_, check := containerDetails.Config.Labels[label]

		if check {
			return nc, nil
		}
	}
	return container.Summary{}, errors.New(fmt.Sprintf("no Server found for label %s", label))
}

func getContainerNameById(containerId string) string {
	interestedContainer, err := GetContainerById(containerId)
	if err != nil {
		logger.Error("cannot load container", zap.Error(err))
	}
	return interestedContainer.Name
}

// fetchDockerLogs retrieves logs from the Docker container
func FetchDockerLogs(ctx context.Context, id string) (<-chan string, error) {
	options := container.LogsOptions{ShowStdout: true, Follow: true, Tail: "1000"}

	reader, err := cli.ContainerLogs(ctx, getContainerNameById(id), options)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve logs: %v", err)
	}

	logChan := make(chan string)

	go func() {
		defer reader.Close()
		defer close(logChan)

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()

			// Filter out lines that only contain ">" or are effectively empty
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || trimmed == ">" {
				continue
			}

			logChan <- line
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("error reading logs: %v\n", err)
		}
	}()

	return logChan, nil
}

func RemoveLocalVolumeDataForContainer(containerId string) {
	containerVolume, err := GetContainerById(containerId)
	if err != nil {
		logger.Error("cannot load container", zap.Error(err))
	}

	for _, c := range containerVolume.Mounts {
		logger.Info(c.Source)
		dir, err := os.ReadDir(c.Source)
		if err != nil {
			logger.Error("cannot read dir", zap.Error(err))
		}
		for _, d := range dir {
			err := os.RemoveAll(filepath.Join(c.Source, d.Name()))
			if err != nil {
				logger.Error("cannot remove volume", zap.Error(err))
			}
		}
		err = os.RemoveAll(c.Source)
		if err != nil {
			logger.Error("cannot remove volume", zap.Error(err))
		}
	}
}

func RemoveContainerById(containerId string, removeVolume bool) error {
	if removeVolume {
		RemoveLocalVolumeDataForContainer(containerId)
	}
	return cli.ContainerRemove(ctx, containerId, container.RemoveOptions{
		RemoveVolumes: removeVolume,
	})
}

func RecreateContainer(containerConfig models.SpoutServer, dataPath string) error {
	logger.Info("Recreating container", zap.String("containerName", containerConfig.Name))
	recreateContainer, err := GetContainer(containerConfig.Name)
	if err != nil {
		return err
	}
	StopContainerById(recreateContainer.ID)
	err = RemoveContainerById(recreateContainer.ID, false)
	if err != nil {
		return err
	}
	StartContainer(containerConfig, dataPath)
	return nil
}

func StopAndRemoveContainerById(containerId string) error {
	StopContainerById(containerId)
	return RemoveContainerById(containerId, true)
}

// CreateInfrastructureContainer creates a container with infrastructure labels
func CreateInfrastructureContainer(s models.SpoutServer, dataPath string) (string, error) {
	logger.Info("Creating infrastructure container", zap.String("name", s.Name))

	// Check if image exists, if not pull it
	_, err := cli.ImageInspect(context.Background(), s.Image)
	if err != nil {
		logger.Info("Pulling infrastructure image", zap.String("image", s.Image))
		PullImage(s.Image)
	}

	// Check if container already exists
	if containerExists(s.Name) {
		existingContainer, err := GetContainer(s.Name)
		if err != nil {
			return "", err
		}
		logger.Info("Infrastructure container already exists", zap.String("name", s.Name))
		return existingContainer.ID, nil
	}

	// Pre-create volume directories with proper user ownership
	// This prevents Docker from creating them as root on Linux
	if err := ensureVolumeDirectoriesExist(s.Volumes, dataPath, s.Name); err != nil {
		return "", fmt.Errorf("failed to create volume directories: %w", err)
	}

	// Map exposed ports and port bindings
	exposedPorts, containerPortBinding := MapExposedPorts(s.Ports)

	// Get spout network
	spoutNetwork := GetSpoutNetwork()

	// Create container labels
	containerLabels := map[string]string{
		"io.spout.servername":     s.Name,
		"io.spout.network":        "true",
		"io.spout.infrastructure": "true",
		"io.spout.database":       "true",
	}

	// Map environment variables
	envVars := MapEnvironmentVariables(s.Env)

	// Prepare volume bindings
	binds := MapVolumeBindings(s.Volumes, dataPath, s.Name)

	// Create container
	infraContainer, err := cli.ContainerCreate(ctx, &container.Config{
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Image:        s.Image,
		Hostname:     s.Name,
		Env:          envVars,
		ExposedPorts: exposedPorts,
		Labels:       containerLabels,
	}, &container.HostConfig{
		Binds:        binds,
		PortBindings: containerPortBinding,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyAlways,
		},
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			spoutNetwork.ID: {NetworkID: spoutNetwork.ID},
		},
	}, nil, s.Name)

	if err != nil {
		return "", fmt.Errorf("failed to create infrastructure container: %w", err)
	}

	logger.Info("Infrastructure container created successfully",
		zap.String("name", s.Name),
		zap.String("container_id", infraContainer.ID))

	return infraContainer.ID, nil
}

// StartContainerByIdSimple starts a container by ID (simplified version for infrastructure)
func StartContainerByIdSimple(containerId string) error {
	logger.Info("Starting container", zap.String("container_id", containerId[:12]))
	if err := cli.ContainerStart(ctx, containerId, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// GetInfrastructureContainers returns all infrastructure containers
func GetInfrastructureContainers() ([]container.Summary, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("label", "io.spout.infrastructure=true")
	list, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: containerFilter})
	if err != nil {
		return []container.Summary{}, err
	}
	return list, nil
}
