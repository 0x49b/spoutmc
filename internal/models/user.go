package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	MinecraftID uuid.UUID `json:"minecraft_id"`
	DisplayName string    `gorm:"uniqueIndex;not null" json:"display_name"`
	Email       string    `gorm:"uniqueIndex;not null" json:"email"`
	Password    string    `gorm:"not null" json:"password"`
	Roles       []Role    `gorm:"many2many:user:role" json:"roles"`
	Avatar      string    `json:"avatar"`
}

type UserResponse struct {
	ID          uint           `json:"id"`
	CreatedAt   time.Time      `json:"createdAt"`
	MinecraftID uuid.UUID      `json:"minecraftId"`
	DisplayName string         `json:"displayName"`
	Email       string         `json:"email"`
	Roles       []RoleResponse `json:"roles"`
	Avatar      string         `json:"avatar"`
}
