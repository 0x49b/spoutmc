package v1

import (
	"encoding/json"
	"go.uber.org/zap"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"time"
)

func broadcastContainerList() {
	for {
		time.Sleep(1 * time.Second)

		clientsMutex.Lock()
		subscriptionsMutex.Lock()

		for ws, sub := range subscriptions {
			if sub.Subscriptions[SUB_LIST] {

				replyJson, err := prepareContainerListAsJson()
				if err != nil {
					logger.Error("Failed to marshal reply", zap.Error(err))
					continue
				}

				if !safeSend(ws, string(replyJson)) {
					continue
				}
			}
		}

		clientsMutex.Unlock()
		subscriptionsMutex.Unlock()
	}
}

func broadcastContainerStats() {
	for {
		time.Sleep(1 * time.Second)

		clientsMutex.Lock()
		subscriptionsMutex.Lock()

		for ws, sub := range subscriptions {
			if sub.Subscriptions[SUB_STATS] {

				stats, err := docker.GetContainerStats(sub.ContainerId)
				if err != nil {
					logger.Error("Stats fetch failed", zap.String("id", sub.ContainerId), zap.Error(err))
					log.HandleError(err)
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
				if !safeSend(ws, string(replyJson)) {
					continue
				}
			}
		}

		subscriptionsMutex.Unlock()
		clientsMutex.Unlock()
	}
}
