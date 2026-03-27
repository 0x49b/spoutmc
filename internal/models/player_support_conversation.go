package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerSupportConversation struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	McPlayerUUID *uuid.UUID `gorm:"index:idx_conv_mc_uuid_staff,priority:1" json:"mcPlayerUuid,omitempty"`
	McPlayerName string     `gorm:"not null;index:idx_conv_mc_name_staff,priority:1" json:"mcPlayerName"`
	StaffUserID  uint       `gorm:"not null;index:idx_conv_mc_uuid_staff,priority:2;index:idx_conv_mc_name_staff,priority:2" json:"staffUserId"`
	ClosedAt     *time.Time `json:"closedAt,omitempty"`
}

func (PlayerSupportConversation) TableName() string {
	return "player_support_conversations"
}
