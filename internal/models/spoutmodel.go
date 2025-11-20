package models

import "time"

type SpoutConfiguration struct {
	Git      *GitConfig     `json:"git,omitempty" yaml:"git,omitempty"`
	Storage  *StorageConfig `json:"storage,omitempty" yaml:"storage,omitempty"`
	Files    *FilesConfig   `json:"files,omitempty" yaml:"files,omitempty"`
	EULA     *EULAConfig    `json:"eula,omitempty" yaml:"eula,omitempty"`
	Versions []string       `json:"versions,omitempty" yaml:"versions,omitempty"`
	Servers  []SpoutServer  `json:"servers,omitempty" yaml:"servers,omitempty"`
}

type GitConfig struct {
	Enabled       bool          `json:"enabled" yaml:"enabled"`
	Repository    string        `json:"repository" yaml:"repository"`
	Branch        string        `json:"branch" yaml:"branch"`
	Token         string        `json:"token,omitempty" yaml:"token,omitempty"`
	PollInterval  time.Duration `json:"poll_interval" yaml:"poll_interval"`
	WebhookSecret string        `json:"webhook_secret,omitempty" yaml:"webhook_secret,omitempty"`
	LocalPath     string        `json:"local_path" yaml:"local_path"`
}

type StorageConfig struct {
	DataPath string `json:"data_path" yaml:"data_path"`
}

type FilesConfig struct {
	ExcludePatterns []string `json:"exclude_patterns" yaml:"exclude_patterns"`
}

type EULAConfig struct {
	Accepted   bool      `json:"accepted" yaml:"accepted"`
	AcceptedOn time.Time `json:"accepted_on" yaml:"accepted_on"`
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
	Containerpath string `json:"containerpath" yaml:"containerpath"`
}

type SpoutServerPorts struct {
	HostPort      string `json:"hostPort" yaml:"hostPort"`
	ContainerPort string `json:"containerPort" yaml:"containerPort"`
}
