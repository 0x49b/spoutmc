package v1

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types/container"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
	"spoutmc/core/docker"
	"spoutmc/core/log"
	"spoutmc/core/watchdog"
)

var (
	logger             = log.GetLogger()
	clients            = make(map[*websocket.Conn]struct{})
	clientsMutex       sync.Mutex
	subscriptions      = make(map[*websocket.Conn]string) // ws -> containerId
	subscriptionsMutex sync.Mutex
)

func WebsocketHandler(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		registerClient(ws)
		defer unregisterClient(ws)

		for {
			msg := ""
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				if err.Error() == "EOF" {
					logger.Info("Client disconnected gracefully")
				} else {
					logger.Error("WebSocket read error", zap.Error(err))
				}
				break
			}

			logger.Info("Got Message from Client", zap.String("msg", msg))
			messageParser([]byte(msg), ws)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func registerClient(ws *websocket.Conn) {
	clientsMutex.Lock()
	clients[ws] = struct{}{}
	clientsMutex.Unlock()
}

func unregisterClient(ws *websocket.Conn) {
	clientsMutex.Lock()
	delete(clients, ws)
	clientsMutex.Unlock()

	subscriptionsMutex.Lock()
	delete(subscriptions, ws)
	subscriptionsMutex.Unlock()

	ws.Close()
}

func messageParser(message []byte, ws *websocket.Conn) {
	messageData := WsMessage{}
	if err := json.Unmarshal(message, &messageData); err != nil {
		return
	}

	switch messageData.Command {
	case CONTAINERLIST:
		containerList(ws)
	case START:
		docker.StartContainerById(messageData.ContainerId)
		watchdog.IncludeToWatchdog(messageData.ContainerId)
	case STOP:
		watchdog.ExcludeFromWatchdog(messageData.ContainerId)
		docker.StopContainerById(messageData.ContainerId)
	case RESTART:
		docker.RestartContainerById(messageData.ContainerId)
	case CREATE:
		// Do create of container
		createServer(messageData.Message)
	case REMOVE:
		// Do remove of container
	case HEARTBEAT:
		sendHeartbeat(ws)
	case LOGS:
		//send logs for container
		sendContainerLogs(ws, messageData.ContainerId)
		break
	case CONTAINERDETAIL:
		sendContainerDetails(ws, messageData.ContainerId)
		break
	case CONTAINERSTATS:
		sendContainerStats(ws, messageData.ContainerId)
		break
	case SUBSCRIBE_CONTAINER_STATS:
		subscriptionsMutex.Lock()
		subscriptions[ws] = messageData.ContainerId
		subscriptionsMutex.Unlock()
		break
	case UNSUBSCRIBE_CONTAINER_STATS:
		subscriptionsMutex.Lock()
		delete(subscriptions, ws)
		subscriptionsMutex.Unlock()
		break
	default:
		logger.Error("Unknown command", zap.String("command", string(messageData.Command)))
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
			Data:        []string{logLine}, // Send as slice for consistency
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

func createServer(commandData string) {

}

func sendHeartbeat(ws *websocket.Conn) {
	if err := websocket.Message.Send(ws, "pong"); err != nil {
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

	replyJson, err := json.Marshal(reply)
	if err != nil {
		return []byte{}, err
	}

	return replyJson, nil
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

	replyJson, err := json.Marshal(reply)
	if err != nil {
		return []byte{}, err
	}

	return replyJson, nil
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

func broadcastContainerList() {

	for {
		time.Sleep(1 * time.Second)

		replyJson, err := prepareContainerListAsJson()
		if err != nil {
			logger.Error("Failed to marshal reply", zap.Error(err))
			continue
		}
		clientsMutex.Lock()
		for ws := range clients {
			if err := websocket.Message.Send(ws, string(replyJson)); err != nil {
				logger.Error("WebSocket write error", zap.Error(err))
				unregisterClient(ws)
			}
		}
		clientsMutex.Unlock()
	}
}

func broadcastContainerStats() {
	for {
		time.Sleep(1 * time.Second)

		clientsMutex.Lock()
		subscriptionsMutex.Lock()

		// Create a map: containerId -> []ws
		containerToClients := make(map[string][]*websocket.Conn)
		for ws, containerId := range subscriptions {
			containerToClients[containerId] = append(containerToClients[containerId], ws)
		}

		for containerId, clientList := range containerToClients {
			stats, err := docker.GetContainerStats(containerId)
			if err != nil {
				logger.Error("Cannot load container stats", zap.String("id", containerId), zap.Error(err))
				continue
			}
			reply := WsReply{
				Command: string(CONTAINERSTATS),
				Data:    stats,
				Ts:      time.Now().Unix(),
			}
			replyJson, err := json.Marshal(reply)
			if err != nil {
				logger.Error("Marshal error", zap.Error(err))
				continue
			}

			for _, ws := range clientList {
				if err := websocket.Message.Send(ws, string(replyJson)); err != nil {
					logger.Error("WebSocket write error", zap.Error(err))
					unregisterClient(ws)
					delete(subscriptions, ws)
				}
			}
		}

		subscriptionsMutex.Unlock()
		clientsMutex.Unlock()
	}
}

func init() {
	go broadcastContainerList()
	go broadcastContainerStats()
}
