package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerKick struct {
	gorm.Model

	MinecraftUUID uuid.UUID `gorm:"index;not null" json:"minecraftUuid"`

	Reason string `gorm:"not null" json:"reason"`

	StaffUserID uint `gorm:"index;not null" json:"staffUserId"`

	OccurredAt time.Time `gorm:"index;not null" json:"occurredAt"`
}
