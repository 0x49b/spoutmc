package v1

type Command string // string mapping

const (
	START                       Command = "start"
	STOP                        Command = "stop"
	RESTART                     Command = "restart"
	CREATE                      Command = "create"
	REMOVE                      Command = "remove"
	CONTAINERLIST               Command = "containerlist"
	HEARTBEAT                   Command = "heartbeat"
	LOGS                        Command = "logs"
	CONTAINERDETAIL             Command = "containerdetail"
	CONTAINERSTATS              Command = "containerstats"
	CONTAINERSTATSLIST          Command = "containerstatslist"
	SUBSCRIBE_CONTAINER_STATS   Command = "subscribe_container_stats"
	UNSUBSCRIBE_CONTAINER_STATS Command = "unsubscribe_container_stats"
)

type WsMessage struct {
	Command     Command `json:"type"`
	Message     string  `json:"message,omitempty"`
	ContainerId string  `json:"containerId,omitempty"`
}

type WsReply struct {
	Command     string      `json:"type"`
	Data        interface{} `json:"data"`
	Ts          int64       `json:"ts"`
	ContainerId string      `json:"containerId,omitempty"`
}
