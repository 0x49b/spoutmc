package kube

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"spoutmc/internal/kubernetes"
	"spoutmc/internal/log"
	"strconv"
	"sync"
)

var lock = sync.Mutex{}
var logger = log.GetLogger()

type Event struct {
	ID        []byte
	Data      []byte
	Event     []byte
	Retry     []byte
	Comment   []byte
	Timestamp int64
}

func RegisterKubernetesRoutes(g *echo.Group) {
	// REST
	g.GET("/kube/replicas/:name/:replicas", updateReplicas)
	g.DELETE("/kube/namespace/:namespace", deleteNamespace)
	g.PATCH("/kube/reset/:namespace", resetKube)
}

func updateReplicas(c echo.Context) error {

	replicas, _ := strconv.ParseInt(c.Param("replicas"), 10, 32)
	err := kubernetes.UpdateDeploymentReplicas(c.Param("name"), int32(replicas))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, replicas)
}

func deleteNamespace(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	err := kubernetes.DeleteNamespace(c.Param("namespace"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, "ok")
}

func resetKube(c echo.Context) error {
	err := kubernetes.ResetKube(c.Param("namespace"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, "ok")
}
