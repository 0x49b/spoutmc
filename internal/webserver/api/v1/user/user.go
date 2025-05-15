package user

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/security"
	"spoutmc/internal/storage"
	"sync"
)

var lock = sync.Mutex{}
var logger = log.GetLogger()

// RegisterUserRoutes registers user-related API routes.
//
// @Tags user
// @Router /user [get,post]
// @Router /user/{id} [get]
// @Produce json
func RegisterUserRoutes(g *echo.Group) {
	// REST
	g.GET("/user", getUsers)
	g.GET("/user/:id", getUser)

	g.POST("/user", createUser)

}

// @Summary Create a new user
// @Description Register a new user account
// @Tags user
// @Accept json
// @Produce json
// @Param user body models.User true "User object"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user [post]
func createUser(c echo.Context) error {
	var user models.User
	var err error

	// Bind request body to user struct
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Optional: Add basic validation
	if user.DisplayName == "" || user.Email == "" || user.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "displayName, email, and password are required",
		})
	}

	user.Password, err = security.Hash(user.Password)
	if err != nil {
		logger.Error(err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Problems arised in creating user",
		})
	}

	// Save user to DB
	db := storage.GetDB()
	if err := db.Create(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}

	// Optional: remove sensitive fields before returning
	user.Password = ""

	return c.JSON(http.StatusCreated, user)
}

// @Summary Get user container info
// @Description Retrieves Docker container info for a given user ID
// @Tags user
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} interface{}
// @Failure 500 {object} map[string]string
// @Router /user/{id} [get]
func getUser(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	container, err := docker.GetContainerById(c.Param("id"))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, container)
}

// @Summary Get all users
// @Description Returns a list of all registered users
// @Tags user
// @Produce json
// @Success 200 {array} models.User
// @Failure 500 {object} map[string]string
// @Router /user [get]
func getUsers(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	db := storage.GetDB()
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		logger.Error("Failed to fetch users: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	for i := range users {
		users[i].Password = "REDACTED"
	}

	return c.JSON(http.StatusOK, users)
}
