package v1

type SubscriptionType string

const (
	SUB_DETAIL SubscriptionType = "CONTAINERDETAIL"
	SUB_LOGS   SubscriptionType = "CONTAINERLOGS"
	SUB_STATS  SubscriptionType = "CONTAINERSTATS"
	SUB_LIST   SubscriptionType = "CONTAINERLIST"
)

type ClientSubscription struct {
	ContainerId   string
	Subscriptions map[SubscriptionType]bool
}

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
	REGISTER_SUBSCRIPTIONS      Command = "register_subscription"
	UNREGISTER_SUBSCRIPTIONS    Command = "unregister_subscriptions"
	EXEC_REQUEST                Command = "exec_request"
	EXEC_RESPONSE               Command = "exec_response"
)

type WsMessage struct {
	Command       Command            `json:"type"`                    // e.g. "REGISTER_SUBSCRIPTIONS"
	ContainerId   string             `json:"containerId,omitempty"`   // used for certain commands
	Subscriptions []SubscriptionType `json:"subscriptions,omitempty"` // used for register/unregister subscriptions
	Message       string             `json:"message,omitempty"`       // used for create or general-purpose
}

type WsReply struct {
	Command     string      `json:"type"`                  // echo command type (e.g. CONTAINERSTATS)
	Data        interface{} `json:"data"`                  // flexible container for any payload
	Ts          int64       `json:"ts"`                    // timestamp
	ContainerId string      `json:"containerId,omitempty"` // optional
}
