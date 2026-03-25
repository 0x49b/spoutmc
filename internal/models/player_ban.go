package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlayerBan records ban/kick lifecycle. Timed bans are defined by UntilAt.
// Permanent bans have UntilAt = NULL.
type PlayerBan struct {
	gorm.Model

	MinecraftUUID uuid.UUID `gorm:"index;not null" json:"minecraftUuid"`

	Reason string `gorm:"not null" json:"reason"`

	// UntilAt is the time when the ban expires. When nil, the ban is permanent.
	UntilAt *time.Time `gorm:"index" json:"untilAt,omitempty"`

	// LiftedAt is set when the ban is removed (via unban or cron expiry).
	LiftedAt *time.Time `json:"liftedAt,omitempty"`

	// StaffUserID indicates who created the ban.
	StaffUserID uint `gorm:"index;not null" json:"staffUserId"`
}
