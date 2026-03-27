package access

import (
	"spoutmc/internal/models"

	"gorm.io/gorm"
)

func UserHasRole(db *gorm.DB, userID uint, roleName string) bool {
	db = resolveDB(db)
	if db == nil || roleName == "" {
		return false
	}
	var user models.User
	if err := db.Preload("Roles").First(&user, userID).Error; err != nil {
		return false
	}
	for _, r := range user.Roles {
		if r.Name == roleName {
			return true
		}
	}
	return false
}

func UserHasPermission(db *gorm.DB, userID uint, key string) bool {
	db = resolveDB(db)
	if db == nil || key == "" {
		return false
	}
	keys, err := EffectivePermissionKeys(db, userID)
	if err != nil {
		return false
	}
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func ClaimsHasRole(claims *Claims, roleName string) bool {
	if claims == nil || roleName == "" {
		return false
	}
	for _, r := range claims.Roles {
		if r == roleName {
			return true
		}
	}
	return false
}

func ClaimsHasPermission(claims *Claims, key string) bool {
	if claims == nil || key == "" {
		return false
	}
	if ClaimsHasRole(claims, AdminRoleName) {
		return true
	}
	for _, p := range claims.Permissions {
		if p == key {
			return true
		}
	}
	return false
}

const ManagerRoleName = "manager"
const PluginManagePermission = "plugins.manage"

func ClaimsCanManagePlugins(claims *Claims) bool {
	if claims == nil {
		return false
	}
	if ClaimsHasRole(claims, AdminRoleName) || ClaimsHasRole(claims, ManagerRoleName) {
		return true
	}
	return ClaimsHasPermission(claims, PluginManagePermission)
}
