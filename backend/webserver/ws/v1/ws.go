package v1

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var (
	upgrader = websocket.Upgrader{}
)

var ws *websocket.Conn

func RegisterWS(g *echo.Group) {
	g.GET("", wsHandler)
}

func wsHandler(c echo.Context) error {
	wsconnection, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	ws = wsconnection
	if err != nil {
		return err
	}
	defer wsconnection.Close()
	defer ws.Close()

	for {

		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error(err)
		}
		messageParser(msg)

		err = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("got %s", msg)))

		// Write
		err = ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
		if err != nil {
			c.Logger().Error(err)
		}
	}
}

func messageParser(message []byte) {

	messageData := WsMessage{}
	err := json.Unmarshal(message, &messageData)
	if err != nil {
		return
	}

	switch messageData.Command {
	case START:
		// Do start of container
		break
	case STOP:
		// Do stop of container
		break
	case RESTART:
		// Do restart of container
		break
	case CREATE:
		// Do create of container
		break
	case REMOVE:
		// Do remove of container
		break
	}
}
