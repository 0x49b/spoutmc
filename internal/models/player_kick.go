package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlayerKick stores when staff kicked a player and why.
type PlayerKick struct {
	gorm.Model

	MinecraftUUID uuid.UUID `gorm:"index;not null" json:"minecraftUuid"`

	Reason string `gorm:"not null" json:"reason"`

	StaffUserID uint `gorm:"index;not null" json:"staffUserId"`

	// OccurredAt is an explicit timestamp for API use; it mirrors CreatedAt.
	// (Kept as separate field so we can later decouple from gorm.Model if needed.)
	OccurredAt time.Time `gorm:"index;not null" json:"occurredAt"`
}
