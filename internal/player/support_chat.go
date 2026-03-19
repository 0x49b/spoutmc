package player

import (
	"fmt"
	"strings"
	"time"

	"spoutmc/internal/models"
	"spoutmc/internal/storage"
)

// NormalizeMcPlayerName lowercases trimmed MC name for consistent DB keys.
func NormalizeMcPlayerName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

var roleRank = map[string]int{
	"admin": 5, "manager": 4, "editor": 3, "mod": 2, "support": 1,
}

// StaffChatSenderLabel prefers Minecraft ign, then display name.
func StaffChatSenderLabel(u models.User) string {
	if s := strings.TrimSpace(u.MinecraftName); s != "" {
		return s
	}
	return strings.TrimSpace(u.DisplayName)
}

// PrimaryRoleDisplay picks the highest-privilege role label for chat prefixes.
func PrimaryRoleDisplay(roles []models.Role) string {
	bestScore := -1
	var label string
	for _, r := range roles {
		score := roleRank[strings.ToLower(strings.TrimSpace(r.Name))]
		if score > bestScore {
			bestScore = score
			if strings.TrimSpace(r.DisplayName) != "" {
				label = strings.TrimSpace(r.DisplayName)
			} else {
				label = strings.TrimSpace(r.Name)
			}
		}
	}
	if label == "" {
		return "Staff"
	}
	return label
}

// AppendSupportChatMessage stores one chat line.
func AppendSupportChatMessage(mcPlayer string, staffID uint, direction, sender, role, message string, at time.Time) error {
	db := storage.GetDB()
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	row := models.PlayerSupportChatMessage{
		McPlayerName: NormalizeMcPlayerName(mcPlayer),
		StaffUserID:  staffID,
		Direction:    direction,
		Sender:       sender,
		Role:         role,
		Message:      message,
		OccurredAt:   at.UTC(),
	}
	return db.Create(&row).Error
}

func supportRowToDTO(r models.PlayerSupportChatMessage, displayPlayer string) PlayerChatMessage {
	return PlayerChatMessage{
		Direction:   r.Direction,
		Player:      displayPlayer,
		StaffUserID: r.StaffUserID,
		Sender:      r.Sender,
		Role:        r.Role,
		Message:     r.Message,
		Timestamp:   r.OccurredAt.UTC().Format(time.RFC3339Nano),
	}
}

// ListSupportChatForStaff returns messages between one MC player and one staff user (panel default).
func ListSupportChatForStaff(displayPlayer string, staffID uint) ([]PlayerChatMessage, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	key := NormalizeMcPlayerName(displayPlayer)
	var rows []models.PlayerSupportChatMessage
	if err := db.Where("mc_player_name = ? AND staff_user_id = ?", key, staffID).
		Order("occurred_at asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PlayerChatMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, supportRowToDTO(row, displayPlayer))
	}
	return out, nil
}

// ListSupportChatAllForPlayer returns every stored thread line for an MC player (admin archive).
func ListSupportChatAllForPlayer(displayPlayer string) ([]PlayerChatMessage, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	key := NormalizeMcPlayerName(displayPlayer)
	var rows []models.PlayerSupportChatMessage
	if err := db.Where("mc_player_name = ?", key).
		Order("occurred_at asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PlayerChatMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, supportRowToDTO(row, displayPlayer))
	}
	return out, nil
}

// LoadUserWithRoles returns the user or gorm.ErrRecordNotFound.
func LoadUserWithRoles(userID uint) (models.User, error) {
	db := storage.GetDB()
	if db == nil {
		return models.User{}, fmt.Errorf("database unavailable")
	}
	var user models.User
	err := db.Preload("Roles").First(&user, userID).Error
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
