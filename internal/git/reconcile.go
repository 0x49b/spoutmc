package git

import (
	"context"
	"fmt"
	"spoutmc/internal/docker"
	"spoutmc/internal/models"
	"strings"

	dockertypes "github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

// ReconcileRuntimeState enforces desired server state against current runtime state.
func ReconcileRuntimeState(ctx context.Context, desiredConfig models.SpoutConfiguration) (SyncSummary, error) {
	summary := SyncSummary{}

	desiredByName := make(map[string]models.SpoutServer, len(desiredConfig.Servers))
	for _, s := range desiredConfig.Servers {
		desiredByName[s.Name] = s
	}

	actualContainers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		return summary, fmt.Errorf("failed to list runtime containers: %w", err)
	}

	actualByName := make(map[string]dockertypes.Summary, len(actualContainers))
	for _, c := range actualContainers {
		if len(c.Names) == 0 {
			continue
		}
		name := strings.TrimPrefix(c.Names[0], "/")
		actualByName[name] = c
	}

	dataPath := ""
	if desiredConfig.Storage != nil {
		dataPath = desiredConfig.Storage.DataPath
	}

	// Ensure all desired servers exist and match expected configuration.
	for name, desired := range desiredByName {
		if _, exists := actualByName[name]; !exists {
			if err := docker.StartContainer(ctx, desired, dataPath); err != nil {
				logger.Error("failed to create missing desired container", zap.String("server", name), zap.Error(err))
				continue
			}
			summary.Created++
		}
	}

	// Remove runtime containers that are no longer desired.
	for actualName, actual := range actualByName {
		if _, shouldExist := desiredByName[actualName]; shouldExist {
			continue
		}
		if err := docker.StopAndRemoveContainerById(ctx, actual.ID); err != nil {
			logger.Error("failed to prune undesired container", zap.String("server", actualName), zap.Error(err))
			continue
		}
		summary.Pruned++
	}

	summary.DriftCorrections = summary.Created + summary.Recreated + summary.Pruned
	return summary, nil
}
