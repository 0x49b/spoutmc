package models

import "gorm.io/gorm"

type Role struct {
	gorm.Model
	Rolename string `gorm:"uniqueIndex;not null" json:"rolename"`
}

type RoleResponse struct {
	Rolename string `json:"rolename"`
}
