package storage

import (
	"context"
	"os"
	"spoutmc/internal/database"
	"spoutmc/internal/log"
	"spoutmc/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var db *gorm.DB
var logger = log.GetLogger(log.ModuleStorage)

// InitDB connects to the SQLite database and runs GORM migrations. Call this during startup before the webserver.
func InitDB(ctx context.Context) error {
	sqlitePath := os.Getenv("SQLITE_DB_PATH")
	if sqlitePath == "" {
		sqlitePath = "data/spoutmc.db"
	}

	conn, err := database.Connect(ctx, sqlitePath)
	if err != nil {
		return err
	}

	db = conn

	// Run migrations for User, Role, and join table
	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.PlayerSupportChatMessage{}); err != nil {
		return err
	}

	// Backfill DisplayName and Slug for existing roles (migration from name-only schema)
	var allRoles []models.Role
	if db.Find(&allRoles).Error == nil {
		for _, r := range allRoles {
			updates := make(map[string]interface{})
			if r.DisplayName == "" {
				updates["display_name"] = r.Name
			}
			if r.Slug == "" {
				updates["slug"] = r.Name
			}
			if len(updates) > 0 {
				db.Model(&models.Role{}).Where("id = ?", r.ID).Updates(updates)
			}
		}
	}

	// Seed default roles if they don't exist
	defaultRoles := []struct {
		Name        string
		DisplayName string
		Slug        string
	}{
		{"admin", "Admin", "admin"},
		{"manager", "Manager", "manager"},
		{"editor", "Editor", "editor"},
		{"mod", "Mod", "mod"},
		{"support", "Support", "support"},
	}
	for _, r := range defaultRoles {
		var count int64
		if db.Model(&models.Role{}).Where("name = ?", r.Name).Count(&count); count == 0 {
			db.Create(&models.Role{Name: r.Name, DisplayName: r.DisplayName, Slug: r.Slug})
			logger.Info("Seeded default role", zap.String("role", r.Name))
		}
	}

	logger.Info("Successfully migrated database schema")
	return nil
}

func GetDB() *gorm.DB {
	return db
}
