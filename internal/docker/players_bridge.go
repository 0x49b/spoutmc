package docker

import (
	"context"
	"strings"
)

// DefaultPlayersBridgePort is the velocity-players-bridge HTTP port inside the proxy container.
const DefaultPlayersBridgePort = "29132"

// ResolvePlayersBridgeBaseURL returns http://<proxy-container-ip>:29132 on the Spout Docker network.
// Use this when SPOUT_PLAYERS_BRIDGE_URL is unset: the bridge listens inside the Velocity container,
// not on the host loopback.
func ResolvePlayersBridgeBaseURL(ctx context.Context) string {
	if cli == nil {
		return ""
	}
	proxy, err := GetProxyContainer(ctx)
	if err != nil {
		return ""
	}
	ins, err := cli.ContainerInspect(ctx, proxy.ID)
	if err != nil {
		return ""
	}
	if ins.NetworkSettings == nil || ins.NetworkSettings.Networks == nil {
		return ""
	}
	spoutNet := GetSpoutNetwork(ctx)
	if spoutNet.ID != "" {
		if ep, ok := ins.NetworkSettings.Networks[spoutNet.ID]; ok && ep != nil && ep.IPAddress != "" {
			return "http://" + ep.IPAddress + ":" + DefaultPlayersBridgePort
		}
	}
	netName := GetNetworkName()
	if netName != "" {
		if ep, ok := ins.NetworkSettings.Networks[netName]; ok && ep != nil && ep.IPAddress != "" {
			return "http://" + ep.IPAddress + ":" + DefaultPlayersBridgePort
		}
	}
	for _, ep := range ins.NetworkSettings.Networks {
		if ep != nil && ep.IPAddress != "" && !strings.HasPrefix(ep.IPAddress, "169.254.") {
			return "http://" + ep.IPAddress + ":" + DefaultPlayersBridgePort
		}
	}
	return ""
}
