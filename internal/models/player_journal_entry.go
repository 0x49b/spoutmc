package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerJournalEntry struct {
	gorm.Model

	MinecraftUUID uuid.UUID `gorm:"index;not null" json:"minecraftUuid"`
	StaffUserID   uint      `gorm:"index;not null" json:"staffUserId"`
	Entry         string    `gorm:"not null" json:"entry"`
	OccurredAt    time.Time `gorm:"index;not null" json:"occurredAt"`
}
