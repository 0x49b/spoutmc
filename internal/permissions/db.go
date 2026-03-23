package permissions

import (
	"sort"
	"spoutmc/internal/models"

	"gorm.io/gorm"
)

// AllKeysFromDB returns every permission key currently stored in the database (sorted).
// This is the runtime source of truth for “all permissions” (e.g. admin effective set).
func AllKeysFromDB(db *gorm.DB) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	var perms []models.Permission
	if err := db.Order("key").Find(&perms).Error; err != nil {
		return nil, err
	}
	keys := make([]string, len(perms))
	for i, p := range perms {
		keys[i] = p.Key
	}
	return keys, nil
}

// AllKeysSorted returns sorted keys (copy).
func AllKeysSorted(keys []string) []string {
	out := append([]string(nil), keys...)
	sort.Strings(out)
	return out
}
