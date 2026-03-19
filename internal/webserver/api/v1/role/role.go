package role

import (
	"net/http"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/roleutil"
	"spoutmc/internal/storage"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var logger = log.GetLogger(log.ModuleUser)

// RegisterRoleRoutes registers role-related API routes
func RegisterRoleRoutes(g *echo.Group) {
	g.GET("/role", getRoles)
	g.GET("/role/:id", getRole)
	g.POST("/role", createRole)
	g.PUT("/role/:id", updateRole)
	g.DELETE("/role/:id", deleteRole)
}

func getRoles(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database not available"})
	}

	var roles []models.Role
	if err := db.Find(&roles).Error; err != nil {
		logger.Error("Failed to fetch roles", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch roles"})
	}

	resp := make([]models.RoleResponse, len(roles))
	for i, r := range roles {
		userCount := countUsersWithRole(db, r.ID)
		resp[i] = models.RoleResponse{
			ID:          r.ID,
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Slug:        r.Slug,
			UserCount:   userCount,
		}
	}
	return c.JSON(http.StatusOK, resp)
}

func countUsersWithRole(db *gorm.DB, roleID uint) int {
	var count int64
	db.Table("user_roles").Where("role_id = ?", roleID).Count(&count)
	return int(count)
}

func getRole(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database not available"})
	}

	var role models.Role
	if err := db.First(&role, c.Param("id")).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role not found"})
	}

	return c.JSON(http.StatusOK, models.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Slug:        role.Slug,
	})
}

func createRole(c echo.Context) error {
	var req struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.Bind(&req); err != nil || req.DisplayName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "displayName is required"})
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database not available"})
	}

	name := roleutil.DisplayNameToName(req.DisplayName)
	slug := roleutil.DisplayNameToSlug(req.DisplayName)
	if name == "" || slug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid display name"})
	}

	role := models.Role{Name: name, DisplayName: req.DisplayName, Slug: slug}
	if err := db.Create(&role).Error; err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Role already exists"})
	}

	return c.JSON(http.StatusCreated, models.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Slug:        role.Slug,
	})
}

func updateRole(c echo.Context) error {
	var req struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.Bind(&req); err != nil || req.DisplayName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "displayName is required"})
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database not available"})
	}

	var role models.Role
	if err := db.First(&role, c.Param("id")).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role not found"})
	}

	name := roleutil.DisplayNameToName(req.DisplayName)
	slug := roleutil.DisplayNameToSlug(req.DisplayName)
	if name == "" || slug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid display name"})
	}

	role.Name = name
	role.DisplayName = req.DisplayName
	role.Slug = slug
	if err := db.Save(&role).Error; err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Role name or slug already exists"})
	}

	return c.JSON(http.StatusOK, models.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Slug:        role.Slug,
	})
}

func deleteRole(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database not available"})
	}

	id := c.Param("id")
	var role models.Role
	if err := db.First(&role, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role not found"})
	}

	userCount := countUsersWithRole(db, role.ID)
	if userCount > 0 {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Cannot delete role: role is assigned to one or more users",
		})
	}

	if err := db.Delete(&role).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete role"})
	}

	return c.NoContent(http.StatusNoContent)
}
