package player

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"

	"gorm.io/gorm"
)

func NormalizeMcPlayerName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

var roleRank = map[string]int{
	"admin": 5, "manager": 4, "editor": 3, "mod": 2, "support": 1,
}

var ErrCloseConversationForbidden = errors.New("forbidden")

func StaffChatSenderLabel(u models.User) string {
	if s := strings.TrimSpace(u.MinecraftName); s != "" {
		return s
	}
	return strings.TrimSpace(u.DisplayName)
}

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

func ensurePlayerRowExists(db *gorm.DB, mcPlayer string, mcPlayerUUID *uuid.UUID) error {
	if mcPlayerUUID == nil {
		return nil
	}
	var p models.Player
	err := db.Where("minecraft_uuid = ?", *mcPlayerUUID).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p = models.Player{
				MinecraftUUID: *mcPlayerUUID,
				MinecraftName: mcPlayer,
			}
			if err := db.Create(&p).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	} else if strings.TrimSpace(p.MinecraftName) == "" && strings.TrimSpace(mcPlayer) != "" {
		if err := db.Model(&p).Update("minecraft_name", mcPlayer).Error; err != nil {
			return err
		}
	}
	return nil
}

func conversationMatchesPlayer(conv *models.PlayerSupportConversation, mcPlayerUUID *uuid.UUID, mcPlayerName string) bool {
	if conv.McPlayerUUID != nil && mcPlayerUUID != nil {
		return *conv.McPlayerUUID == *mcPlayerUUID
	}
	if conv.McPlayerUUID != nil && mcPlayerUUID == nil {
		return false
	}
	if conv.McPlayerUUID == nil {
		return NormalizeMcPlayerName(conv.McPlayerName) == NormalizeMcPlayerName(mcPlayerName)
	}
	return false
}

func AppendSupportChatMessage(conversationID uint, mcPlayer string, mcPlayerUUID *uuid.UUID, staffID uint, direction, sender, role, message string, at time.Time) error {
	db := storage.GetDB()
	if db == nil {
		return fmt.Errorf("database unavailable")
	}

	var conv models.PlayerSupportConversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		return err
	}
	if conv.StaffUserID != staffID {
		return fmt.Errorf("conversation staff mismatch")
	}
	if !conversationMatchesPlayer(&conv, mcPlayerUUID, mcPlayer) {
		return fmt.Errorf("conversation player mismatch")
	}
	if conv.ClosedAt != nil {
		return fmt.Errorf("conversation is closed")
	}

	if err := ensurePlayerRowExists(db, mcPlayer, mcPlayerUUID); err != nil {
		return err
	}

	cid := conversationID
	row := models.PlayerSupportChatMessage{
		McPlayerName:   NormalizeMcPlayerName(mcPlayer),
		McPlayerUUID:   mcPlayerUUID,
		StaffUserID:    staffID,
		ConversationID: &cid,
		Direction:      direction,
		Sender:         sender,
		Role:           role,
		Message:        message,
		OccurredAt:     at.UTC(),
	}
	if err := db.Create(&row).Error; err != nil {
		return err
	}
	_ = db.Model(&models.PlayerSupportConversation{}).Where("id = ?", conversationID).
		UpdateColumn("updated_at", time.Now().UTC()).Error
	return nil
}

