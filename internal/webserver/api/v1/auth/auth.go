package auth

import (
	"net/http"
	"spoutmc/internal/access"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"
	"spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleUser)

// LoginRequest is the request body for login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the response for successful login
type LoginResponse struct {
	Token string              `json:"token"`
	User  models.UserResponse `json:"user"`
}

// RegisterAuthRoutes registers auth-related API routes
func RegisterAuthRoutes(g *echo.Group) {
	g.POST("/auth/login", login)
	// Verify requires JWT - register on a group with middleware
}

// RegisterAuthVerifyRoute registers the verify endpoint on a protected group
func RegisterAuthVerifyRoute(g *echo.Group) {
	g.GET("/auth/verify", verify)
}

func verify(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Service unavailable"})
	}

	var user models.User
	if err := db.Preload("Roles.Permissions").Preload("DirectPermissions").First(&user, cl.UserID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, access.BuildUserResponse(&user))
}

func login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}

	db := storage.GetDB()
	if db == nil {
		logger.Error("Database not initialized")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Service unavailable"})
	}

	var user models.User
	if err := db.Preload("Roles.Permissions").Preload("DirectPermissions").Where("email = ?", req.Email).First(&user).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	if !access.Verify(user.Password, req.Password) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	roleNames := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roleNames[i] = r.Name
	}

	permKeys := access.EffectivePermissionKeysFromUser(&user)
	token, err := access.GenerateToken(user.ID, user.Email, user.DisplayName, roleNames, permKeys)
	if err != nil {
		logger.Error("Failed to generate JWT", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create token"})
	}

	userResp := access.BuildUserResponse(&user)

	return c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  userResp,
	})
}
