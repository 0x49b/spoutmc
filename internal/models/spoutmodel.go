package models

type SpoutConfiguration struct {
	ContainerNetworkID uint                  `json:"container-network-id,omitempty"`
	ContainerNetwork   SpoutContainerNetwork `json:"container-network,omitempty"`
	Servers            []SpoutServer         `json:"servers,omitempty"`
}

type SpoutContainerNetwork struct {
	Name   string `json:"name,omitempty"`
	Driver string `json:"driver,omitempty"`
}

type SpoutServer struct {
	SpoutConfigurationID uint                 `json:"spout-configuration-id,omitempty"`
	Name                 string               `json:"name"`
	Image                string               `json:"image"`
	Proxy                bool                 `json:"proxy,omitempty"`
	Lobby                bool                 `json:"lobby,omitempty"`
	EnvID                uint                 `json:"env-id,omitempty"`
	Env                  StringMap            `gorm:"type:text" json:"env,omitempty"`
	PortsID              uint                 `json:"ports-id,omitempty"`
	Port                 uint                 `json:"port,omitempty"`
	Ports                []SpoutServerPorts   `gorm:"many2many:server_ports" json:"ports,omitempty"`
	Volumes              []SpoutServerVolumes `gorm:"many2many:server_volumes" json:"volumes,omitempty"`
}

type SpoutServerVolumes struct {
	Hostpath      StringSlice `gorm:"type:text" json:"hostpath"`
	Containerpath string      `json:"containerpath"`
}

type SpoutServerPorts struct {
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
}
