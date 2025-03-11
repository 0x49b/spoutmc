package docker

import (
	"github.com/docker/docker/api/types/network"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

func CreateSpoutNetwork(networkName string) network.Inspect {

	networkList, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return network.Inspect{}
	}

	var availableNetworks []string

	for _, n := range networkList {
		availableNetworks = append(availableNetworks, n.Name)
	}

	if !slices.Contains(availableNetworks, networkName) {
		spoutNetwork, err := cli.NetworkCreate(ctx, networkName, network.CreateOptions{Driver: "bridge"})
		if err != nil {
			logger.Error("Cannot create network", zap.Error(err))
		}
		return network.Inspect{ID: spoutNetwork.ID, Name: networkName}
	} else {
		for _, n := range networkList {
			if networkName == n.Name {
				return n
			}
		}
	}

	return network.Inspect{}
}

func GetSpoutNetwork() network.Inspect {
	networkList, _ := cli.NetworkList(ctx, network.ListOptions{})
	networkName := "spoutnetwork" // todo get this from config
	for _, n := range networkList {
		if networkName == n.Name {
			return n
		}
	}

	return network.Inspect{}
}

// todo don't know if this is needed
func destroySpoutNetwork() {
	err := cli.NetworkRemove(ctx, GetSpoutNetwork().ID)
	if err != nil {
		logger.Error("", zap.Error(err))
	}
}
