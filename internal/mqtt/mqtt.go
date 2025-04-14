package mqtt

import (
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"os"
	"os/signal"
	"spoutmc/internal/log"
	"syscall"
)

var (
	logger = log.GetLogger()
	Broker *mqtt.Server
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

	err := Broker.AddHook(new(ServerHook), &ServerHookOptions{
		Broker: Broker,
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

	go broadcastContainerList()

	<-done
}

func ShutdownMQTT() error {
	if Broker != nil {
		logger.Info("Shutting down MQTT broker")
		return Broker.Close()
	}
	return nil
}
