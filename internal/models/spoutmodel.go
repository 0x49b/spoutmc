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
	Env                  SpoutServerEnv       `json:"env,omitempty"`
	PortsID              uint                 `json:"ports-id,omitempty"`
	Ports                []SpoutServerPorts   `json:"ports,omitempty"`
	Volumes              []SpoutServerVolumes `json:"volumes,omitempty"`
}

type SpoutServerEnv struct {
	Eula                 string   `json:"EULA"`
	Type                 string   `json:"TYPE"`
	OnlineMode           string   `json:"ONLINE_MODE"`
	EnforceSecureProfile string   `json:"ENFORCE_SECURE_PROFILE"`
	MaxMemory            string   `json:"MAX_MEMORY"`
	Version              string   `json:"VERSION"`
	Gui                  string   `json:"GUI"`
	Console              string   `json:"CONSOLE"`
	LogTimestamp         string   `json:"LOG_TIMESTAMP"`
	Tz                   string   `json:"TZ"`
	Plugins              []string `json:"PLUGINS,omitempty"`
	SpigetIds            string   `json:"SPIGET_RESOURCES,omitempty"`
}

type SpoutServerVolumes struct {
	Hostpath      []string `json:"hostpath"`
	Containerpath string   `json:"containerpath"`
}

type SpoutServerPorts struct {
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
}
