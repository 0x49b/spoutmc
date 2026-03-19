package models

import (
	"time"

	"gorm.io/gorm"
)

// PlayerSupportChatMessage is one line in a staff↔player support thread (keyed by MC player + staff user id).
type PlayerSupportChatMessage struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	McPlayerName string         `gorm:"not null;index:idx_mc_staff_time,priority:1" json:"mcPlayerName"`
	StaffUserID  uint           `gorm:"not null;index:idx_mc_staff_time,priority:2" json:"staffUserId"`
	Direction    string         `gorm:"not null" json:"direction"` // outgoing | incoming
	Sender       string         `gorm:"not null" json:"sender"`
	Role         string         `json:"role,omitempty"`
	Message      string         `gorm:"not null" json:"message"`
	OccurredAt   time.Time      `gorm:"not null;index:idx_mc_staff_time,priority:3" json:"occurredAt"`
}
