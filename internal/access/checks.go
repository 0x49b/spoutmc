package access

import (
	"spoutmc/internal/models"

	"gorm.io/gorm"
)

// UserHasRole returns true if the user has a role with the given name.
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

// UserHasPermission returns true if the effective permissions for the user include key.
// Admin role implies all keys from the registry.
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

// ClaimsHasRole checks JWT claims for a role name (may be stale vs DB).
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

// ClaimsHasPermission checks JWT claims for a permission key (may be stale vs DB).
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

// ClaimsCanManagePlugins returns true if the user may create, update, or delete user-defined plugins
// (admin, manager, or explicit plugins.manage permission).
func ClaimsCanManagePlugins(claims *Claims) bool {
	if claims == nil {
		return false
	}
	if ClaimsHasRole(claims, AdminRoleName) || ClaimsHasRole(claims, ManagerRoleName) {
		return true
	}
	return ClaimsHasPermission(claims, PluginManagePermission)
}
