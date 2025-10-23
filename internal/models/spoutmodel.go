package models

type SpoutConfiguration struct {
	Servers []SpoutServer `json:"servers,omitempty" yaml:"servers,omitempty"`
}

type SpoutServer struct {
	SpoutConfigurationID uint                 `json:"spout-configuration-id,omitempty" yaml:"spout-configuration-id,omitempty"`
	Name                 string               `json:"name" yaml:"name"`
	Image                string               `json:"image" yaml:"image"`
	Proxy                bool                 `json:"proxy,omitempty" yaml:"proxy,omitempty"`
	Lobby                bool                 `json:"lobby,omitempty" yaml:"lobby,omitempty"`
	EnvID                uint                 `json:"env-id,omitempty" yaml:"env-id,omitempty"`
	Env                  StringMap            `json:"env,omitempty" yaml:"env,omitempty"`
	PortsID              uint                 `json:"ports-id,omitempty" yaml:"ports-id,omitempty"`
	Port                 uint                 `json:"port,omitempty" yaml:"port,omitempty"`
	Ports                []SpoutServerPorts   `json:"ports,omitempty" yaml:"ports,omitempty"`
	Volumes              []SpoutServerVolumes `json:"volumes,omitempty" yaml:"volumes,omitempty"`
}

type SpoutServerVolumes struct {
	Hostpath      StringSlice `json:"hostpath" yaml:"hostpath"`
	Containerpath string      `json:"containerpath" yaml:"containerpath"`
}

type SpoutServerPorts struct {
	HostPort      string `json:"hostPort" yaml:"hostPort"`
	ContainerPort string `json:"containerPort" yaml:"containerPort"`
}
