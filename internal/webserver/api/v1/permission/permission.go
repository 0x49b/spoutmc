package permission

import (
	"errors"
	"net/http"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"
	"spoutmc/internal/webserver/guards"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var logger = log.GetLogger(log.ModuleUser)

// RegisterPermissionRoutes registers permission list and admin CRUD.
func RegisterPermissionRoutes(g *echo.Group) {
	g.GET("/permission", listPermissions)
	g.POST("/permission", createPermission)
	g.PUT("/permission/:id", updatePermission)
	g.DELETE("/permission/:id", deletePermission)
}

func requireAdmin(c echo.Context) error { return guards.RequireAdmin(c) }

func listPermissions(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database not available"})
	}

	var perms []models.Permission
	if err := db.Order("key").Find(&perms).Error; err != nil {
		logger.Error("Failed to list permissions", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list permissions"})
	}

	out := make([]models.PermissionResponse, len(perms))
	for i, p := range perms {
		out[i] = models.PermissionResponse{ID: p.ID, Key: p.Key, Description: p.Description}
	}
	return c.JSON(http.StatusOK, out)
}

type createPermissionBody struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

func createPermission(c echo.Context) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	var body createPermissionBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	key := strings.TrimSpace(body.Key)
	if key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
	}

	db := storage.GetDB()
	var existing models.Permission
	err := db.Where("key = ?", key).First(&existing).Error
	if err == nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Permission key already exists"})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("Permission lookup failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
	}

	p := models.Permission{Key: key, Description: strings.TrimSpace(body.Description)}
	if err := db.Create(&p).Error; err != nil {
		logger.Error("Failed to create permission", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create permission"})
	}

	return c.JSON(http.StatusCreated, models.PermissionResponse{
		ID: p.ID, Key: p.Key, Description: p.Description,
	})
}

type updatePermissionBody struct {
	Key         *string `json:"key"`
	Description *string `json:"description"`
}

func updatePermission(c echo.Context) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	var body updatePermissionBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	db := storage.GetDB()
	var p models.Permission
	if err := db.First(&p, c.Param("id")).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Permission not found"})
	}

	if body.Key != nil {
		nk := strings.TrimSpace(*body.Key)
		if nk == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "key cannot be empty"})
		}
		if nk != p.Key {
			var clash models.Permission
			err := db.Where("key = ? AND id <> ?", nk, p.ID).First(&clash).Error
			if err == nil {
				return c.JSON(http.StatusConflict, map[string]string{"error": "Permission key already exists"})
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
			}
			p.Key = nk
		}
	}
	if body.Description != nil {
		p.Description = strings.TrimSpace(*body.Description)
	}

	if err := db.Save(&p).Error; err != nil {
		logger.Error("Failed to update permission", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update permission"})
	}

	return c.JSON(http.StatusOK, models.PermissionResponse{
		ID: p.ID, Key: p.Key, Description: p.Description,
	})
}

func deletePermission(c echo.Context) error {
	if err := requireAdmin(c); err != nil {
		return err
	}

	db := storage.GetDB()
	id := c.Param("id")
	var p models.Permission
	if err := db.First(&p, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Permission not found"})
	}

	if err := db.Exec("DELETE FROM role_permissions WHERE permission_id = ?", p.ID).Error; err != nil {
		logger.Error("Failed to clear role_permissions", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete permission"})
	}
	if err := db.Exec("DELETE FROM user_permissions WHERE permission_id = ?", p.ID).Error; err != nil {
		logger.Error("Failed to clear user_permissions", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete permission"})
	}
	if err := db.Delete(&p).Error; err != nil {
		logger.Error("Failed to delete permission", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete permission"})
	}

	return c.NoContent(http.StatusNoContent)
}
