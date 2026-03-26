package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Player is the persisted representation of a Minecraft user.
// Identity is the canonical Minecraft UUID (stable across gamertag changes).
type Player struct {
	gorm.Model

	MinecraftUUID uuid.UUID `gorm:"uniqueIndex;not null" json:"minecraftUuid"`

	// Display fields (best-effort; can change when the player changes their gamertag).
	MinecraftName string `gorm:"index" json:"minecraftName"`
	AvatarDataURL string `json:"avatarDataUrl,omitempty"`

	LastLoggedInAt  *time.Time  `json:"lastLoggedInAt,omitempty"`
	LastLoggedOutAt *time.Time  `json:"lastLoggedOutAt,omitempty"`
	CurrentServer   string      `json:"currentServer,omitempty"`
	ClientBrand     string      `json:"clientBrand,omitempty"`
	ClientMods      StringSlice `gorm:"type:text" json:"clientMods,omitempty"`
}
