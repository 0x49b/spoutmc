package models

import (
	"time"

	"gorm.io/gorm"
)

// SystemNotification is a global notification persisted for all users.
type SystemNotification struct {
	gorm.Model
	Key         string     `gorm:"uniqueIndex;not null" json:"key"`
	Severity    string     `gorm:"not null;default:warning" json:"severity"`
	Title       string     `gorm:"not null" json:"title"`
	Message     string     `gorm:"type:text" json:"message"`
	Source      string     `gorm:"not null;default:system" json:"source"`
	IsOpen      bool       `gorm:"not null;default:true;index" json:"isOpen"`
	DismissedAt *time.Time `json:"dismissedAt,omitempty"`
	DismissedBy *uint      `json:"dismissedBy,omitempty"`
}

type SystemNotificationResponse struct {
	ID          uint       `json:"id"`
	Key         string     `json:"key"`
	Severity    string     `json:"severity"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	Source      string     `json:"source"`
	IsOpen      bool       `json:"isOpen"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DismissedAt *time.Time `json:"dismissedAt,omitempty"`
}
