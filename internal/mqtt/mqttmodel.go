package mqtt

type WsReply struct {
	Command     string      `json:"type"`                  // echo command type (e.g. CONTAINERSTATS)
	Data        interface{} `json:"data"`                  // flexible container for any payload
	Ts          int64       `json:"ts"`                    // timestamp
	ContainerId string      `json:"containerId,omitempty"` // optional
}

// Message types
const (
	START                       string = "start"
	STOP                        string = "stop"
	RESTART                     string = "restart"
	CREATE                      string = "create"
	REMOVE                      string = "remove"
	CONTAINERLIST               string = "containerlist"
	HEARTBEAT                   string = "heartbeat"
	LOGS                        string = "logs"
	CONTAINERDETAIL             string = "containerdetail"
	CONTAINERSTATS              string = "containerstats"
	CONTAINERSTATSLIST          string = "containerstatslist"
	SUBSCRIBE_CONTAINER_STATS   string = "subscribe_container_stats"
	UNSUBSCRIBE_CONTAINER_STATS string = "unsubscribe_container_stats"
	REGISTER_SUBSCRIPTIONS      string = "register_subscription"
	UNREGISTER_SUBSCRIPTIONS    string = "unregister_subscriptions"
	EXEC_REQUEST                string = "exec_request"
	EXEC_RESPONSE               string = "exec_response"
)

// Channels
const (
	SERVERLIST string = "server"
)
