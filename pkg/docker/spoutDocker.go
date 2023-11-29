package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"io"
	"slices"
	"spoutmc/pkg/log"
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

func isImagePulled(imageName string) bool {

	containerList, err := cli.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		return false
	}

	for _, i := range containerList {
		for _, l := range i.RepoTags {
			if l == imageName {
				return true
			}
		}
	}

	return false
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
