package storage

import (
	"fmt"
	"strings"

	"spoutmc/internal/models"

	"gorm.io/gorm"
)

func normalizeMcPlayerName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func BackfillPlayerSupportConversations(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	var n int64
	if err := db.Model(&models.PlayerSupportChatMessage{}).Where("conversation_id IS NULL").Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	var rows []models.PlayerSupportChatMessage
	if err := db.Where("conversation_id IS NULL").Order("occurred_at asc, id asc").Find(&rows).Error; err != nil {
		return err
	}

	type groupKey struct {
		uuidKey string
		nameKey string
		staff   uint
	}

	buckets := make(map[groupKey][]models.PlayerSupportChatMessage)
	for _, r := range rows {
		var k groupKey
		k.staff = r.StaffUserID
		if r.McPlayerUUID != nil {
			k.uuidKey = r.McPlayerUUID.String()
		} else {
			k.nameKey = r.McPlayerName
		}
		buckets[k] = append(buckets[k], r)
	}

	for _, msgs := range buckets {
		if len(msgs) == 0 {
			continue
		}
		first := msgs[0]
		conv := models.PlayerSupportConversation{
			McPlayerName: normalizeMcPlayerName(first.McPlayerName),
			StaffUserID:  first.StaffUserID,
			ClosedAt:     nil,
		}
		if first.McPlayerUUID != nil {
			u := *first.McPlayerUUID
			conv.McPlayerUUID = &u
		}

		if err := db.Create(&conv).Error; err != nil {
			return fmt.Errorf("backfill conversation: %w", err)
		}

		ids := make([]uint, 0, len(msgs))
		for _, m := range msgs {
			ids = append(ids, m.ID)
		}
		cid := conv.ID
		if err := db.Model(&models.PlayerSupportChatMessage{}).Where("id IN ?", ids).Update("conversation_id", cid).Error; err != nil {
			return fmt.Errorf("backfill message conversation_id: %w", err)
		}
	}

	return nil
}
