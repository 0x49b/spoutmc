package model

type VelocityConfig struct {
	ConfigVersion                 string      `toml:"config-version"`
	Bind                          string      `toml:"bind"`
	Motd                          string      `toml:"motd"`
	ShowMaxPlayers                int         `toml:"show-max-players"`
	OnlineMode                    bool        `toml:"online-mode"`
	ForceKeyAuthentication        bool        `toml:"force-key-authentication"`
	PreventClientProxyConnections bool        `toml:"prevent-client-proxy-connections"`
	PlayerInfoForwardingMode      string      `toml:"player-info-forwarding-mode"`
	ForwardingSecretFile          string      `toml:"forwarding-secret-file"`
	AnnounceForge                 bool        `toml:"announce-forge"`
	KickExistingPlayers           bool        `toml:"kick-existing-players"`
	PingPassthrough               string      `toml:"ping-passthrough"`
	EnablePlayerAddressLogging    bool        `toml:"enable-player-address-logging"`
	Servers                       Servers     `toml:"servers"`
	ForcedHosts                   ForcedHosts `toml:"forced-hosts"`
	Advanced                      Advanced    `toml:"advanced"`
	Query                         Query       `toml:"query"`
}

type ServerConfig struct {
	Host string
	Port int
}

type Servers struct {
	Spoutlobby    string   `toml:"spoutlobby"`
	Spoutskyblock string   `toml:"spoutskyblock"`
	Spouttest     string   `toml:"spouttest"`
	Try           []string `toml:"try"`
}

type ForcedHosts struct {
	SpoutlobbyExampleCom    []string `toml:"spoutlobby.example.com"`
	SpoutskyblockExampleCom []string `toml:"spoutskyblock.example.com"`
}
type Advanced struct {
	CompressionThreshold                 int  `toml:"compression-threshold"`
	CompressionLevel                     int  `toml:"compression-level"`
	LoginRatelimit                       int  `toml:"login-ratelimit"`
	ConnectionTimeout                    int  `toml:"connection-timeout"`
	ReadTimeout                          int  `toml:"read-timeout"`
	HaproxyProtocol                      bool `toml:"haproxy-protocol"`
	TCPFastOpen                          bool `toml:"tcp-fast-open"`
	BungeePluginMessageChannel           bool `toml:"bungee-plugin-message-channel"`
	ShowPingRequests                     bool `toml:"show-ping-requests"`
	FailoverOnUnexpectedServerDisconnect bool `toml:"failover-on-unexpected-server-disconnect"`
	AnnounceProxyCommands                bool `toml:"announce-proxy-commands"`
	LogCommandExecutions                 bool `toml:"log-command-executions"`
	LogPlayerConnections                 bool `toml:"log-player-connections"`
}
type Query struct {
	Enabled     bool   `toml:"enabled"`
	Port        int    `toml:"port"`
	Map         string `toml:"map"`
	ShowPlugins bool   `toml:"show-plugins"`
}
