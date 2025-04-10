package v1

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
	"spoutmc/internal/docker"
	"spoutmc/internal/watchdog"
)

func messageParser(message []byte, ws *websocket.Conn) {
	messageData := WsMessage{}
	if err := json.Unmarshal(message, &messageData); err != nil {
		return
	}

	switch messageData.Command {
	case START:
		docker.StartContainerById(messageData.ContainerId)
		watchdog.IncludeToWatchdog(messageData.ContainerId)
	case STOP:
		watchdog.ExcludeFromWatchdog(messageData.ContainerId)
		docker.StopContainerById(messageData.ContainerId)
	case RESTART:
		docker.RestartContainerById(messageData.ContainerId)
	case CREATE:
		createServer(messageData.Message)
	case REMOVE:
		// To be implemented
	case HEARTBEAT:
		sendHeartbeat(ws)
	case LOGS:
		sendContainerLogs(ws, messageData.ContainerId)
	case CONTAINERDETAIL:
		sendContainerDetails(ws, messageData.ContainerId)
	case CONTAINERSTATS:
		sendContainerStats(ws, messageData.ContainerId)
	case SUBSCRIBE_CONTAINER_STATS:
		// Todo; will be replace by new Subscription handling
		// registerSubscription(ws, messageData.ContainerId)
	case UNSUBSCRIBE_CONTAINER_STATS:
	// Todo; will be replace by new Subscription handling
	// unregisterSubscription(ws)
	case EXEC_REQUEST:
		fmt.Println("EXEC_REQUEST")
		executeCommands(ws, messageData)
	case REGISTER_SUBSCRIPTIONS:
		var payload struct {
			ContainerId   string             `json:"containerId"`
			Subscriptions []SubscriptionType `json:"subscriptions"`
		}
		if err := json.Unmarshal(message, &payload); err != nil {
			logger.Error("Invalid register message", zap.Error(err))
			return
		}
		registerSubscription(ws, payload.ContainerId, payload.Subscriptions)
	case UNREGISTER_SUBSCRIPTIONS:
		var payload struct {
			Subscriptions []SubscriptionType `json:"subscriptions"`
		}
		if err := json.Unmarshal(message, &payload); err != nil {
			logger.Error("Invalid unregister message", zap.Error(err))
			return
		}
		unregisterSubscriptions(ws, payload.Subscriptions)
	default:
		logger.Error("Unknown command", zap.String("command", string(messageData.Command)))
	}
}

func createServer(message string) {}
