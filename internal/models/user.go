package models

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID          uint         `gorm:"primarykey" json:"id"`
	MinecraftID uuid.UUID    `json:"minecraft_id"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"-"`
	DeletedAt   sql.NullTime `gorm:"index" json:"-"`
	DisplayName string       `gorm:"uniqueIndex;not null" json:"display_name"`
	Email       string       `gorm:"uniqueIndex;not null" json:"email"`
	Password    string       `gorm:"not null" json:"password"`
	Roles       []Role       `gorm:"many2many:user:role" json:"roles"`
}

type Role struct {
	ID        uint         `gorm:"primarykey" json:"id"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"-"`
	DeletedAt sql.NullTime `gorm:"index" json:"-"`
	Rolename  string       `gorm:"uniqueIndex;not null" json:"rolename"`
}
