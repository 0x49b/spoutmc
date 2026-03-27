package models

import (
	"gorm.io/gorm"
)

type UserPlugin struct {
	gorm.Model
	Name        string             `gorm:"not null" json:"name"`
	URL         string             `gorm:"not null" json:"url"`
	Description string             `json:"description,omitempty"`
	Servers     []UserPluginServer `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type UserPluginServer struct {
	gorm.Model
	UserPluginID uint       `gorm:"not null;uniqueIndex:idx_user_plugin_server" json:"userPluginId"`
	ServerName   string     `gorm:"not null;uniqueIndex:idx_user_plugin_server" json:"serverName"`
	UserPlugin   UserPlugin `gorm:"foreignKey:UserPluginID" json:"-"`
}
