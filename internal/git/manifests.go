package git

import (
	"fmt"
	"spoutmc/internal/infrastructure"
	"spoutmc/internal/models"

	"gopkg.in/yaml.v3"
)

const (
	APIVersionV1Alpha1          = "spoutmc.io/v1alpha1"
	KindSpoutServer             = "SpoutServer"
	KindInfrastructureContainer = "InfrastructureContainer"
)

type ManifestMetadata struct {
	Name        string            `yaml:"name,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
type ServerManifest struct {
	APIVersion string             `yaml:"apiVersion"`
	Kind       string             `yaml:"kind"`
	Metadata   ManifestMetadata   `yaml:"metadata,omitempty"`
	Spec       models.SpoutServer `yaml:"spec"`
}
type InfrastructureManifest struct {
	APIVersion string                                 `yaml:"apiVersion"`
	Kind       string                                 `yaml:"kind"`
	Metadata   ManifestMetadata                       `yaml:"metadata,omitempty"`
	Spec       infrastructure.InfrastructureContainer `yaml:"spec"`
}

type manifestHeader struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

func MarshalServerManifest(server models.SpoutServer) ([]byte, error) {
	manifest := ServerManifest{
		APIVersion: APIVersionV1Alpha1,
		Kind:       KindSpoutServer,
		Metadata: ManifestMetadata{
			Name: server.Name,
		},
		Spec: server,
	}

	return yaml.Marshal(manifest)
}

func ParseServerYAML(data []byte) (models.SpoutServer, error) {
	var header manifestHeader
	if err := yaml.Unmarshal(data, &header); err != nil {
		return models.SpoutServer{}, fmt.Errorf("failed to parse yaml header: %w", err)
	}

	if header.Kind == "" && header.APIVersion == "" {
		var server models.SpoutServer
		if err := yaml.Unmarshal(data, &server); err != nil {
			return models.SpoutServer{}, fmt.Errorf("failed to parse legacy server yaml: %w", err)
		}
		return server, nil
	}

	if header.Kind != KindSpoutServer {
		return models.SpoutServer{}, fmt.Errorf("unsupported kind %q (expected %q)", header.Kind, KindSpoutServer)
	}

	if header.APIVersion == "" {
		return models.SpoutServer{}, fmt.Errorf("apiVersion is required for manifest kind %q", header.Kind)
	}

	var manifest ServerManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return models.SpoutServer{}, fmt.Errorf("failed to parse server manifest: %w", err)
	}

	server := manifest.Spec
	if server.Name == "" && manifest.Metadata.Name != "" {
		server.Name = manifest.Metadata.Name
	}

	if manifest.Metadata.Name != "" && server.Name != manifest.Metadata.Name {
		return models.SpoutServer{}, fmt.Errorf("metadata.name (%s) must match spec.name (%s)", manifest.Metadata.Name, server.Name)
	}

	return server, nil
}

func ParseInfrastructureYAML(data []byte) (infrastructure.InfrastructureContainer, error) {
	var header manifestHeader
	if err := yaml.Unmarshal(data, &header); err != nil {
		return infrastructure.InfrastructureContainer{}, fmt.Errorf("failed to parse yaml header: %w", err)
	}

	if header.Kind == "" && header.APIVersion == "" {
		var container infrastructure.InfrastructureContainer
		if err := yaml.Unmarshal(data, &container); err != nil {
			return infrastructure.InfrastructureContainer{}, fmt.Errorf("failed to parse legacy infrastructure yaml: %w", err)
		}
		return container, nil
	}

	if header.Kind != KindInfrastructureContainer {
		return infrastructure.InfrastructureContainer{}, fmt.Errorf("unsupported kind %q (expected %q)", header.Kind, KindInfrastructureContainer)
	}

	if header.APIVersion == "" {
		return infrastructure.InfrastructureContainer{}, fmt.Errorf("apiVersion is required for manifest kind %q", header.Kind)
	}

	var manifest InfrastructureManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return infrastructure.InfrastructureContainer{}, fmt.Errorf("failed to parse infrastructure manifest: %w", err)
	}

	container := manifest.Spec
	if container.Name == "" && manifest.Metadata.Name != "" {
		container.Name = manifest.Metadata.Name
	}

	if manifest.Metadata.Name != "" && container.Name != manifest.Metadata.Name {
		return infrastructure.InfrastructureContainer{}, fmt.Errorf("metadata.name (%s) must match spec.name (%s)", manifest.Metadata.Name, container.Name)
	}

	return container, nil
}
