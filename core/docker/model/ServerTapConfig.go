package model

type ServerTapConfig struct {
	Port              int    `yaml:"port"`
	Debug             bool   `yaml:"debug"`
	UseKeyAuth        bool   `yaml:"useKeyAuth"`
	Key               string `yaml:"key"`
	NormalizeMessages bool   `yaml:"normalizeMessages"`
	TLS               struct {
		Enabled          bool   `yaml:"enabled"`
		Keystore         string `yaml:"keystore"`
		KeystorePassword string `yaml:"keystorePassword"`
		Sni              bool   `yaml:"sni"`
	} `yaml:"tls"`
	CorsOrigins            []string    `yaml:"corsOrigins"`
	WebsocketConsoleBuffer int         `yaml:"websocketConsoleBuffer"`
	DisableSwagger         bool        `yaml:"disable-swagger"`
	BlockedPaths           interface{} `yaml:"blocked-paths"`
}
