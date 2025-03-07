package v1

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dcl "github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	"spoutmc/backend/utils"
	"spoutmc/backend/watchdog"
	"spoutmc/backend/webserver/api/v1/model"
	"time"
	"unicode/utf8"
)

var logger = log.CreateLogger()
var upgrader = websocket.Upgrader{}
var inout chan []byte
var output chan []byte

type CommandRequest struct {
	Command string `json:"command"`
}

type NewServerRequest struct {
	ServerName string `json:"servername"`
}

func RegisterContainerAPI(v1Group *echo.Group) {
	g := v1Group.Group("/container")
	g.GET("", getContainerList)
	g.GET("/withDetails", getContainerListWithDetails)
	g.GET("/name/:name", getContainerByName)
	g.GET("/id/:id", getContainerById)
	g.GET("/logs/:name", echoLogs)

	g.GET("/start/:id", startContainerById)
	g.GET("/stop/:id", stopContainerById)
	g.GET("/restart/:id", restartContainerById)

	g.GET("/bannedPlayers/:id", listBannedPlayers)
	g.GET("/opPlayers/:id", listOpPlayers)

	g.POST("/command/:id", executeCommand)
	g.POST("/create", createNewServer)

	g.DELETE("/id/:id", removeServer)

	g.GET("/plugins", listPlugins)
	g.GET("/plugins/:id", listPluginsForContainer)

	g.GET("/reset/:id", resetContainer)
}

func resetContainer(c echo.Context) error {

	containerId := c.Param("id")
	logger.Info(fmt.Sprintf("Reset ordered for %s", containerId))

	container, err := docker.GetContainerById(containerId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &model.APIError{E: err.Error()})
	}

	watchdog.ExcludeFromWatchdog(containerId)
	docker.StopContainerById(containerId)

	dirPath := filepath.Join(container.Mounts[0].Source, "*.jar")

	files, err := filepath.Glob(dirPath)

	for _, f := range files {
		logger.Info(fmt.Sprintf("Removing --> %s", f))
		if err := os.Remove(f); err != nil {
			panic(err)
		}
	}

	watchdog.IncludeToWatchdog(containerId)
	return c.JSON(http.StatusOK, "")
}

type PluginsList struct {
	Name    string        `json:"name"`
	Id      string        `json:"id"`
	Plugins []*utils.File `json:"plugins"`
}

func listPlugins(c echo.Context) error {
	var pluginList []PluginsList

	networkContainers, err := docker.GetNetworkContainers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &model.APIError{E: err.Error()})
	}

	for _, nc := range networkContainers {

		containerDetails, err := docker.GetContainerById(nc.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, &model.APIError{E: err.Error()})
		}

		containerPath := containerDetails.Mounts[0].Source
		dirPath := filepath.Join(containerPath, "plugins")

		plugins := utils.FileToJSON(dirPath)

		pluginList = append(pluginList, PluginsList{
			Name:    containerDetails.Config.Hostname,
			Id:      containerDetails.ID,
			Plugins: plugins.Children,
		})

	}
	return c.JSON(http.StatusOK, pluginList)
}

func listPluginsForContainer(c echo.Context) error {
	containerId := c.Param("id")
	containerDetail, err := docker.GetContainerById(containerId)
	if err != nil {
		logger.Error("Cannot load containerdetails", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &model.APIError{E: err.Error()})
	}

	containerPath := containerDetail.Mounts[0].Source
	dirPath := filepath.Join(containerPath, "plugins")

	plugins := utils.FileToJSON(dirPath)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, &model.APIError{E: err.Error()})
	}
	return c.JSON(http.StatusOK, plugins)
}

func getContainerListWithDetails(c echo.Context) error {

	var containerListWithDetails []types.ContainerJSON
	containerList, err := docker.GetNetworkContainers()
	if err != nil {
		logger.Error("Cannot load containerlist", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &model.APIError{E: err.Error()})
	}
	for _, c := range containerList {
		containerDetails, err := docker.GetContainerById(c.ID)
		if err != nil {
			return err
		}
		containerListWithDetails = append(containerListWithDetails, containerDetails)
	}

	return c.JSON(http.StatusOK, containerListWithDetails)
}

func removeServer(c echo.Context) error {
	containerId := c.Param("id")

	removedContainer, err := docker.DeleteContainer(containerId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Could not delete container %s", err),
			})
	}

	return c.JSON(http.StatusOK, removedContainer)
}

func createNewServer(c echo.Context) error {

	var requestBody NewServerRequest
	if err := c.Bind(&requestBody); err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Bad JSON Format"),
			})
	}

	serverName := requestBody.ServerName
	newContainer, err := docker.CreateContainer(serverName, false, false) // to do add proxy and lobby flag
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Could not create new Server %s", serverName),
			})
	}

	return c.JSON(http.StatusOK, newContainer)
}

func executeCommand(c echo.Context) error {

	var requestBody CommandRequest

	if err := c.Bind(&requestBody); err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Bad JSON Format"),
			})
	}

	containerId := c.Param("id")
	command := requestBody.Command

	logger.Info(fmt.Sprintf("Sending command [%s] to container %s ", command, containerId))
	execCommand, err := docker.ExecCommand(containerId, command)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: "Something went wrong",
			})
	}

	if execCommand != 0 {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Something went wrong, Exit Code was %d", execCommand),
			})
	}

	container, err := docker.GetContainerById(containerId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: "Something went wrong",
			})
	}

	return c.JSON(http.StatusOK, container)
}

func listBannedPlayers(c echo.Context) error {
	cont, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
			})
	}
	path := fmt.Sprintf("%s%c%s", cont.Mounts[0].Source, os.PathSeparator, "banned-players.json")
	bannedPlayersFile, err := os.ReadFile(path)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find banned-players.json at %s", path),
			})
	}
	return c.JSON(http.StatusOK, string(bannedPlayersFile))
}

func listOpPlayers(c echo.Context) error {
	cont, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
			})
	}
	path := fmt.Sprintf("%s%c%s", cont.Mounts[0].Source, os.PathSeparator, "ops.json")
	opsFile, err := os.ReadFile(path)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find ops.json at %s", path),
			})
	}
	return c.JSON(http.StatusOK, string(opsFile))
}

func startContainerById(c echo.Context) error {

	containerId := c.Param("id")
	container, err := docker.GetContainerById(containerId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
			})
	}

	watchdog.IncludeToWatchdog(containerId)
	return c.JSON(http.StatusOK, container)
}

func stopContainerById(c echo.Context) error {
	containerId := c.Param("id")
	watchdog.ExcludeFromWatchdog(containerId)

	docker.StopContainerById(containerId)
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
	containerId := c.Param("id")
	docker.RestartContainerById(containerId)

	container, err := docker.GetContainerById(containerId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			&model.APIError{
				E: fmt.Sprintf("Cannot find container with name %s", c.Param("name")),
			})
	}

	watchdog.IncludeToWatchdog(containerId)
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
	execConfig := container.ExecOptions{
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
	execAttachConfig := container.ExecAttachOptions{
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
