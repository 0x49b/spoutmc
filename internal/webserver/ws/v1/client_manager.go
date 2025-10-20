package v1

import (
	"sync"

	"golang.org/x/net/websocket"
)

var (
	clients            = make(map[*websocket.Conn]struct{})
	clientsMutex       sync.Mutex
	subscriptions      = make(map[*websocket.Conn]*ClientSubscription)
	subscriptionsMutex sync.Mutex
)

func registerClient(ws *websocket.Conn) {
	clientsMutex.Lock()
	clients[ws] = struct{}{}
	clientsMutex.Unlock()
}

func unregisterClient(ws *websocket.Conn) {
	clientsMutex.Lock()
	delete(clients, ws)
	clientsMutex.Unlock()
}

func registerSubscription(ws *websocket.Conn, containerId string, subs []SubscriptionType) {
	subscriptionsMutex.Lock()
	defer subscriptionsMutex.Unlock()

	if _, exists := subscriptions[ws]; !exists {
		subscriptions[ws] = &ClientSubscription{
			ContainerId:   containerId,
			Subscriptions: make(map[SubscriptionType]bool),
		}
	}

	for _, sub := range subs {
		subscriptions[ws].Subscriptions[sub] = true
	}

	// Update containerId if needed
	subscriptions[ws].ContainerId = containerId
}

func unregisterSubscriptions(ws *websocket.Conn, subs []SubscriptionType) {
	subscriptionsMutex.Lock()
	defer subscriptionsMutex.Unlock()

	if sub, exists := subscriptions[ws]; exists {
		for _, s := range subs {
			delete(sub.Subscriptions, s)
		}
		// if there are no subscriptions left, remove the client from subscriptions
		if len(sub.Subscriptions) == 0 {
			delete(subscriptions, ws)
		}
	}
}

func deleteSubscription(ws *websocket.Conn) {
	subscriptionsMutex.Lock()
	delete(subscriptions, ws)
	subscriptionsMutex.Unlock()
}
