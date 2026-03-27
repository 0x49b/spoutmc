package models

import "gorm.io/gorm"

type Permission struct {
	gorm.Model
	Key         string `gorm:"uniqueIndex;not null" json:"key"`
	Description string `gorm:"" json:"description"`
}

type PermissionResponse struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Description string `json:"description"`
}
