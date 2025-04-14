package mqtt

import (
	"encoding/json"
	"github.com/docker/docker/api/types/container"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"syscall"
	"time"
)

var (
	logger = log.GetLogger()
	Broker *mqtt.Server
)

type WsReply struct {
	Command     string      `json:"type"`                  // echo command type (e.g. CONTAINERSTATS)
	Data        interface{} `json:"data"`                  // flexible container for any payload
	Ts          int64       `json:"ts"`                    // timestamp
	ContainerId string      `json:"containerId,omitempty"` // optional
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

func StartMQTT() {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- true
	}()

	Broker = mqtt.New(&mqtt.Options{
		InlineClient: true,
		Logger:       log.GetSLogger(),
	})
	_ = Broker.AddHook(new(auth.AllowHook), nil)
	err := Broker.AddHook(new(ExampleHook), &ExampleHookOptions{
		Server: Broker,
	})
	if err != nil {
		log.HandleError(err)
	}

	tcp := listeners.NewTCP(listeners.Config{ID: "t1", Address: ":1883"})
	err = Broker.AddListener(tcp)
	if err != nil {
		log.HandleError(err)
	}

	ws := listeners.NewWebsocket(listeners.Config{ID: "ws", Address: ":9001"})
	err = Broker.AddListener(ws)
	if err != nil {
		log.HandleError(err)
	}

	go func() {
		err := Broker.Serve()
		if err != nil {
			log.HandleError(err)
		}
	}()

	logger.Info("🚀 Embedded MQTT broker running on ports 1883 (TCP) and 9001 (WS)")

	go func() {
		for {
			replyJson, err := prepareContainerListAsJson()
			if err != nil {
				log.HandleError(err)
			}
			err = Broker.Publish("server", replyJson, false, 0)
			if err != nil {
				log.HandleError(err)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	<-done
}

func ShutdownMQTT() error {
	if Broker != nil {
		logger.Info("Shutting down MQTT broker")
		return Broker.Close()
	}
	return nil
}

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
