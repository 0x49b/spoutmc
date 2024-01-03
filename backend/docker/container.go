package docker

import (
	"os"
	"strings"
)

// todo this needs lot of refactoring man

import (
	"bufio"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"go.uber.org/zap"
	"path/filepath"
	"spoutmc/backend/models"
)

func addToProxyConfig(newServerName string) {

	getwd, err := os.Getwd()
	if err != nil {
		return
	}

	velocityFilepath := filepath.Join(getwd, "testservers", "data", "spoutproxy", "velocity.toml")

	// Open the file
	file, err := os.Open(velocityFilepath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	var lines []string

	// Iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	serverStartLine := 0
	serverEndLine := 0
	for i, l := range lines {

		if l == "[servers]" {
			serverStartLine = i
		}

		if l == "[forced-hosts]" {
			serverEndLine = i - 1
		}

	}

	logger.Info(fmt.Sprintf("ServerStartLine %d -> ServerEndLine %d", serverStartLine, serverEndLine))

	containers, err := GetNetworkContainers()
	if err != nil {
		return
	}

	var newServerLines []string
	lobbyContainer := ""

	for _, c := range containers {

		containerDetails, err := cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			return
		}

		_, lobby := containerDetails.Config.Labels["io.spout.lobby"]
		_, proxy := containerDetails.Config.Labels["io.spout.proxy"]

		if lobby {
			lobbyContainer = containerDetails.Config.Hostname
		}

		logger.Info(containerDetails.Config.Hostname)

		if !proxy {
			newServerLines = append(newServerLines, fmt.Sprintf("%s=\"%s:25565\"", containerDetails.Config.Hostname, containerDetails.Config.Hostname))
		}
	}

	for i := serverStartLine + 1; i < serverEndLine+1; i++ {
		lines[i] = ""
	}

	newServerLines = append(newServerLines, fmt.Sprintf("%s=\"%s:25565\"", newServerName, newServerName))
	newServerLines = append(newServerLines, "")
	newServerLines = append(newServerLines, fmt.Sprintf("try = [\"%s\"]", lobbyContainer))

	startIndex := serverStartLine + 1
	for _, n := range newServerLines {
		lines = insertAndShift(lines, startIndex, n)
		startIndex = startIndex + 1
	}

	lines = compactConfig(lines)

	for _, k := range lines {
		fmt.Println(k)
	}

	err = writeToVelocityConfig(lines, velocityFilepath)
	if err != nil {
		logger.Error("", zap.Error(err))
	}

}

func removeFromConfig(serverName string) {
	var result []string

	needle := fmt.Sprintf("%s=\"%s:25565\"", serverName, serverName)

	getwd, err := os.Getwd()
	if err != nil {
		logger.Error("", zap.Error(err))
	}

	velocityFilepath := filepath.Join(getwd, "testservers", "data", "spoutproxy", "velocity.toml")

	// Open the file
	file, err := os.Open(velocityFilepath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		logger.Error("", zap.Error(err))
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	var lines []string

	// Iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	for _, str := range lines {
		if str != needle {
			result = append(result, str)
		}
	}

	err = writeToVelocityConfig(result, velocityFilepath)
	if err != nil {
		logger.Error("", zap.Error(err))
	}
}

func compactConfig(slice []string) []string {
	var result []string

	for _, str := range slice {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}

func writeToVelocityConfig(lines []string, filename string) error {
	// Open the file for writing, truncating it if it exists, and creating it if it doesn't
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a buffered writer to improve write performance
	writer := bufio.NewWriter(file)

	// Write each line to the file
	for _, line := range lines {
		_, err := fmt.Fprintln(writer, line)
		if err != nil {
			return err
		}
	}

	// Flush the buffered writer to ensure all data is written to the file
	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func insertAndShift(slice []string, index int, value string) []string {
	// Ensure that the index is within bounds
	if index < 0 || index > len(slice) {
		fmt.Println("Index out of bounds")
		return slice
	}

	// Expand the slice by one element
	slice = append(slice, "")

	// Shift the elements to make room for the new element
	copy(slice[index+1:], slice[index:])
	slice[index] = value

	return slice
}

func RestartProxy() {
	proxyContainer, err := GetProxyContainer()
	if err != nil {
		logger.Error("", zap.Error(err))
	}

	err = cli.ContainerRestart(ctx, proxyContainer.ID, container.StopOptions{})
	if err != nil {
		logger.Error("", zap.Error(err))
	}
	logger.Info("Proxy restart initiated")
}

func removeDataDirectory(directoryPath string) {
	err := os.RemoveAll(directoryPath)
	if err != nil {
		return
	}
}

func DeleteContainer(containerId string) (types.ContainerJSON, error) {

	removeContainer, err := GetContainerById(containerId)
	removeFromConfig(removeContainer.Config.Hostname)

	if err != nil {
		logger.Error("", zap.Error(err))
		return types.ContainerJSON{}, err
	}

	err = cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		logger.Error("", zap.Error(err))
		return types.ContainerJSON{}, err
	}
	logger.Info(fmt.Sprintf("removed server %s", removeContainer.Config.Hostname))

	RestartProxy()

	for _, v := range removeContainer.Mounts {
		removeDataDirectory(v.Source)
	}

	return removeContainer, nil
}

func CreateContainer(serverName string) (container.CreateResponse, error) {

	serverName = strings.ToLower(serverName)

	addToProxyConfig(serverName)

	logger.Info(fmt.Sprintf("Creating container %s", serverName))
	spoutNetwork := GetSpoutNetwork()
	hostNetwork, err := getHostNetworkId()
	if err != nil {
		logger.Error("", zap.Error(err))
		return container.CreateResponse{}, err
	}

	endpoints := make(map[string]*network.EndpointSettings, 2)
	endpoints[spoutNetwork.ID] = &network.EndpointSettings{EndpointID: spoutNetwork.ID}
	endpoints[hostNetwork.ID] = &network.EndpointSettings{EndpointID: hostNetwork.ID}

	spoutContainer, err := cli.ContainerCreate(ctx, &container.Config{
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Image:        "itzg/minecraft-server",
		Hostname:     serverName,
		Env:          MapEnvironmentVariables(models.SpoutServerEnv{Eula: "TRUE", Type: "PAPER", OnlineMode: "FALSE", EnforceSecureProfile: "FALSE", MaxMemory: "4G", Version: "1.20.4", Gui: "FALSE", Console: "FALSE", LogTimestamp: "TRUE", Tz: "Europ/Zurich"}),
		Labels:       map[string]string{"io.spout.servername": serverName, "io.spout.network": "true"},
	}, &container.HostConfig{
		Binds: MapVolumeBindings([]models.SpoutServerVolumes{{Hostpath: []string{"testservers", "data", serverName}, Containerpath: "/data"}}),
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{spoutNetwork.ID: {NetworkID: spoutNetwork.ID}}},
		nil, serverName)
	if err != nil {
		logger.Error("Error creating container", zap.Error(err))
		return container.CreateResponse{}, err
	}

	statusCh, errCh := cli.ContainerWait(ctx, spoutContainer.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return container.CreateResponse{}, err
		}
	case <-statusCh:
		logger.Info(fmt.Sprintf("container %s created", serverName))
	}

	if err := cli.ContainerStart(ctx, spoutContainer.ID, types.ContainerStartOptions{}); err != nil {
		logger.Error("Cannot start container", zap.Error(err))
	}

	RestartProxy()
	return spoutContainer, nil
}
