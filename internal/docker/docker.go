package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/plugins"
	"spoutmc/internal/storage"
	"spoutmc/internal/utils/path"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleDocker)

func PullImage(ctx context.Context, imageName string) {

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

func GetNetworkContainers(ctx context.Context) ([]container.Summary, error) {
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

func containerExists(ctx context.Context, containerName string) bool {
	_, err := GetContainer(ctx, containerName)
	return err == nil
}

// ContainerExists reports whether a container with this name exists (Spout server name).
func ContainerExists(ctx context.Context, containerName string) bool {
	return containerExists(ctx, containerName)
}

func GetContainer(ctx context.Context, containerName string) (container.Summary, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("name", containerName)

	containerList, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: containerFilter})

	if err != nil {
		return container.Summary{}, err
	}

	if len(containerList) < 1 {
		return container.Summary{}, fmt.Errorf("Cannot find container for name %s", containerName)
	}

	return containerList[0], nil
}

func GetContainerById(ctx context.Context, containerId string) (container.InspectResponse, error) {
	requestedContainer, err := cli.ContainerInspect(ctx, containerId)
	if err != nil {
		return container.InspectResponse{}, err
	}
	return requestedContainer, nil
}

func GetContainerStats(ctx context.Context, containerId string) (container.StatsResponse, error) {
	stats, err := cli.ContainerStats(ctx, containerId, false)
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

func getHostNetworkId(ctx context.Context) (network.Inspect, error) {
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

func StartContainer(ctx context.Context, s models.SpoutServer, dataPath string) error {
	normalizedDataPath := pathutil.NormalizeHostPath(dataPath)

	// Check if image exists, if not pull it
	_, err := cli.ImageInspect(ctx, s.Image)
	if err != nil {
		// Pull Image for Server
		PullImage(ctx, s.Image)
	}

	// Pre-create volume directories with proper user ownership
	// This prevents Docker from creating them as root on Linux
	if err := ensureVolumeDirectoriesExist(s.Volumes, normalizedDataPath, s.Name); err != nil {
		logger.Error("Failed to create volume directories", zap.Error(err))
		return fmt.Errorf("failed to create volume directories: %w", err)
	}

	if !containerExists(ctx, s.Name) {
		logger.Info(fmt.Sprintf("Creating container %s", s.Name))
		exposedPorts, containerPortBinding := nat.PortSet{}, nat.PortMap{}
		if s.Proxy {
			exposedPorts, containerPortBinding = MapExposedPorts(s.Ports)
			// Expose the Velocity players-bridge API on localhost for the SpoutMC host process.
			bridgePort := nat.Port(DefaultPlayersBridgePort + "/tcp")
			exposedPorts[bridgePort] = struct{}{}
			if _, exists := containerPortBinding[bridgePort]; !exists {
				containerPortBinding[bridgePort] = []nat.PortBinding{{
					HostIP:   "127.0.0.1",
					HostPort: DefaultPlayersBridgePort,
				}}
			}
		}

		spoutNetwork := GetSpoutNetwork(ctx)
		hostNetwork, err := getHostNetworkId(ctx)
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

		envWithPlugins := plugins.MergePluginsEnv(storage.GetDB(), s)
		envVars := MapEnvironmentVariables(envWithPlugins)

		// Prepare volume bindings
		mounts := MapVolumeBindings(s.Volumes, normalizedDataPath, s.Name)

		// For proxy servers, mount the forwarding.secret file
		// The itzg/mc-proxy image copies files from /config to /server on startup
		if s.Proxy {
			// Ensure the forwarding.secret file exists BEFORE mounting it
			// If we try to mount a non-existent file, Docker creates it as a directory
			secretSourcePath := filepath.Join(normalizedDataPath, s.Name, "server", "forwarding.secret")
			if err := ensureForwardingSecret(secretSourcePath); err != nil {
				logger.Warn("Failed to create forwarding.secret file before mounting",
					zap.Error(err),
					zap.String("path", secretSourcePath))
				// Continue anyway - the file might be created by velocity.toml sync
			}

			secretTargetPath := "/config/forwarding.secret"
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   secretSourcePath,
				Target:   secretTargetPath,
				ReadOnly: true,
			})
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
			Mounts:        mounts,
			PortBindings:  containerPortBinding,
			RestartPolicy: mapServerRestartPolicy(s),
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
		startContainer, err := GetContainer(ctx, s.Name)
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

func mapServerRestartPolicy(server models.SpoutServer) container.RestartPolicy {
	if server.RestartPolicy == nil || server.RestartPolicy.Container == nil {
		return container.RestartPolicy{}
	}

	policy := server.RestartPolicy.Container
	if policy.Policy == "" {
		return container.RestartPolicy{}
	}

	restartPolicy := container.RestartPolicy{
		Name: container.RestartPolicyMode(policy.Policy),
	}

	if policy.Policy == models.DockerRestartPolicyOnFailure && policy.MaxRetries != nil {
		restartPolicy.MaximumRetryCount = int(*policy.MaxRetries)
	}

	return restartPolicy
}

func ShutdownContainers(ctx context.Context) error {
	containers, err := GetNetworkContainers(ctx)
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

func StopContainerById(ctx context.Context, containerId string) {
	if err := cli.ContainerStop(ctx, containerId, container.StopOptions{Signal: "SIGKILL"}); err != nil {
		logger.Error("Cannot stop container", zap.Error(err))
	}

}

func StartContainerById(ctx context.Context, containerId string) {
	if err := cli.ContainerStart(ctx, containerId, container.StartOptions{}); err != nil {
		logger.Error("Cannot start container", zap.Error(err))
	}
}

func RestartContainerById(ctx context.Context, containerId string) {
	if err := cli.ContainerRestart(ctx, containerId, container.StopOptions{}); err != nil {
		logger.Error("Cannot start container", zap.Error(err))
	}
}

// RestartProxyContainer restarts the proxy container if present.
func RestartProxyContainer(ctx context.Context) error {
	proxyContainer, err := GetProxyContainer(ctx)
	if err != nil {
		return fmt.Errorf("failed to get proxy container: %w", err)
	}

	if err := cli.ContainerRestart(ctx, proxyContainer.ID, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to restart proxy container %s: %w", proxyContainer.ID, err)
	}

	logger.Info("Proxy container restarted",
		zap.String("name", strings.TrimPrefix(proxyContainer.Names[0], "/")),
		zap.String("id", proxyContainer.ID[:12]))
	return nil
}

func GetProxyContainer(ctx context.Context) (container.Summary, error) {
	proxyContainer, err := filterForContainerLabel(ctx, "io.spout.proxy")
	if err != nil {
		return container.Summary{}, err
	}
	return proxyContainer, nil
}

func GetLobbyContainer(ctx context.Context) (container.Summary, error) {
	lobbyContainer, err := filterForContainerLabel(ctx, "io.spout.lobby")
	if err != nil {
		return container.Summary{}, err
	}
	return lobbyContainer, nil
}

func GetProxyVolumeMount(ctx context.Context) string {
	proxyContainer, err := GetProxyContainer(ctx)
	if err != nil {
		return ""
	}
	proxyContainerInspect, err := cli.ContainerInspect(ctx, proxyContainer.ID)
	if err != nil {
		return ""
	}
	return proxyContainerInspect.Mounts[0].Source
}

func GetProxyConfigFilePath(ctx context.Context) string {
	return filepath.Join(GetProxyVolumeMount(ctx), "velocity.toml")
}

func filterForContainerLabel(ctx context.Context, label string) (container.Summary, error) {
	networkContainer, err := GetNetworkContainers(ctx)
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
	return container.Summary{}, fmt.Errorf("no Server found for label %s", label)
}

// FetchDockerLogs streams Docker container logs. When TTY is disabled (typical for Paper/Velocity),
// Docker multiplexes stdout/stderr with framing bytes; those must be decoded with stdcopy before
// line scanning — otherwise the console shows garbage or nothing useful.
func FetchDockerLogs(ctx context.Context, id string) (<-chan string, error) {
	info, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("unable to inspect container %s: %w", id, err)
	}
	isTTY := info.Config != nil && info.Config.Tty

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "1000",
	}

	readCloser, err := cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve logs for container %s: %w", id, err)
	}

	logChan := make(chan string)

	go func() {
		defer readCloser.Close()
		defer close(logChan)

		pump := func(r io.Reader) {
			if r == nil {
				return
			}
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				line := scanner.Text()
				trimmed := strings.TrimSpace(line)
				if trimmed == "" || trimmed == ">" {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case logChan <- line:
				}
			}
			if err := scanner.Err(); err != nil {
				logger.Warn("error reading docker log stream", zap.Error(err))
			}
		}

		if isTTY {
			pump(readCloser)
			return
		}

		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		go func() {
			defer outW.Close()
			defer errW.Close()
			_, _ = stdcopy.StdCopy(outW, errW, readCloser)
		}()

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			pump(outR)
		}()
		go func() {
			defer wg.Done()
			pump(errR)
		}()
		wg.Wait()
	}()

	return logChan, nil
}

