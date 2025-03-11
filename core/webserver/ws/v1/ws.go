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
var socket *websocket.Conn

func WebsocketHandler(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		socket = ws

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

			messageParser([]byte(msg))

			if err != nil {
				c.Logger().Error("WebSocket write error", zap.Error(err))
				break // Exit the loop if writing fails
			}

		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func messageParser(message []byte) {

	messageData := WsMessage{}
	err := json.Unmarshal(message, &messageData)
	if err != nil {
		return
	}

	switch messageData.Command {
	case CONTAINERLIST: //
		containerList()
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
	default:
		logger.Error("Unknown command", zap.String("", string(messageData.Command)))
	}
}

func containerList() {

	containerList, err := docker.GetNetworkContainers()
	if err != nil {
		logger.Error("Cannot load containerlist", zap.Error(err))
	}

	containerListJson, err := json.Marshal(containerList)

	reply := WsReply{
		Command: "containerlist",
		Data:    containerListJson,
	}

	replyJson, err := json.Marshal(reply)

	logger.Info("Sending Data: ", zap.ByteString("reply", replyJson))
	err = websocket.Message.Send(socket, reply)
	if err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
}