func CreateSupportConversation(mcPlayerName string, mcPlayerUUID *uuid.UUID, staffID uint) (*models.PlayerSupportConversation, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	conv := models.PlayerSupportConversation{
		McPlayerName: NormalizeMcPlayerName(mcPlayerName),
		McPlayerUUID: mcPlayerUUID,
		StaffUserID:  staffID,
		ClosedAt:     nil,
	}
	if err := db.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func CloseOpenConversationsForStaffPlayer(db *gorm.DB, playerUUID uuid.UUID, staffID uint, legacyNameNormalized string) error {
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	now := time.Now().UTC()
	q := db.Model(&models.PlayerSupportConversation{}).
		Where("staff_user_id = ? AND closed_at IS NULL", staffID)
	if strings.TrimSpace(legacyNameNormalized) != "" {
		q = q.Where("(mc_player_uuid = ? OR (mc_player_uuid IS NULL AND mc_player_name = ?))", playerUUID, legacyNameNormalized)
	} else {
		q = q.Where("mc_player_uuid = ?", playerUUID)
	}
	return q.Update("closed_at", now).Error
}

func FindOpenConversationForStaffPlayer(playerUUID uuid.UUID, staffID uint) (*models.PlayerSupportConversation, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	var conv models.PlayerSupportConversation
	err := db.Where("mc_player_uuid = ? AND staff_user_id = ? AND closed_at IS NULL", playerUUID, staffID).
		Order("updated_at desc").First(&conv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func FindOpenConversationForStaffPlayerByName(mcPlayerName string, staffID uint) (*models.PlayerSupportConversation, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	key := NormalizeMcPlayerName(mcPlayerName)
	var conv models.PlayerSupportConversation
	err := db.Where("mc_player_uuid IS NULL AND mc_player_name = ? AND staff_user_id = ? AND closed_at IS NULL", key, staffID).
		Order("updated_at desc").First(&conv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func GetConversationByID(id uint) (*models.PlayerSupportConversation, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	var conv models.PlayerSupportConversation
	if err := db.First(&conv, id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func ResolveOpenConversationForIngest(mcPlayerUUID *uuid.UUID, staffID uint, mcPlayerName string) (uint, error) {
	db := storage.GetDB()
	if db == nil {
		return 0, fmt.Errorf("database unavailable")
	}
	if mcPlayerUUID != nil {
		var conv models.PlayerSupportConversation
		err := db.Where("mc_player_uuid = ? AND staff_user_id = ? AND closed_at IS NULL", *mcPlayerUUID, staffID).
			Order("updated_at desc").First(&conv).Error
		if err == nil {
			return conv.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	key := NormalizeMcPlayerName(mcPlayerName)
	var conv2 models.PlayerSupportConversation
	err := db.Where("mc_player_uuid IS NULL AND mc_player_name = ? AND staff_user_id = ? AND closed_at IS NULL", key, staffID).
		Order("updated_at desc").First(&conv2).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("no open conversation")
	}
	if err != nil {
		return 0, err
	}
	return conv2.ID, nil
}

func ValidateConversationForOutgoingMessage(conv *models.PlayerSupportConversation, playerUUID *uuid.UUID, mcPlayerName string, staffID uint) error {
	if conv.StaffUserID != staffID {
		return fmt.Errorf("not your conversation")
	}
	if !conversationMatchesPlayer(conv, playerUUID, mcPlayerName) {
		return fmt.Errorf("conversation does not match player")
	}
	return nil
}

func ConversationBelongsToPlayer(conv *models.PlayerSupportConversation, playerUUID uuid.UUID, legacyNameNormalized string) bool {
	if conv.McPlayerUUID != nil && *conv.McPlayerUUID == playerUUID {
		return true
	}
	if conv.McPlayerUUID == nil && strings.TrimSpace(legacyNameNormalized) != "" {
		return NormalizeMcPlayerName(conv.McPlayerName) == legacyNameNormalized
	}
	return false
}

func CloseConversationForActor(conversationID uint, actorUserID uint, allowViewAll bool) error {
	db := storage.GetDB()
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	var conv models.PlayerSupportConversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		return err
	}
	if conv.StaffUserID != actorUserID && !allowViewAll {
		return ErrCloseConversationForbidden
	}
	if conv.ClosedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	return db.Model(&conv).Update("closed_at", now).Error
}

type ConversationListEntry struct {
	ID             uint
	StaffUserID    uint
	ClosedAt       *time.Time
	LastMessage    string
	LastOccurredAt time.Time
}

func ListConversationsForPlayer(playerUUID uuid.UUID, legacyNameNormalized string) ([]ConversationListEntry, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	var convs []models.PlayerSupportConversation
	var err error
	if strings.TrimSpace(legacyNameNormalized) != "" {
		err = db.Where("(mc_player_uuid = ? OR (mc_player_uuid IS NULL AND mc_player_name = ?))", playerUUID, legacyNameNormalized).
			Order("updated_at desc").Find(&convs).Error
	} else {
		err = db.Where("mc_player_uuid = ?", playerUUID).Order("updated_at desc").Find(&convs).Error
	}
	if err != nil {
		return nil, err
	}

	out := make([]ConversationListEntry, 0, len(convs))
	for _, c := range convs {
		var last models.PlayerSupportChatMessage
		lastMsg := ""
		var lastAt time.Time
		if err := db.Where("conversation_id = ?", c.ID).Order("occurred_at desc, id desc").First(&last).Error; err == nil {
			lastMsg = last.Message
			lastAt = last.OccurredAt
		}
		out = append(out, ConversationListEntry{
			ID:             c.ID,
			StaffUserID:    c.StaffUserID,
			ClosedAt:       c.ClosedAt,
			LastMessage:    lastMsg,
			LastOccurredAt: lastAt,
		})
	}
	return out, nil
}

func ListSupportChatMessagesForConversation(conversationID uint, displayPlayer string) ([]PlayerChatMessage, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	var rows []models.PlayerSupportChatMessage
	if err := db.Where("conversation_id = ?", conversationID).
		Order("occurred_at asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PlayerChatMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, supportRowToDTO(row, displayPlayer))
	}
	return out, nil
}

func supportRowToDTO(r models.PlayerSupportChatMessage, displayPlayer string) PlayerChatMessage {
	m := PlayerChatMessage{
		Direction:   r.Direction,
		Player:      displayPlayer,
		StaffUserID: r.StaffUserID,
		Sender:      r.Sender,
		Role:        r.Role,
		Message:     r.Message,
		Timestamp:   r.OccurredAt.UTC().Format(time.RFC3339Nano),
	}
	if r.ConversationID != nil {
		m.ConversationID = *r.ConversationID
	}
	return m
}

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

func ListSupportChatForStaffByUUID(playerUUID uuid.UUID, displayPlayer string, staffID uint) ([]PlayerChatMessage, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	var rows []models.PlayerSupportChatMessage
	if err := db.Where("mc_player_uuid = ? AND staff_user_id = ?", playerUUID, staffID).
		Order("occurred_at asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]PlayerChatMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, supportRowToDTO(row, displayPlayer))
	}
	return out, nil
}

func ListSupportChatAllForPlayerByUUID(playerUUID uuid.UUID, displayPlayer string) ([]PlayerChatMessage, error) {
	db := storage.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	var rows []models.PlayerSupportChatMessage
	if err := db.Where("mc_player_uuid = ?", playerUUID).
		Order("occurred_at asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]PlayerChatMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, supportRowToDTO(row, displayPlayer))
	}
	return out, nil
}

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
