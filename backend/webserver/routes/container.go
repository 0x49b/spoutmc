package routes

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"spoutmc/backend/docker"
)

func SetupContainerRoutes(router fiber.Router) {
	containerRouter := router.Group("/container", getContainerList)
	containerRouter.Get("/name/:name", getContainerByName)
	containerRouter.Get("/id/:id", getContainerById)
}

func getContainerList(c *fiber.Ctx) error {
	containers, err := docker.GetNetworkContainers()
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Cannot load container list",
		})
	}
	return c.JSON(containers)
}

func getContainerByName(c *fiber.Ctx) error {
	if c.Params("name") != "" {
		container, err := docker.GetContainer(c.Params("name"))
		if err != nil {
			return c.JSON(fiber.Map{
				"error": fmt.Sprintf("Cannot find container with name %s", c.Params("name")),
			})
		}
		return c.JSON(container)
	}
	return c.JSON(fiber.Map{
		"error": "cannot find container for given name",
	})
}

func getContainerById(c *fiber.Ctx) error {
	if c.Params("id") != "" {
		requestedContainer, err := docker.GetContainerById(c.Params("id"))
		if err != nil {
			return c.JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(requestedContainer)
	}

	return c.JSON(fiber.Map{
		"error": "Cannot find any Container with given ID",
	})
}
