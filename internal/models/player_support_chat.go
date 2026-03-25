package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlayerSupportChatMessage is one line in a staff↔player support thread (keyed by MC player + staff user id).
type PlayerSupportChatMessage struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// MinecraftUUID is stable across gamertag changes.
	// It is optional for backward compatibility with older rows.
	McPlayerUUID *uuid.UUID `gorm:"index:idx_mc_uuid_staff_time,priority:1" json:"mcPlayerUuid,omitempty"`

	McPlayerName string `gorm:"not null;index:idx_mc_staff_time,priority:1" json:"mcPlayerName"`
	StaffUserID  uint   `gorm:"not null;index:idx_mc_staff_time,priority:2;index:idx_mc_uuid_staff_time,priority:2" json:"staffUserId"`

	ConversationID *uint                      `gorm:"index" json:"conversationId,omitempty"`
	Conversation   *PlayerSupportConversation `gorm:"foreignKey:ConversationID" json:"-"`
	Direction      string                     `gorm:"not null" json:"direction"` // outgoing | incoming
	Sender         string                     `gorm:"not null" json:"sender"`
	Role           string                     `json:"role,omitempty"`
	Message        string                     `gorm:"not null" json:"message"`
	OccurredAt     time.Time                  `gorm:"not null;index:idx_mc_staff_time,priority:3;index:idx_mc_uuid_staff_time,priority:3" json:"occurredAt"`
}
