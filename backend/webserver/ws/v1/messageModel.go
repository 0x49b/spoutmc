package v1

type Command string // string mapping

const (
	START   Command = "start"
	STOP    Command = "stop"
	RESTART Command = "restart"
	CREATE  Command = "create"
	REMOVE  Command = "remove"
)

type WsMessage struct {
	Command     Command `json:"type"` // START, STOP, RESTART, CREATE, REMOVE, ...
	Message     string  `json:"message,omitempty"`
	ContainerId string  `json:"containerId"`
}
