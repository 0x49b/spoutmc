package models

import "gorm.io/gorm"

type Role struct {
	gorm.Model
	Name        string       `gorm:"uniqueIndex;not null" json:"name"` // camelCase identifier (e.g. forumModerator)
	DisplayName string       `gorm:"" json:"display_name"`             // Human readable (e.g. Forum Moderator)
	Slug        string       `gorm:"uniqueIndex" json:"slug"`          // URL-friendly (e.g. forum-moderator)
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

type RoleResponse struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	DisplayName string               `json:"displayName"`
	Slug        string               `json:"slug"`
	UserCount   int                  `json:"userCount,omitempty"` // Only set when listing roles for management
	Permissions []PermissionResponse `json:"permissions,omitempty"`
}
