package v1

import (
	"context"
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"time"
)

func executeCommands(ws *websocket.Conn, message WsMessage) {
	// 1. Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := message.Message
	cmdChan := make(chan string, 1)
	outputChan := docker.ExecCommand(ctx, message.ContainerId, cmdChan)

	// 2. Send command
	cmdChan <- cmd
	close(cmdChan)

	// 3. Wait for result
	for result := range outputChan {
		reply := WsReply{
			Command:     string(EXEC_RESPONSE),
			Data:        []string{result},
			Ts:          time.Now().Unix(),
			ContainerId: message.ContainerId,
		}
		sendReply(ws, reply)
	}
}

func sendContainerList(ws *websocket.Conn) {
	replyJson, err := prepareContainerListAsJson()
	if err != nil {
		log.HandleError(err)
	}

	if !safeSend(ws, string(replyJson)) {
		log.HandleError(errors.New("failed to send reply for container list"))
	}
}

func sendContainerLogs(ws *websocket.Conn, id string) {
	ctx := context.Background()
	logChan, err := docker.FetchDockerLogs(ctx, id)
	if err != nil {
		logger.Error("Error fetching docker logs", zap.Error(err))
		return
	}

	for logLine := range logChan {
		reply := WsReply{
			Command:     string(LOGS),
			Data:        []string{logLine},
			Ts:          time.Now().Unix(),
			ContainerId: id,
		}

		sendReply(ws, reply)
	}
}

func sendContainerDetails(ws *websocket.Conn, containerId string) {
	containerDetails, err := docker.GetContainerById(containerId)
	if err != nil {
		logger.Error("Cannot load container details", zap.Error(err))
	}

	reply := WsReply{
		Command: string(CONTAINERDETAIL),
		Data:    containerDetails,
		Ts:      time.Now().Unix(),
	}

	sendReply(ws, reply)
}

func sendContainerStats(ws *websocket.Conn, containerId string) {
	containerStats, err := docker.GetContainerStats(containerId)
	if err != nil {
		logger.Error("Cannot load container stats", zap.Error(err))
	}
	reply := WsReply{
		Command: string(CONTAINERSTATS),
		Data:    containerStats,
		Ts:      time.Now().Unix(),
	}
	sendReply(ws, reply)
}

func sendReply(ws *websocket.Conn, reply WsReply) {
	replyJson, err := json.Marshal(reply)
	if err != nil {
		log.HandleError(err)
	}
	if err = websocket.Message.Send(ws, string(replyJson)); err != nil {
		log.HandleError(err)
	}
}

func sendHeartbeat(ws *websocket.Conn) {
	if err := websocket.Message.Send(ws, "pong"); err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
}
