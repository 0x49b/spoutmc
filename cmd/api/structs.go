package main

type SpoutServers struct {
	Servers []SpoutServer `json:"server"`
}
type SpoutServerEnv struct {
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
	Hostpath      string `json:"hostpath"`
	Containerpath string `json:"containerpath"`
}

type SpoutServerPorts struct {
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
}

type SpoutServer struct {
	Name    string               `json:"name"`
	Image   string               `json:"image"`
	Env     SpoutServerEnv       `json:"env,omitempty"`
	Ports   SpoutServerPorts     `json:"ports,omitempty"`
	Volumes []SpoutServerVolumes `json:"volumes,omitempty"`
}
