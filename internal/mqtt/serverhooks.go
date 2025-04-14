package mqtt

import (
	"bytes"
	"fmt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type ServerHookOptions struct {
	Broker *mqtt.Server
}

type ServerHook struct {
	mqtt.HookBase
	config        *ServerHookOptions
	subscriptions map[string][]string
}

func (h *ServerHook) ID() string {
	return "server-hook"
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

	// Example demonstrating how to subscribe to a topic within the hook.
	h.config.Broker.Subscribe("hook/direct/publish", 1, h.subscribeCallback)

	// Example demonstrating how to publish a message within the hook
	err := h.config.Broker.Publish("hook/direct/publish", []byte("packet hook message"), false, 0)
	if err != nil {
		h.Log.Error("hook.publish", "error", err)
	}

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
	h.subscriptions[cl.ID] = append(h.subscriptions[cl.ID], topics...)
	h.Log.Info(fmt.Sprintf("subscribed qos=%v", reasonCodes), "client", cl.ID, "filters", pk.Filters)
}

func (h *ServerHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	current := h.subscriptions[cl.ID]
	newList := []string{}

	// Remove the unsubscribed filters
	for _, topic := range current {
		shouldKeep := true
		for _, f := range pk.Filters {
			if topic == f.Filter {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			newList = append(newList, topic)
		}
	}

	h.subscriptions[cl.ID] = newList
	h.Log.Info("unsubscribed", "client", cl.ID, "filters", pk.Filters)
}

func (h *ServerHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	h.Log.Info("received from client", "client", cl.ID, "payload", string(pk.Payload))

	pkx := pk
	if string(pk.Payload) == "hello" {
		pkx.Payload = []byte("hello world")
		h.Log.Info("received modified packet from client", "client", cl.ID, "payload", string(pkx.Payload))
	}

	return pkx, nil
}

func (h *ServerHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info("published to client", "client", cl.ID, "payload", string(pk.Payload))
}
