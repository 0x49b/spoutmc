package image

type SpoutContainerImage int

const (
	GameServer SpoutContainerImage = iota
	ProxyServer
)

var stateName = map[SpoutContainerImage]string{
	GameServer:  "itzg/minecraft-server",
	ProxyServer: "itzg/mc-proxy",
}

func (ss SpoutContainerImage) String() string {
	return stateName[ss]
}
