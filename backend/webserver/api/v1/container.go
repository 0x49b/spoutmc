package v1

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	dcl "github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"io"
	"net/http"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	"spoutmc/backend/webserver/api/v1/model"
	"time"
	"unicode/utf8"
)

var logger = log.New()
var upgrader = websocket.Upgrader{}
var inout chan []byte
var output chan []byte

func RegisterContainerAPI(v1Group *echo.Group) {
	g := v1Group.Group("/container")
	g.GET("", getContainerList)
	g.GET("/name/:name", getContainerByName)
	g.GET("/id/:id", getContainerById)
	g.GET("/logs/:name", echoLogs)

	g.GET("/start/:id", startContainerById)
	g.GET("/stop/:id", stopContainerById)
	g.GET("/restart/:id", restartContainerById)
}

func startContainerById(c echo.Context) error {
	docker.StartContainerById(c.Param("id"))

	container, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
			})
	}
	return c.JSON(http.StatusOK, container)
}

func stopContainerById(c echo.Context) error {
	docker.StopContainerById(c.Param("id"))

	container, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
			})
	}
	return c.JSON(http.StatusOK, container)

}

func restartContainerById(c echo.Context) error {
	docker.RestartContainerById(c.Param("id"))

	container, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
			})
	}
	return c.JSON(http.StatusOK, container)

}

// c echo.Context
// w http.ResponseWriter, r *http.Request
func echoLogs(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logger.Error("", zap.Error(err))
		return err
	}
	defer conn.Close()

	cli, err := dcl.NewClientWithOpts()
	if err != nil {
		logger.Error("", zap.Error(err))
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return err
	}

	ctx := context.Background()
	execConfig := types.ExecConfig{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Cmd:          []string{"/bin/sh"},
		Tty:          true,
		Detach:       false,
	}

	//set target container
	exec, err := cli.ContainerExecCreate(ctx, c.Param("name"), execConfig)
	if err != nil {
		logger.Error("", zap.Error(err))
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return err
	}
	execAttachConfig := types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	}
	containerConn, err := cli.ContainerExecAttach(ctx, exec.ID, execAttachConfig)
	if err != nil {
		logger.Error("", zap.Error(err))
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return err
	}
	defer containerConn.Close()

	bufin := bufio.NewReader(containerConn.Reader)

	// Write to docker container
	go func(w io.WriteCloser) {
		for {
			data, ok := <-inout
			//log.Println("Received to send to docker", data)
			logger.Info("Received to send to docker")
			if !ok {
				fmt.Println("!ok")
				w.Close()
			}

			w.Write(append(data, '\n'))
		}
	}(containerConn.Conn)

	// Received of Container Docker
	go func() {
		for {
			buffer := make([]byte, 4096, 4096)
			c, err := bufin.Read(buffer)
			if err != nil {
				fmt.Println(err)
			}
			//c, err := containerConn.Reader.Read(buffer)
			if c > 0 {
				output <- buffer[:c]
			}
			if c == 0 {
				output <- []byte{' '}
			}
			if err != nil {
				break
			}
		}
	}()

	for {
		conn.CloseHandler()
		mt, message, err := conn.ReadMessage()
		fmt.Println(mt)
		if err != nil {
			fmt.Println("read:", err)
			break
		} else {
			fmt.Printf("recv: %s\n", message)
			inout <- message
			select {
			case data := <-output:
				stringData := string(data[:])
				if !utf8.ValidString(stringData) {
					v := make([]rune, 0, len(stringData))
					for i, r := range stringData {
						if r == utf8.RuneError {
							_, size := utf8.DecodeRuneInString(stringData[i:])
							if size == 1 {
								continue
							}
						}
						v = append(v, r)
					}
					stringData = string(v)
				}
				err = conn.WriteMessage(mt, []byte(stringData))
				if err != nil {
					fmt.Println("write:", err)
				}

			case <-time.After(time.Second * 1):
				fmt.Println("Timeout")
			}
		}
	}

	return nil
}

func streamLogs(c echo.Context) error {

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	if c.Param("name") != "" {
		go docker.StreamLogsFromContainer(c.Param("name"))
	}

	for {
		// Write
		err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
		if err != nil {
			c.Logger().Error(err)
		}

		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error(err)
		}
		fmt.Printf("%s\n", msg)
	}

	return nil

	/*if c.Param("name") != "" {
		// docker.StreamLogsFromContainer(c.Param("name"))

		websocket.Handler(func(ws *websocket.Conn) {
			defer ws.Close()
			for {
				// Write
				err := websocket.Message.Send(ws, "Hello, Client!")
				if err != nil {
					c.Logger().Error(err)
				}

				// Read
				msg := ""
				err = websocket.Message.Receive(ws, &msg)
				if err != nil {
					c.Logger().Error(err)
				}
				fmt.Printf("%s\n", msg)
			}
		}).ServeHTTP(c.Response(), c.Request())
		return nil

	}
	*/
	return nil

}

func getContainerList(c echo.Context) error {

	containerList, err := docker.GetNetworkContainers()

	if err != nil {
		logger.Error("Cannot load containerlist", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &model.APIError{E: err.Error()})
	}

	return c.JSON(http.StatusOK, containerList)
}

func getContainerByName(c echo.Context) error {
	if c.Param("name") != "" {
		container, err := docker.GetContainer(c.Param("name"))
		if err != nil {
			return c.JSON(http.StatusInternalServerError,
				&model.APIError{
					E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
				})
		}
		return c.JSON(http.StatusOK, container)
	}
	return c.JSON(http.StatusInternalServerError,
		&model.APIError{
			E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
		})
}

func getContainerById(c echo.Context) error {
	if c.Param("id") != "" {
		requestedContainer, err := docker.GetContainerById(c.Param("id"))
		if err != nil {
			return c.JSON(http.StatusInternalServerError,
				&model.APIError{
					E: err.Error(),
				})
		}

		return c.JSON(http.StatusOK, requestedContainer)
	}

	return c.JSON(http.StatusInternalServerError,
		&model.APIError{
			E: "Cannot find any Container with given ID",
		})
}
