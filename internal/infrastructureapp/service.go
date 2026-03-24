package infrastructureapp

import (
	"context"
	"errors"
	"spoutmc/internal/docker"
	"sync"

	"github.com/docker/docker/api/types/container"
)

var ErrInfrastructureNotFound = errors.New("infrastructure container not found")

type Service struct{}

func NewService() *Service {
	return &Service{}
}

type InfrastructureContainer struct {
	Summary container.Summary `json:"summary"`
	Type    string            `json:"type"`
}

type ContainerWithStats struct {
	Summary container.Summary `json:"summary"`
	Type    string            `json:"type"`
	Stats   interface{}       `json:"stats,omitempty"`
}

func (s *Service) ListContainers(ctx context.Context) ([]InfrastructureContainer, error) {
	containers, err := docker.GetInfrastructureContainers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]InfrastructureContainer, 0, len(containers))
	for _, cont := range containers {
		out = append(out, InfrastructureContainer{
			Summary: cont,
			Type:    DetermineType(cont.Labels),
		})
	}
	return out, nil
}

func (s *Service) GetContainer(ctx context.Context, id string) (InfrastructureContainer, interface{}, error) {
	inspectData, err := docker.GetContainerById(ctx, id)
	if err != nil {
		return InfrastructureContainer{}, nil, ErrInfrastructureNotFound
	}

	containers, err := docker.GetInfrastructureContainers(ctx)
	if err != nil {
		return InfrastructureContainer{}, nil, err
	}

	for _, cont := range containers {
		if cont.ID == id {
			return InfrastructureContainer{
				Summary: cont,
				Type:    DetermineType(cont.Labels),
			}, inspectData, nil
		}
	}

	return InfrastructureContainer{}, nil, ErrInfrastructureNotFound
}

func (s *Service) GetContainerStats(ctx context.Context, id string) (interface{}, error) {
	return docker.GetContainerStats(ctx, id)
}

func (s *Service) FetchContainerLogs(ctx context.Context, id string) (<-chan string, error) {
	return docker.FetchDockerLogs(ctx, id)
}

func (s *Service) StreamSnapshot(ctx context.Context) ([]ContainerWithStats, error) {
	containers, err := docker.GetInfrastructureContainers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ContainerWithStats, len(containers))
	var wg sync.WaitGroup
	for i, cont := range containers {
		wg.Add(1)
		go func(index int, item container.Summary) {
			defer wg.Done()
			enriched := ContainerWithStats{
				Summary: item,
				Type:    DetermineType(item.Labels),
			}
			stats, err := docker.GetContainerStats(ctx, item.ID)
			if err == nil {
				enriched.Stats = stats
			}
			out[index] = enriched
		}(i, cont)
	}
	wg.Wait()
	return out, nil
}

func DetermineType(labels map[string]string) string {
	if value, exists := labels["io.spout.database"]; exists && value == "true" {
		return "database"
	}
	return "unknown"
}
