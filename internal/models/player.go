package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Player struct {
	gorm.Model

	MinecraftUUID uuid.UUID `gorm:"uniqueIndex;not null" json:"minecraftUuid"`

	MinecraftName string `gorm:"index" json:"minecraftName"`
	AvatarDataURL string `json:"avatarDataUrl,omitempty"`

	LastLoggedInAt  *time.Time  `json:"lastLoggedInAt,omitempty"`
	LastLoggedOutAt *time.Time  `json:"lastLoggedOutAt,omitempty"`
	CurrentServer   string      `json:"currentServer,omitempty"`
	ClientBrand     string      `json:"clientBrand,omitempty"`
	ClientMods      StringSlice `gorm:"type:text" json:"clientMods,omitempty"`
}
