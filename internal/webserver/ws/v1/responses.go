package v1

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
	"spoutmc/internal/docker"
	"time"
)

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

		replyJson, err := json.Marshal(reply)
		if err != nil {
			logger.Error("Cannot marshal reply", zap.Error(err))
			continue
		}

		if err := websocket.Message.Send(ws, string(replyJson)); err != nil {
			logger.Error("WebSocket write error", zap.Error(err))
			break
		}
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

	replyJson, err := json.Marshal(reply)
	if err != nil {
		logger.Error("Cannot marshal reply", zap.Error(err))
	}
	if err := websocket.Message.Send(ws, string(replyJson)); err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
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
	replyJson, err := json.Marshal(reply)
	if err != nil {
		logger.Error("Cannot marshal reply", zap.Error(err))
	}
	if err := websocket.Message.Send(ws, string(replyJson)); err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
}

func sendHeartbeat(ws *websocket.Conn) {
	if err := websocket.Message.Send(ws, "pong"); err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
}
