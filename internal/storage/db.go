package storage

import (
	"context"
	"errors"
	"os"
	"spoutmc/internal/database"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/permissions"

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

	// Run migrations for User, Role, Permission, and join tables
	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.UserPlugin{},
		&models.UserPluginServer{},
	); err != nil {
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

	// Seed default permission definitions only on empty table (first install). After that, the DB is authoritative.
	keyToID := make(map[string]uint)
	var permCount int64
	db.Model(&models.Permission{}).Count(&permCount)
	if permCount == 0 {
		for _, def := range permissions.Definitions {
			p := models.Permission{Key: def.Key, Description: def.Description}
			if err := db.Create(&p).Error; err != nil {
				logger.Error("Failed to seed permission", zap.String("key", def.Key), zap.Error(err))
				continue
			}
			keyToID[def.Key] = p.ID
			logger.Info("Seeded permission", zap.String("key", def.Key))
		}
	} else {
		var all []models.Permission
		if err := db.Find(&all).Error; err == nil {
			for _, p := range all {
				keyToID[p.Key] = p.ID
			}
		}
	}

	// Insert missing permission definitions (new keys added in a Spout release).
	for _, def := range permissions.Definitions {
		var existing models.Permission
		err := db.Where("key = ?", def.Key).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p := models.Permission{Key: def.Key, Description: def.Description}
			if err := db.Create(&p).Error; err != nil {
				logger.Error("Failed to insert permission definition", zap.String("key", def.Key), zap.Error(err))
				continue
			}
			keyToID[def.Key] = p.ID
			logger.Info("Inserted new permission definition", zap.String("key", def.Key))
			// Grant new keys to admin role
			var adminRole models.Role
			if err := db.Where("name = ?", "admin").First(&adminRole).Error; err == nil {
				_ = db.Model(&adminRole).Association("Permissions").Append(&p)
			}
		}
	}
	// Refresh key map after inserts
	var allPerms []models.Permission
	if err := db.Find(&allPerms).Error; err == nil {
		for _, p := range allPerms {
			keyToID[p.Key] = p.ID
		}
	}

	// Ensure manager role has plugins.manage when that permission exists
	if pid, ok := keyToID["plugins.manage"]; ok {
		var manager models.Role
		if err := db.Where("name = ?", "manager").First(&manager).Error; err == nil {
			var count int64
			db.Table("role_permissions").Where("role_id = ? AND permission_id = ?", manager.ID, pid).Count(&count)
			if count == 0 {
				perm := models.Permission{Model: gorm.Model{ID: pid}}
				if err := db.Model(&manager).Association("Permissions").Append(&perm); err != nil {
					logger.Error("Failed to grant plugins.manage to manager", zap.Error(err))
				} else {
					logger.Info("Granted plugins.manage to manager role")
				}
			}
		}
	}

	// Attach default permissions to built-in roles only when they have none yet (avoid overwriting admin edits).
	var rolesForPermSeed []models.Role
	if db.Find(&rolesForPermSeed).Error == nil {
		for _, role := range rolesForPermSeed {
			if db.Model(&role).Association("Permissions").Count() > 0 {
				continue
			}
			if role.Name == "admin" {
				var allPerms []models.Permission
				if err := db.Find(&allPerms).Error; err != nil || len(allPerms) == 0 {
					continue
				}
				if err := db.Model(&role).Association("Permissions").Replace(allPerms); err != nil {
					logger.Error("Failed to attach permissions to role", zap.String("role", role.Name), zap.Error(err))
				}
				continue
			}
			ks, ok := permissions.RolePermissionKeys[role.Name]
			if !ok {
				continue
			}
			var perms []models.Permission
			for _, k := range ks {
				if id, ok := keyToID[k]; ok {
					perms = append(perms, models.Permission{Model: gorm.Model{ID: id}})
				}
			}
			if len(perms) == 0 {
				continue
			}
			if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
				logger.Error("Failed to attach permissions to role", zap.String("role", role.Name), zap.Error(err))
			}
		}
	}

	logger.Info("Successfully migrated database schema")
	return nil
}

func GetDB() *gorm.DB {
	return db
}
