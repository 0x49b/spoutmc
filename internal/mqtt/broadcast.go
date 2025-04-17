package mqtt

import (
	"encoding/json"
	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
	"reflect"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"time"
)

func prepareContainerListAsJson() ([]byte, error) {
	reply := WsReply{
		Command: string(CONTAINERLIST),
		Data:    getContainerListWithDetails(),
		Ts:      time.Now().Unix(),
	}
	return json.Marshal(reply)
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

func containerListsEqual(a, b []container.InspectResponse) bool {
	return reflect.DeepEqual(a, b)
}

func broadcastContainerList() {
	//var lastList []container.InspectResponse

	for {
		currentList := getContainerListWithDetails()

		//if !containerListsEqual(lastList, currentList) {
		replyJson, err := json.Marshal(WsReply{
			Command: CONTAINERLIST,
			Data:    currentList,
			Ts:      time.Now().Unix(),
		})
		if err != nil {
			log.HandleError(err)
			continue
		}
		err = Broker.Publish(SERVERLIST, replyJson, false, 0)
		if err != nil {
			log.HandleError(err)
			continue
		}
		//lastList = currentList
		//}
		time.Sleep(1 * time.Second)
	}
}

func sendContainerList() {
	currentList := getContainerListWithDetails()
	replyJson, err := json.Marshal(WsReply{
		Command: CONTAINERLIST,
		Data:    currentList,
		Ts:      time.Now().Unix(),
	})
	if err != nil {
		log.HandleError(err)
	}
	err = Broker.Publish(SERVERLIST, replyJson, false, 0)
	if err != nil {
		log.HandleError(err)
	}
}
