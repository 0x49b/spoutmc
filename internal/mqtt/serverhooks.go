package mqtt

import (
	"bytes"
	"encoding/json"
	"fmt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"regexp"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"sync"
	"time"
)

type ServerHookOptions struct {
	Broker *mqtt.Server
}

type ServerHook struct {
	mqtt.HookBase
	config        *ServerHookOptions
	subscriptions map[string][]string
}

var subscriptionsMu sync.RWMutex

func (h *ServerHook) ID() string {
	return "server-stats-hook"
}

func (h *ServerHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
		mqtt.OnSubscribed,
		mqtt.OnUnsubscribed,
		mqtt.OnPublished,
		mqtt.OnPublish,
	}, []byte{b})
}

func (h *ServerHook) Init(config any) error {
	h.Log.Info("initialised")
	h.subscriptions = make(map[string][]string)
	if _, ok := config.(*ServerHookOptions); !ok && config != nil {
		return mqtt.ErrInvalidConfigType
	}

	h.config = config.(*ServerHookOptions)
	h.StartStatsBroadcaster()

	if h.config.Broker == nil {
		return mqtt.ErrInvalidConfigType
	}
	return nil
}

// subscribeCallback handles messages for subscribed topics
func (h *ServerHook) subscribeCallback(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
	h.Log.Info("hook subscribed message", "client", cl.ID, "topic", pk.TopicName)
}

func (h *ServerHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	h.Log.Info("client connected", "client", cl.ID)
	return nil
}

func (h *ServerHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	if err != nil {
		h.Log.Info("client disconnected", "client", cl.ID, "expire", expire, "error", err)
	} else {
		h.Log.Info("client disconnected", "client", cl.ID, "expire", expire)
	}

}

func (h *ServerHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
	topics := make([]string, 0, len(pk.Filters))
	for _, f := range pk.Filters {
		topics = append(topics, f.Filter)
	}

	subscriptionsMu.Lock()
	h.subscriptions[cl.ID] = append(h.subscriptions[cl.ID], topics...)
	subscriptionsMu.Unlock()

	h.Log.Info(fmt.Sprintf("subscribed qos=%v", reasonCodes), "client", cl.ID, "filters", pk.Filters)
}

func (h *ServerHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	subscriptionsMu.Lock()
	defer subscriptionsMu.Unlock()

	current := h.subscriptions[cl.ID]
	newList := []string{}
	for _, topic := range current {
		keep := true
		for _, f := range pk.Filters {
			if topic == f.Filter {
				keep = false
				break
			}
		}
		if keep {
			newList = append(newList, topic)
		}
	}
	h.subscriptions[cl.ID] = newList
	h.Log.Info("unsubscribed", "client", cl.ID, "filters", pk.Filters)
}

func (h *ServerHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	// exclude inline client to not get loops
	if cl.ID == "inline" {
		return pk, nil // or just return early
	}

	h.Log.Info("received from client", "client", cl.ID, "payload", string(pk.Payload))

	return pk, nil
}

func (h *ServerHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {

	// exclude inline client to not get loops
	if cl.ID == "inline" {
		return
	}
	h.Log.Info("published to client", "client", cl.ID, "payload", string(pk.Payload))
}

func (h *ServerHook) BroadcastToChannel(topic string, payload []byte) error {

	err := h.config.Broker.Publish(topic, payload, false, 2)
	if err != nil {
		h.Log.Error("failed to broadcast", "topic", topic, "error", err)
		return err
	}
	h.Log.Info("broadcasted message", "topic", topic, "payload", string(payload))
	return nil
}

func (h *ServerHook) StartStatsBroadcaster() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C

			//fmt.Println(h.subscriptions)

			// Lock for reading the subscriptions map
			subscriptionsMu.RLock()
			for _, topics := range h.subscriptions {
				for _, topic := range topics {

					containerId, err := getServerIDFromTopic(topic)
					if err != nil {
						log.HandleError(err)
					}
					containerStats, err := docker.GetContainerStats(containerId)
					if err != nil {
						log.HandleError(err)

					}

					replyJson, err := json.Marshal(containerStats)
					if err != nil {
						log.HandleError(err)
					}

					// Broadcast dummy stats to each topic
					err = h.BroadcastToChannel(topic, replyJson)
					if err != nil {
						h.Log.Error("error broadcasting stats", "topic", topic, "err", err)
					}
				}
			}
			subscriptionsMu.RUnlock()
		}
	}()
}

func topicContainsId(path string) bool {
	re := regexp.MustCompile(`^server/[a-f0-9]{64}/stats$`)
	return re.MatchString(path)
}

func getServerIDFromTopic(topic string) (string, error) {
	re := regexp.MustCompile(`/([a-f0-9]{64})/`)
	match := re.FindStringSubmatch(topic)
	if len(match) < 2 {
		return "", fmt.Errorf("no container ID found in: %s", topic)
	}
	return match[1], nil
}
