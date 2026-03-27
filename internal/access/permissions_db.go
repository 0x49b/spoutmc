package access

import (
	"sort"
	"spoutmc/internal/models"

	"gorm.io/gorm"
)

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

func AllKeysSorted(keys []string) []string {
	out := append([]string(nil), keys...)
	sort.Strings(out)
	return out
}
