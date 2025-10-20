package v1

import (
	"encoding/json"
	"spoutmc/internal/docker"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

func getContainerListWithDetails() []container.InspectResponse {
	var containerListWithDetails []container.InspectResponse
	containerList, err := docker.GetNetworkContainers()
	if err != nil {
		logger.Error("Cannot load containerlist", zap.Error(err))
	}
	for _, c := range containerList {
		containerDetails, err := docker.GetContainerById(c.ID)
		if err != nil {
			logger.Error("Cannot load container details", zap.Error(err))
		}
		containerListWithDetails = append(containerListWithDetails, containerDetails)
	}
	return containerListWithDetails
}

func prepareContainerListAsJson() ([]byte, error) {
	reply := WsReply{
		Command: string(CONTAINERLIST),
		Data:    getContainerListWithDetails(),
		Ts:      time.Now().Unix(),
	}
	return json.Marshal(reply)
}

func prepareContainerStatsAsJson() ([]byte, error) {
	networkContainers, err := docker.GetNetworkContainers()
	if err != nil {
		logger.Error("Cannot load network containers", zap.Error(err))
	}

	containerStatsList := make([]container.StatsResponse, 0)
	var wg sync.WaitGroup
	statsCh := make(chan container.StatsResponse, len(networkContainers))

	for _, c := range networkContainers {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			stat, err := docker.GetContainerStats(id)
			if err != nil {
				logger.Error("Cannot load container stats", zap.Error(err))
				return
			}
			statsCh <- stat
		}(c.ID)
	}

	wg.Wait()
	close(statsCh)

	for stat := range statsCh {
		containerStatsList = append(containerStatsList, stat)
	}

	reply := WsReply{
		Command: string(CONTAINERSTATSLIST),
		Data:    containerStatsList,
		Ts:      time.Now().Unix(),
	}

	return json.Marshal(reply)
}

func containerList(ws *websocket.Conn) {
	replyJson, err := prepareContainerListAsJson()
	if err != nil {
		logger.Error("Failed to marshal reply", zap.Error(err))
		return
	}

	if err := websocket.Message.Send(ws, string(replyJson)); err != nil {
		logger.Error("WebSocket write error", zap.Error(err))
	}
}

func safeSend(ws *websocket.Conn, payload string) bool {
	if err := websocket.Message.Send(ws, payload); err != nil {
		logger.Warn("WebSocket send failed, unregistering client", zap.Error(err))
		unregisterClient(ws)
		return false
	}
	return true
}
