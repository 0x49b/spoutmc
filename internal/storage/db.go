package storage

import (
	"context"
	"errors"
	"os"
	"spoutmc/internal/access"
	"spoutmc/internal/database"
	"spoutmc/internal/log"
	"spoutmc/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var db *gorm.DB
var logger = log.GetLogger(log.ModuleStorage)

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
	access.SetDBProvider(GetDB)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Player{},
		&models.PlayerSupportConversation{},
		&models.PlayerSupportChatMessage{},
		&models.PlayerBan{},
		&models.PlayerKick{},
		&models.PlayerJournalEntry{},
		&models.UserPlugin{},
		&models.UserPluginServer{},
		&models.SystemNotification{},
	); err != nil {
		return err
	}

	if err := BackfillPlayerSupportConversations(db); err != nil {
		logger.Error("Failed to backfill player support conversations", zap.Error(err))
		return err
	}

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

	keyToID := make(map[string]uint)
	var permCount int64
	db.Model(&models.Permission{}).Count(&permCount)
	if permCount == 0 {
		for _, def := range access.Definitions {
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

	for _, def := range access.Definitions {
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
			var adminRole models.Role
			if err := db.Where("name = ?", "admin").First(&adminRole).Error; err == nil {
				_ = db.Model(&adminRole).Association("Permissions").Append(&p)
			}
		}
	}
	var allPerms []models.Permission
	if err := db.Find(&allPerms).Error; err == nil {
		for _, p := range allPerms {
			keyToID[p.Key] = p.ID
		}
	}

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

	if pid, ok := keyToID["player.conversations.view_all"]; ok {
		var manager models.Role
		if err := db.Where("name = ?", "manager").First(&manager).Error; err == nil {
			var count int64
			db.Table("role_permissions").Where("role_id = ? AND permission_id = ?", manager.ID, pid).Count(&count)
			if count == 0 {
				perm := models.Permission{Model: gorm.Model{ID: pid}}
				if err := db.Model(&manager).Association("Permissions").Append(&perm); err != nil {
					logger.Error("Failed to grant player.conversations.view_all to manager", zap.Error(err))
				} else {
					logger.Info("Granted player.conversations.view_all to manager role")
				}
			}
		}
	}

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
			ks, ok := access.RolePermissionKeys[role.Name]
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
