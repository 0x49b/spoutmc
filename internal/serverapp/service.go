package serverapp

import (
	"context"
	"errors"
	"spoutmc/internal/docker"
	serverpkg "spoutmc/internal/server"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
)

var ErrServerNotFound = errors.New("server container not found")

type Service struct{}

func NewService() *Service {
	return &Service{}
}

type EnrichedContainer struct {
	container.Summary
	StartedAt string `json:"StartedAt,omitempty"`
	Type      string `json:"Type,omitempty"`
}

type ContainerWithStats struct {
	Container EnrichedContainer `json:"container"`
	Stats     interface{}       `json:"stats,omitempty"`
}

func (s *Service) GetServer(ctx context.Context, id string) (EnrichedContainer, error) {
	inspectData, err := docker.GetContainerById(ctx, id)
	if err != nil {
		return EnrichedContainer{}, err
	}

	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		return EnrichedContainer{}, err
	}

	for _, cont := range containers {
		if cont.ID != id {
			continue
		}
		enriched := EnrichedContainer{
			Summary: cont,
			Type:    serverpkg.DetermineServerType(cont.Labels),
		}
		if inspectData.State != nil {
			enriched.StartedAt = inspectData.State.StartedAt
		}
		return enriched, nil
	}

	return EnrichedContainer{}, ErrServerNotFound
}

func (s *Service) ListServers(ctx context.Context) ([]EnrichedContainer, error) {
	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		return nil, err
	}

	enrichedContainers := make([]EnrichedContainer, 0, len(containers))
	for _, cont := range containers {
		enriched := EnrichedContainer{
			Summary: cont,
			Type:    serverpkg.DetermineServerType(cont.Labels),
		}

		inspectData, err := docker.GetContainerById(ctx, cont.ID)
		if err == nil && inspectData.State != nil {
			enriched.StartedAt = inspectData.State.StartedAt
		}

		enrichedContainers = append(enrichedContainers, enriched)
	}

	return enrichedContainers, nil
}

func (s *Service) GetContainerStats(ctx context.Context, id string) (interface{}, error) {
	return docker.GetContainerStats(ctx, id)
}

func (s *Service) FetchContainerLogs(ctx context.Context, id string) (<-chan string, error) {
	return docker.FetchDockerLogs(ctx, id)
}

func (s *Service) StreamSnapshot(ctx context.Context) ([]ContainerWithStats, error) {
	containers, err := docker.GetNetworkContainers(ctx)
	if err != nil {
		return nil, err
	}

	enrichedContainers := make([]ContainerWithStats, len(containers))
	var wg sync.WaitGroup

	for i, cont := range containers {
		wg.Add(1)
		go func(index int, containerSummary container.Summary) {
			defer wg.Done()

			enrichedContainer := EnrichedContainer{
				Summary: containerSummary,
				Type:    serverpkg.DetermineServerType(containerSummary.Labels),
			}

			inspectData, err := docker.GetContainerById(ctx, containerSummary.ID)
			if err == nil && inspectData.State != nil {
				enrichedContainer.StartedAt = inspectData.State.StartedAt
			}

			containerData := ContainerWithStats{Container: enrichedContainer}
			stats, err := docker.GetContainerStats(ctx, containerSummary.ID)
			if err == nil {
				containerData.Stats = stats
			}

			enrichedContainers[index] = containerData
		}(i, cont)
	}

	wg.Wait()
	return enrichedContainers, nil
}

func IsContextCanceled(err error, ctx context.Context) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	return ctx.Err() != nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(errText, "context canceled") ||
		strings.Contains(errText, "request canceled")
}