func RemoveLocalVolumeDataForContainer(ctx context.Context, containerId string) {
	containerVolume, err := GetContainerById(ctx, containerId)
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

func RemoveContainerById(ctx context.Context, containerId string, removeVolume bool) error {
	if removeVolume {
		RemoveLocalVolumeDataForContainer(ctx, containerId)
	}
	return cli.ContainerRemove(ctx, containerId, container.RemoveOptions{
		RemoveVolumes: removeVolume,
	})
}

func RecreateContainer(ctx context.Context, containerConfig models.SpoutServer, dataPath string) error {
	logger.Info("Recreating container", zap.String("containerName", containerConfig.Name))
	recreateContainer, err := GetContainer(ctx, containerConfig.Name)
	if err != nil {
		return err
	}
	StopContainerById(ctx, recreateContainer.ID)
	err = RemoveContainerById(ctx, recreateContainer.ID, false)
	if err != nil {
		return err
	}
	StartContainer(ctx, containerConfig, dataPath)
	return nil
}

func StopAndRemoveContainerById(ctx context.Context, containerId string) error {
	StopContainerById(ctx, containerId)
	return RemoveContainerById(ctx, containerId, true)
}

// CreateOrRecreateInfrastructureContainer creates or recreates an infrastructure container.
// Recreating ensures config changes (e.g. ports) are applied when the container already exists.
func CreateOrRecreateInfrastructureContainer(ctx context.Context, s models.SpoutServer, dataPath string) (string, error) {
	if containerExists(ctx, s.Name) {
		logger.Info("Recreating infrastructure container to apply config", zap.String("name", s.Name))
		existingContainer, err := GetContainer(ctx, s.Name)
		if err != nil {
			return "", err
		}
		StopContainerById(ctx, existingContainer.ID)
		if err := RemoveContainerById(ctx, existingContainer.ID, false); err != nil {
			return "", fmt.Errorf("failed to remove existing container: %w", err)
		}
	}
	return CreateInfrastructureContainer(ctx, s, dataPath)
}

// CreateInfrastructureContainer creates a container with infrastructure labels
func CreateInfrastructureContainer(ctx context.Context, s models.SpoutServer, dataPath string) (string, error) {
	logger.Info("Creating infrastructure container", zap.String("name", s.Name))
	normalizedDataPath := pathutil.NormalizeHostPath(dataPath)

	// Check if image exists, if not pull it
	_, err := cli.ImageInspect(ctx, s.Image)
	if err != nil {
		logger.Info("Pulling infrastructure image", zap.String("image", s.Image))
		PullImage(ctx, s.Image)
	}

	// Pre-create volume directories with proper user ownership
	// This prevents Docker from creating them as root on Linux
	if err := ensureVolumeDirectoriesExist(s.Volumes, normalizedDataPath, s.Name); err != nil {
		return "", fmt.Errorf("failed to create volume directories: %w", err)
	}

	// Map exposed ports and port bindings
	exposedPorts, containerPortBinding := MapExposedPorts(s.Ports)

	// Get spout network
	spoutNetwork := GetSpoutNetwork(ctx)

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
	mounts := MapVolumeBindings(s.Volumes, normalizedDataPath, s.Name)

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
		Mounts:       mounts,
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
func StartContainerByIdSimple(ctx context.Context, containerId string) error {
	logger.Info("Starting container", zap.String("container_id", containerId[:12]))
	if err := cli.ContainerStart(ctx, containerId, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// GetInfrastructureContainers returns all infrastructure containers
func GetInfrastructureContainers(ctx context.Context) ([]container.Summary, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("label", "io.spout.infrastructure=true")
	list, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: containerFilter})
	if err != nil {
		return []container.Summary{}, err
	}
	return list, nil
}
