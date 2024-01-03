package models

import "gorm.io/gorm"

type SpoutConfiguration struct {
	gorm.Model
	ContainerNetworkID uint                  `json:"container-network-id,omitempty" gorm:"index"`
	ContainerNetwork   SpoutContainerNetwork `json:"container-network,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ContainerNetworkID"`
	Servers            []SpoutServer         `json:"servers,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:SpoutConfigurationID"`
}

type SpoutContainerNetwork struct {
	gorm.Model
	Name   string `json:"name,omitempty"`
	Driver string `json:"driver,omitempty"`
}

type SpoutServer struct {
	gorm.Model
	SpoutConfigurationID uint                 `json:"spout-configuration-id,omitempty" gorm:"index"`
	Name                 string               `json:"name"`
	Image                string               `json:"image"`
	Proxy                bool                 `json:"proxy,omitempty"`
	Lobby                bool                 `json:"lobby,omitempty"`
	EnvID                uint                 `json:"env-id,omitempty" gorm:"index"`
	Env                  SpoutServerEnv       `json:"env,omitempty" gorm:"foreignKey:EnvID"`
	PortsID              uint                 `json:"ports-id,omitempty" gorm:"index"`
	Ports                SpoutServerPorts     `json:"ports,omitempty" gorm:"foreignKey:PortsID"`
	Volumes              []SpoutServerVolumes `json:"volumes,omitempty"`
}

type SpoutServerEnv struct {
	gorm.Model
	Eula                 string `json:"EULA"`
	Type                 string `json:"TYPE"`
	OnlineMode           string `json:"ONLINE_MODE"`
	EnforceSecureProfile string `json:"ENFORCE_SECURE_PROFILE"`
	MaxMemory            string `json:"MAX_MEMORY"`
	Version              string `json:"VERSION"`
	Gui                  string `json:"GUI"`
	Console              string `json:"CONSOLE"`
	LogTimestamp         string `json:"LOG_TIMESTAMP"`
	Tz                   string `json:"TZ"`
}

type SpoutServerVolumes struct {
	gorm.Model
	Hostpath      []string `json:"hostpath"`
	Containerpath string   `json:"containerpath"`
}

type SpoutServerPorts struct {
	gorm.Model
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
}
