package v1

type Command string // string mapping

const (
	START         Command = "start"
	STOP          Command = "stop"
	RESTART       Command = "restart"
	CREATE        Command = "create"
	REMOVE        Command = "remove"
	CONTAINERLIST Command = "containerlist"
)

type WsMessage struct {
	Command     Command `json:"type"`
	Message     string  `json:"message,omitempty"`
	ContainerId string  `json:"containerId,omitempty"`
}

type WsReply struct {
	Command string `json:"type"`
	Data    []byte `json:"data"`
}
