package v1

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
	"spoutmc/core/docker"
	"spoutmc/core/log"
)

var logger = log.GetLogger()

func WebsocketHandler(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			// Read
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				if err.Error() == "EOF" {
					logger.Info("Client disconnected gracefully")
				} else {
					logger.Error("WebSocket read error", zap.Error(err))
				}
				break // Exit the loop if an error occurs
			}

			logger.Info("Got Message from Client: ", zap.String("msg", msg))

			messageParser([]byte(msg), ws)

			if err != nil {
				c.Logger().Error("WebSocket write error", zap.Error(err))
				break // Exit the loop if writing fails
			}

		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func messageParser(message []byte, ws *websocket.Conn) {

	messageData := WsMessage{}
	err := json.Unmarshal(message, &messageData)
	if err != nil {
		return
	}

	switch messageData.Command {
	case CONTAINERLIST: //
		containerList(ws)
		break
	case START:
		// Do start of container
		break
	case STOP:
		// Do stop of container
		break
	case RESTART:
		// Do restart of container
		break
	case CREATE:
		// Do create of container
		break
	case REMOVE:
		// Do remove of container
		break
	case HEARTBEAT:
		sendHeartbeat(ws)
		break
	default:
		logger.Error("Unknown command", zap.String("", string(messageData.Command)))
	}
}

func sendHeartbeat(ws *websocket.Conn) {
	err := websocket.Message.Send(ws, "pong") // Ensure it's a string
	if err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
}

func containerList(ws *websocket.Conn) {
	containerListSummary, err := docker.GetNetworkContainers()
	if err != nil {
		logger.Error("Cannot load container list", zap.Error(err))
		return
	}

	reply := WsReply{
		Command: "containerlist",
		Data:    containerListSummary, // Send as JSON, not a byte array
	}

	replyJson, err := json.Marshal(reply)
	if err != nil {
		logger.Error("Failed to marshal reply", zap.Error(err))
		return
	}

	err = websocket.Message.Send(ws, string(replyJson)) // Ensure it's a string
	if err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
}
