package models

import "gorm.io/gorm"

// Permission is a named capability (key format: component.module.permission).
type Permission struct {
	gorm.Model
	Key         string `gorm:"uniqueIndex;not null" json:"key"`
	Description string `gorm:"" json:"description"`
}

// PermissionResponse is returned by APIs (list/detail).
type PermissionResponse struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Description string `json:"description"`
}
