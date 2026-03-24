package notifications

import (
	"fmt"
	"time"

	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var logger = log.GetLogger(log.ModuleGit)

func UpsertOpen(key, severity, title, message, source string) error {
	if key == "" {
		return fmt.Errorf("notification key is required")
	}
	if severity == "" {
		severity = "warning"
	}
	if source == "" {
		source = "system"
	}

	db := storage.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	var existing models.SystemNotification
	err := db.Where("key = ?", key).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if err == nil {
		updates := map[string]interface{}{
			"severity":     severity,
			"title":        title,
			"message":      message,
			"source":       source,
			"is_open":      true,
			"dismissed_at": nil,
			"dismissed_by": nil,
		}
		if saveErr := db.Model(&existing).Updates(updates).Error; saveErr != nil {
			return saveErr
		}
		return nil
	}

	entry := models.SystemNotification{
		Key:      key,
		Severity: severity,
		Title:    title,
		Message:  message,
		Source:   source,
		IsOpen:   true,
	}
	if createErr := db.Create(&entry).Error; createErr != nil {
		return createErr
	}
	logger.Info("Created system notification", zap.String("key", key), zap.String("source", source))
	return nil
}

func ListOpen() ([]models.SystemNotification, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var entries []models.SystemNotification
	if err := db.Where("is_open = ?", true).Order("created_at DESC").Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

func Dismiss(id uint, userID uint) error {
	db := storage.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"is_open":      false,
		"dismissed_at": &now,
		"dismissed_by": &userID,
	}
	result := db.Model(&models.SystemNotification{}).Where("id = ? AND is_open = ?", id, true).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
