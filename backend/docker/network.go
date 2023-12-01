package docker

import (
	"github.com/docker/docker/api/types"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

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

	return types.NetworkResource{}
}

// todo don't know if this is needed
func destroySpoutNetwork() {
	err := cli.NetworkRemove(ctx, GetSpoutNetwork().ID)
	if err != nil {
		logger.Error("", zap.Error(err))
	}
}
