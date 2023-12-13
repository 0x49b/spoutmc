package v1

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"spoutmc/backend/docker"
	"spoutmc/backend/log"
	"spoutmc/backend/webserver/api/v1/model"
)

var logger = log.New()

func RegisterContainerAPI(v1Group *echo.Group) {
	g := v1Group.Group("/container")
	g.GET("", getContainerList)
	g.GET("/name/:name", getContainerByName)
	g.GET("/id/:id", getContainerById)

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
