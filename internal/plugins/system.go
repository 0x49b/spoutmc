package plugins

type ServerKind string

const (
	ServerKindProxy ServerKind = "proxy"
	ServerKindLobby ServerKind = "lobby"
	ServerKindGame  ServerKind = "game"
)

type SystemPluginEntry struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url"`
	Kinds       []ServerKind `json:"kinds"`
}

var SystemPlugins = []SystemPluginEntry{
	/*{
		ID:          "spoutmc-bridge",
		Name:        "SpoutMC Bridge",
		Description: "Bridge between Velocity and SpoutMC",
		URL:         "https://github.com/0x49b/spoutmc/releases/download/v0.0.7/velocity-players-bridge-0.0.7.jar",
		Kinds:       []ServerKind{ServerKindProxy},
	},*/
}

func SystemURLsForKind(kind ServerKind) []string {
	var out []string
	for _, e := range SystemPlugins {
		for _, k := range e.Kinds {
			if k == kind {
				if e.URL != "" {
					out = append(out, e.URL)
				}
				break
			}
		}
	}
	return out
}

func KindFromSpoutServer(proxy, lobby bool) ServerKind {
	if proxy {
		return ServerKindProxy
	}
	if lobby {
		return ServerKindLobby
	}
	return ServerKindGame
}
