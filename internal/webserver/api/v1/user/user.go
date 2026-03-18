package user

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"spoutmc/internal/log"
	"spoutmc/internal/minime/processor"
	"spoutmc/internal/models"
	"spoutmc/internal/security"
	"spoutmc/internal/storage"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var logger = log.GetLogger(log.ModuleUser)

var defaultAvatar = "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"

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
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user [post]
func createUser(c echo.Context) error {
	var user models.User
	var err error
	var avatarUrl string
	var avatarImageProcessed image.Image

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

	user.MinecraftID, avatarUrl, err = getMojangData(user.DisplayName)

	if err != nil {
		user.Avatar = defaultAvatar
		user.MinecraftID = uuid.Nil
	}

	avatarImageProcessed, err = processor.ProcessSkin(avatarUrl, true, true, 256)
	user.Avatar, err = processor.EncodeToBase64(avatarImageProcessed)

	if err != nil {
		logger.Error(err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Problems arised in creating useravatar",
		})
	}

	// Save user to DB
	db := storage.GetDB()
	if err := db.Create(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}

	userResponse := &models.UserResponse{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		MinecraftID: user.MinecraftID,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		Roles:       convertRolesToResponse(user.Roles),
		Avatar:      user.Avatar,
	}

	return c.JSON(http.StatusCreated, userResponse)
}

// @Summary Get user container info
// @Description Retrieves Docker container info for a given user ID
// @Tags user
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.UserResponse
// @Failure 500 {object} map[string]string
// @Router /user/{id} [get]
func getUser(c echo.Context) error {
	db := storage.GetDB()
	var user models.User

	if err := db.Preload("Roles").First(&user, "id = ?", c.Param("id")).Error; err != nil {
		logger.Error("Failed to fetch users: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	response := models.UserResponse{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		MinecraftID: user.MinecraftID,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		Roles:       convertRolesToResponse(user.Roles),
		Avatar:      user.Avatar,
	}

	return c.JSON(http.StatusOK, response)
}

// @Summary Get all users
// @Description Returns a list of all registered users
// @Tags user
// @Produce json
// @Success 200 {array} models.UserResponse
// @Failure 500 {object} map[string]string
// @Router /user [get]
func getUsers(c echo.Context) error {
	db := storage.GetDB()
	var users []models.User
	if err := db.Preload("Roles").Find(&users).Error; err != nil {
		logger.Error("Failed to fetch users: " + err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	var userResponses []models.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, models.UserResponse{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			MinecraftID: user.MinecraftID,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Roles:       convertRolesToResponse(user.Roles),
			Avatar:      user.Avatar,
		})
	}

	return c.JSON(http.StatusOK, userResponses)
}

func getMojangData(displayname string) (uuid.UUID, string, error) {
	url := fmt.Sprintf("https://playerdb.co/api/player/minecraft/%s", displayname)
	resp, err := http.Get(url)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("error making GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, "", fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("error reading response body: %w", err)
	}

	var data models.MojangResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("error parsing JSON: %w", err)
	}

	if !data.Success {
		return uuid.Nil, "", fmt.Errorf("failed to get UUID for player %s: %s", displayname, data.Message)
	}

	playerUUID, err := uuid.Parse(data.Data.Player.RawID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("error parsing UUID: %w", err)
	}

	return playerUUID, data.Data.Player.SkinTexture, nil
}

func convertRolesToResponse(roles []models.Role) []models.RoleResponse {
	roleResponses := make([]models.RoleResponse, len(roles))
	for i, role := range roles {
		roleResponses[i] = models.RoleResponse{
			Rolename: role.Rolename,
		}
	}
	return roleResponses
}
