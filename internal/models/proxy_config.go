package models

type Config struct {
	ConfigVersion                 string              `toml:"config-version"`
	Bind                          string              `toml:"bind"`
	MOTD                          string              `toml:"motd"`
	ShowMaxPlayers                int                 `toml:"show-max-players"`
	OnlineMode                    bool                `toml:"online-mode"`
	ForceKeyAuthentication        bool                `toml:"force-key-authentication"`
	PreventClientProxyConnections bool                `toml:"prevent-client-proxy-connections"`
	PlayerInfoForwardingMode      string              `toml:"player-info-forwarding-mode"`
	ForwardingSecretFile          string              `toml:"forwarding-secret-file"`
	AnnounceForge                 bool                `toml:"announce-forge"`
	KickExistingPlayers           bool                `toml:"kick-existing-players"`
	PingPassThrough               string              `toml:"ping-passthrough"`
	EnablePlayerAddressLogging    bool                `toml:"enable-player-address-logging"`
	Servers                       map[string]string   `toml:"servers"`
	Try                           []string            `toml:"try"`
	ForcedHosts                   map[string][]string `toml:"forced-hosts"`
	Advanced                      AdvancedConfig      `toml:"advanced"`
	Query                         QueryConfig         `toml:"query"`
}

type AdvancedConfig struct {
	CompressionThreshold                 int  `toml:"compression-threshold"`
	CompressionLevel                     int  `toml:"compression-level"`
	LoginRateLimit                       int  `toml:"login-ratelimit"`
	ConnectionTimeout                    int  `toml:"connection-timeout"`
	ReadTimeout                          int  `toml:"read-timeout"`
	HAProxyProtocol                      bool `toml:"haproxy-protocol"`
	TCPFastOpen                          bool `toml:"tcp-fast-open"`
	BungeePluginMessageChannel           bool `toml:"bungee-plugin-message-channel"`
	ShowPingRequests                     bool `toml:"show-ping-requests"`
	FailoverOnUnexpectedServerDisconnect bool `toml:"failover-on-unexpected-server-disconnect"`
	AnnounceProxyCommands                bool `toml:"announce-proxy-commands"`
	LogCommandExecutions                 bool `toml:"log-command-executions"`
	LogPlayerConnections                 bool `toml:"log-player-connections"`
	AcceptsTransfers                     bool `toml:"accepts-transfers"`
	EnableReusePort                      bool `toml:"enable-reuse-port"`
}

type QueryConfig struct {
	Enabled     bool   `toml:"enabled"`
	Port        int    `toml:"port"`
	Map         string `toml:"map"`
	ShowPlugins bool   `toml:"show-plugins"`
}
