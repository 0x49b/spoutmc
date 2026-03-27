package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerBan struct {
	gorm.Model

	MinecraftUUID uuid.UUID `gorm:"index;not null" json:"minecraftUuid"`

	Reason string `gorm:"not null" json:"reason"`

	UntilAt *time.Time `gorm:"index" json:"untilAt,omitempty"`

	LiftedAt *time.Time `json:"liftedAt,omitempty"`

	StaffUserID uint `gorm:"index;not null" json:"staffUserId"`
}
