package v1

import (
	"spoutmc/internal/log"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

var logger = log.GetLogger()

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
					log.HandleError(err)
				}

				unregisterClient(ws)
				deleteSubscription(ws)

				break
			}

			logger.Info("Got Message from Client", zap.String("msg", msg))
			messageParser([]byte(msg), ws)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
