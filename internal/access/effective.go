package access

import (
	"sort"
	"spoutmc/internal/models"

	"gorm.io/gorm"
)

const AdminRoleName = "admin"

func EffectivePermissionKeys(db *gorm.DB, userID uint) ([]string, error) {
	db = resolveDB(db)
	if db == nil {
		return nil, nil
	}
	var user models.User
	if err := db.Preload("Roles.Permissions").Preload("DirectPermissions").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return EffectivePermissionKeysFromUserWithDB(db, &user), nil
}

func EffectivePermissionKeysFromUser(user *models.User) []string {
	return EffectivePermissionKeysFromUserWithDB(resolveDB(nil), user)
}

func EffectivePermissionKeysFromUserWithDB(db *gorm.DB, user *models.User) []string {
	if user == nil {
		return nil
	}
	for _, r := range user.Roles {
		if r.Name == AdminRoleName {
			db = resolveDB(db)
			keys, err := AllKeysFromDB(db)
			if err != nil || len(keys) == 0 {
				return nil
			}
			return AllKeysSorted(keys)
		}
	}
	set := make(map[string]struct{})
	for _, r := range user.Roles {
		for _, p := range r.Permissions {
			if p.Key != "" {
				set[p.Key] = struct{}{}
			}
		}
	}
	for _, p := range user.DirectPermissions {
		if p.Key != "" {
			set[p.Key] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
