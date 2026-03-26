package player

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"spoutmc/internal/minecraft"
	"spoutmc/internal/models"
	playerpkg "spoutmc/internal/player"
	"spoutmc/internal/storage"
	"spoutmc/internal/webserver/middleware"
)

type PlayerSummaryDTO struct {
	MinecraftUUID string `json:"minecraftUuid"`
	MinecraftName string `json:"minecraftName,omitempty"`
	AvatarDataURL string `json:"avatarDataUrl,omitempty"`

	Status          string     `json:"status,omitempty"`
	CurrentServer   string     `json:"currentServer,omitempty"`
	LastLoggedInAt  *time.Time `json:"lastLoggedInAt,omitempty"`
	LastLoggedOutAt *time.Time `json:"lastLoggedOutAt,omitempty"`
	ClientBrand     string     `json:"clientBrand,omitempty"`
	ClientMods      []string   `json:"clientMods,omitempty"`

	Banned     bool       `json:"banned"`
	BanReason  string     `json:"banReason,omitempty"`
	BanUntilAt *time.Time `json:"banUntilAt,omitempty"`
}

type PlayerConversationDTO struct {
	ID               uint    `json:"id"`
	StaffUserID      uint    `json:"staffUserId"`
	StaffDisplayName string  `json:"staffDisplayName"`
	LastMessage      string  `json:"lastMessage"`
	LastOccurredAt   string  `json:"lastOccurredAt"`
	Closed           bool    `json:"closed"`
	ClosedAt         *string `json:"closedAt,omitempty"`
}

type PlayerConversationListDTO struct {
	Conversations         []PlayerConversationDTO `json:"conversations"`
	HasOtherConversations bool                    `json:"hasOtherConversations"`
}

type PlayerBanHistoryDTO struct {
	StaffUserID      uint       `json:"staffUserId"`
	StaffDisplayName string     `json:"staffDisplayName"`
	Reason           string     `json:"reason"`
	CreatedAt        time.Time  `json:"createdAt"`
	UntilAt          *time.Time `json:"untilAt,omitempty"`
	LiftedAt         *time.Time `json:"liftedAt,omitempty"`
	Permanent        bool       `json:"permanent"`
}

type PlayerKickHistoryDTO struct {
	StaffUserID      uint      `json:"staffUserId"`
	StaffDisplayName string    `json:"staffDisplayName"`
	Reason           string    `json:"reason"`
	OccurredAt       time.Time `json:"occurredAt"`
}

type PlayerJournalEntryDTO struct {
	StaffUserID      uint      `json:"staffUserId"`
	StaffDisplayName string    `json:"staffDisplayName"`
	Entry            string    `json:"entry"`
	OccurredAt       time.Time `json:"occurredAt"`
}

func getPlayerUUIDAndNameOrError(c echo.Context) (uuid.UUID, string, error) {
	identifier := strings.TrimSpace(c.Param("uuid"))
	if identifier == "" {
		return uuid.Nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid minecraft uuid")
	}

	// Fast path: UUID already provided.
	if parsed, err := uuid.Parse(identifier); err == nil {
		db := storage.GetDB()
		if db != nil {
			var p models.Player
			if err2 := db.Where("minecraft_uuid = ?", parsed).First(&p).Error; err2 == nil {
				return parsed, p.MinecraftName, nil
			}
		}
		return parsed, "", nil
	}

	// Fallback for legacy clients / older bridge: resolve username -> UUID.
	playerUUID, username, skinURL, err := minecraft.GetPlayerProfile(identifier)
	if err != nil {
		return uuid.Nil, "", echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Best-effort: upsert player metadata so detail pages have a display name.
	db := storage.GetDB()
	if db != nil {
		var p models.Player
		if err2 := db.Where("minecraft_uuid = ?", playerUUID).First(&p).Error; err2 != nil {
			_ = db.Create(&models.Player{
				MinecraftUUID: playerUUID,
				MinecraftName: username,
				AvatarDataURL: skinURL,
			}).Error
		} else if strings.TrimSpace(p.MinecraftName) == "" && strings.TrimSpace(username) != "" {
			_ = db.Model(&p).Updates(map[string]any{
				"minecraft_name":  username,
				"avatar_data_url": skinURL,
			}).Error
		}
	}

	return playerUUID, username, nil
}

// getPlayerSummary returns persisted player + active ban info from SQLite.
func getPlayerSummary(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, resolvedName, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	// Avoid db.First(&x) with ErrRecordNotFound: it generates noisy "record not found" logs
	// even though this is an expected "no data yet" case.
	var pRows []models.Player
	err = db.Where("minecraft_uuid = ?", playerUUID).Limit(1).Find(&pRows).Error
	var p models.Player
	if len(pRows) > 0 {
		p = pRows[0]
	}

	// Active ban: lifted_at IS NULL.
	var activeBanRows []models.PlayerBan
	err = db.Where("minecraft_uuid = ? AND lifted_at IS NULL", playerUUID).
		Order("created_at desc").Limit(1).Find(&activeBanRows).Error

	var activeBan *models.PlayerBan
	if len(activeBanRows) > 0 {
		activeBan = &activeBanRows[0]
	}

	status := "offline"
	if strings.TrimSpace(p.CurrentServer) != "" {
		status = "online"
	}
	if activeBan != nil {
		status = "banned"
	}

	resp := PlayerSummaryDTO{
		MinecraftUUID:   playerUUID.String(),
		MinecraftName:   p.MinecraftName,
		AvatarDataURL:   p.AvatarDataURL,
		Banned:          false,
		Status:          status,
		CurrentServer:   p.CurrentServer,
		LastLoggedInAt:  p.LastLoggedInAt,
		LastLoggedOutAt: p.LastLoggedOutAt,
		ClientBrand:     p.ClientBrand,
		ClientMods:      []string(p.ClientMods),
	}
	if strings.TrimSpace(resp.MinecraftName) == "" && strings.TrimSpace(resolvedName) != "" {
		resp.MinecraftName = resolvedName
	}

	if err == nil && activeBan != nil {
		resp.Banned = true
		resp.BanReason = activeBan.Reason
		resp.BanUntilAt = activeBan.UntilAt
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, resp)
}

func resolveDisplayPlayerName(db *gorm.DB, playerUUID uuid.UUID) string {
	// Prefer the first chat line with a “real” name. Outgoing lines may store the uuid string,
	// so skip values that parse as UUID.
	display := playerUUID.String()

	var candidates []models.PlayerSupportChatMessage
	// Keep it small to avoid loading huge chat histories.
	if err := db.Where("mc_player_uuid = ?", playerUUID).
		Order("occurred_at asc, id asc").Limit(50).Find(&candidates).Error; err != nil {
		return display
	}

	for _, row := range candidates {
		if strings.TrimSpace(row.McPlayerName) == "" {
			continue
		}
		if _, err := uuid.Parse(row.McPlayerName); err == nil {
			// Likely stored from uuid identifier instead of a gamertag.
			continue
		}
		return row.McPlayerName
	}

	// Fall back to Player table name (if it exists).
	var p models.Player
	if err := db.Where("minecraft_uuid = ?", playerUUID).First(&p).Error; err == nil && strings.TrimSpace(p.MinecraftName) != "" {
		return p.MinecraftName
	}

	return display
}

// listPlayerConversations returns conversation threads for the given player (from player_support_conversations).
func listPlayerConversations(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, resolvedName, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	canArchive := chatArchiveAllowed(cl)

	displayPlayer := resolveDisplayPlayerName(db, playerUUID)
	if strings.TrimSpace(displayPlayer) == playerUUID.String() && strings.TrimSpace(resolvedName) != "" {
		displayPlayer = resolvedName
	}

	legacyName := ""
	if dp := strings.TrimSpace(displayPlayer); dp != "" && dp != playerUUID.String() {
		legacyName = playerpkg.NormalizeMcPlayerName(dp)
	} else if strings.TrimSpace(resolvedName) != "" {
		legacyName = playerpkg.NormalizeMcPlayerName(resolvedName)
	}

	entries, err := playerpkg.ListConversationsForPlayer(playerUUID, legacyName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var visible []playerpkg.ConversationListEntry
	hasOtherConversations := false
	if canArchive {
		visible = entries
	} else {
		for _, e := range entries {
			if e.StaffUserID == cl.UserID {
				visible = append(visible, e)
			} else {
				hasOtherConversations = true
			}
		}
	}

	staffIDs := make([]uint, 0, len(visible))
	for _, e := range visible {
		staffIDs = append(staffIDs, e.StaffUserID)
	}

	staffUsers := make([]models.User, 0, len(staffIDs))
	if len(staffIDs) > 0 {
		if err := db.Where("id IN ?", staffIDs).Find(&staffUsers).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	staffByID := make(map[uint]models.User, len(staffUsers))
	for _, u := range staffUsers {
		staffByID[u.ID] = u
	}

	conversations := make([]PlayerConversationDTO, 0, len(visible))
	for _, e := range visible {
		staffLabel := "Staff"
		if u, ok := staffByID[e.StaffUserID]; ok && u.ID != 0 {
			staffLabel = playerpkg.StaffChatSenderLabel(u)
		}
		var closedAt *string
		if e.ClosedAt != nil {
			s := e.ClosedAt.UTC().Format(time.RFC3339Nano)
			closedAt = &s
		}
		lastAt := ""
		if !e.LastOccurredAt.IsZero() {
			lastAt = e.LastOccurredAt.UTC().Format(time.RFC3339Nano)
		}
		conversations = append(conversations, PlayerConversationDTO{
			ID:               e.ID,
			StaffUserID:      e.StaffUserID,
			StaffDisplayName: staffLabel,
			LastMessage:      e.LastMessage,
			LastOccurredAt:   lastAt,
			Closed:           e.ClosedAt != nil,
			ClosedAt:         closedAt,
		})
	}

	return c.JSON(http.StatusOK, PlayerConversationListDTO{
		Conversations:         conversations,
		HasOtherConversations: hasOtherConversations,
	})
}

// getConversationMessages returns ordered chat lines for one conversation.
func getConversationMessages(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, resolvedName, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	conversationID64, err := strconv.ParseUint(c.Param("conversationId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid conversationId")
	}
	conversationID := uint(conversationID64)

	conv, err := playerpkg.GetConversationByID(conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "conversation not found")
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	displayPlayer := resolveDisplayPlayerName(db, playerUUID)
	if strings.TrimSpace(displayPlayer) == playerUUID.String() && strings.TrimSpace(resolvedName) != "" {
		displayPlayer = resolvedName
	}

	legacyName := ""
	if dp := strings.TrimSpace(displayPlayer); dp != "" && dp != playerUUID.String() {
		legacyName = playerpkg.NormalizeMcPlayerName(dp)
	} else if strings.TrimSpace(resolvedName) != "" {
		legacyName = playerpkg.NormalizeMcPlayerName(resolvedName)
	}

	if !playerpkg.ConversationBelongsToPlayer(conv, playerUUID, legacyName) {
		return echo.NewHTTPError(http.StatusNotFound, "conversation not found")
	}

	canArchive := chatArchiveAllowed(cl)
	if !canArchive && conv.StaffUserID != cl.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	messages, err := playerpkg.ListSupportChatMessagesForConversation(conversationID, displayPlayer)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, messages)
}

func closePlayerConversation(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	playerUUID, resolvedName, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	conversationID64, err := strconv.ParseUint(c.Param("conversationId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid conversationId")
	}
	conversationID := uint(conversationID64)

	conv, err := playerpkg.GetConversationByID(conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "conversation not found")
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db := storage.GetDB()
	displayPlayer := resolveDisplayPlayerName(db, playerUUID)
	if strings.TrimSpace(displayPlayer) == playerUUID.String() && strings.TrimSpace(resolvedName) != "" {
		displayPlayer = resolvedName
	}

	legacyName := ""
	if dp := strings.TrimSpace(displayPlayer); dp != "" && dp != playerUUID.String() {
		legacyName = playerpkg.NormalizeMcPlayerName(dp)
	} else if strings.TrimSpace(resolvedName) != "" {
		legacyName = playerpkg.NormalizeMcPlayerName(resolvedName)
	}

	if !playerpkg.ConversationBelongsToPlayer(conv, playerUUID, legacyName) {
		return echo.NewHTTPError(http.StatusNotFound, "conversation not found")
	}

	canArchive := chatArchiveAllowed(cl)
	if err := playerpkg.CloseConversationForActor(conversationID, cl.UserID, canArchive); err != nil {
		if errors.Is(err, playerpkg.ErrCloseConversationForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "closed"})
}

func listPlayerBanHistory(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, _, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	var bans []models.PlayerBan
	if err := db.Where("minecraft_uuid = ?", playerUUID).Order("created_at desc").Find(&bans).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	staffIDs := make(map[uint]struct{}, len(bans))
	for _, b := range bans {
		staffIDs[b.StaffUserID] = struct{}{}
	}
	ids := make([]uint, 0, len(staffIDs))
	for id := range staffIDs {
		ids = append(ids, id)
	}

	staffUsers := make([]models.User, 0, len(ids))
	if len(ids) > 0 {
		if err := db.Where("id IN ?", ids).Find(&staffUsers).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	staffByID := make(map[uint]models.User, len(staffUsers))
	for _, u := range staffUsers {
		staffByID[u.ID] = u
	}

	out := make([]PlayerBanHistoryDTO, 0, len(bans))
	for _, b := range bans {
		staffLabel := "Staff"
		if u, ok := staffByID[b.StaffUserID]; ok && u.ID != 0 {
			staffLabel = playerpkg.StaffChatSenderLabel(u)
		}
		out = append(out, PlayerBanHistoryDTO{
			StaffUserID:      b.StaffUserID,
			StaffDisplayName: staffLabel,
			Reason:           b.Reason,
			CreatedAt:        b.CreatedAt.UTC(),
			UntilAt:          b.UntilAt,
			LiftedAt:         b.LiftedAt,
			Permanent:        b.UntilAt == nil,
		})
	}

	return c.JSON(http.StatusOK, out)
}

func listPlayerKickHistory(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, _, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	var kicks []models.PlayerKick
	if err := db.Where("minecraft_uuid = ?", playerUUID).Order("occurred_at desc").Find(&kicks).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	staffIDs := make(map[uint]struct{}, len(kicks))
	for _, k := range kicks {
		staffIDs[k.StaffUserID] = struct{}{}
	}
	ids := make([]uint, 0, len(staffIDs))
	for id := range staffIDs {
		ids = append(ids, id)
	}

	staffUsers := make([]models.User, 0, len(ids))
	if len(ids) > 0 {
		if err := db.Where("id IN ?", ids).Find(&staffUsers).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	staffByID := make(map[uint]models.User, len(staffUsers))
	for _, u := range staffUsers {
		staffByID[u.ID] = u
	}

	out := make([]PlayerKickHistoryDTO, 0, len(kicks))
	for _, k := range kicks {
		staffLabel := "Staff"
		if u, ok := staffByID[k.StaffUserID]; ok && u.ID != 0 {
			staffLabel = playerpkg.StaffChatSenderLabel(u)
		}
		out = append(out, PlayerKickHistoryDTO{
			StaffUserID:      k.StaffUserID,
			StaffDisplayName: staffLabel,
			Reason:           k.Reason,
			OccurredAt:       k.OccurredAt.UTC(),
		})
	}

	return c.JSON(http.StatusOK, out)
}

func listPlayerJournalEntries(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, _, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	var entries []models.PlayerJournalEntry
	if err := db.Where("minecraft_uuid = ?", playerUUID).Order("occurred_at desc").Find(&entries).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	staffIDs := make(map[uint]struct{}, len(entries))
	for _, e := range entries {
		staffIDs[e.StaffUserID] = struct{}{}
	}
	ids := make([]uint, 0, len(staffIDs))
	for id := range staffIDs {
		ids = append(ids, id)
	}

	staffUsers := make([]models.User, 0, len(ids))
	if len(ids) > 0 {
		if err := db.Where("id IN ?", ids).Find(&staffUsers).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	staffByID := make(map[uint]models.User, len(staffUsers))
	for _, u := range staffUsers {
		staffByID[u.ID] = u
	}

	out := make([]PlayerJournalEntryDTO, 0, len(entries))
	for _, e := range entries {
		staffLabel := "Staff"
		if u, ok := staffByID[e.StaffUserID]; ok && u.ID != 0 {
			staffLabel = playerpkg.StaffChatSenderLabel(u)
		}
		out = append(out, PlayerJournalEntryDTO{
			StaffUserID:      e.StaffUserID,
			StaffDisplayName: staffLabel,
			Entry:            e.Entry,
			OccurredAt:       e.OccurredAt.UTC(),
		})
	}

	return c.JSON(http.StatusOK, out)
}

func addPlayerJournalEntry(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, _, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	type addJournalBody struct {
		Entry string `json:"entry"`
	}
	var body addJournalBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	entry := strings.TrimSpace(body.Entry)
	if entry == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "entry is required"})
	}

	occurredAt := time.Now().UTC()
	row := models.PlayerJournalEntry{
		MinecraftUUID: playerUUID,
		StaffUserID:   cl.UserID,
		Entry:         entry,
		OccurredAt:    occurredAt,
	}
	if err := db.Create(&row).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to persist journal entry"})
	}

	staffLabel := "Staff"
	if user, err := playerpkg.LoadUserWithRoles(cl.UserID); err == nil {
		staffLabel = playerpkg.StaffChatSenderLabel(user)
	}

	return c.JSON(http.StatusCreated, PlayerJournalEntryDTO{
		StaffUserID:      cl.UserID,
		StaffDisplayName: staffLabel,
		Entry:            entry,
		OccurredAt:       occurredAt,
	})
}

func listPlayerAliases(c echo.Context) error {
	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, resolvedName, err := getPlayerUUIDAndNameOrError(c)
	if err != nil {
		return err
	}

	// Prefer UUID-keyed rows.
	var names []string
	if err := db.Model(&models.PlayerSupportChatMessage{}).
		Where("mc_player_uuid = ?", playerUUID).
		Distinct().
		Pluck("mc_player_name", &names).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Legacy fallback: if UUID-keyed chat rows don't exist, use mc_player_name.
	if len(names) == 0 && strings.TrimSpace(resolvedName) != "" {
		_ = db.Model(&models.PlayerSupportChatMessage{}).
			Where("mc_player_name = ?", playerpkg.NormalizeMcPlayerName(resolvedName)).
			Distinct().
			Pluck("mc_player_name", &names).Error
	}

	filtered := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		// Skip values that are actually UUIDs (defensive against legacy identifier storage).
		if _, err := uuid.Parse(n); err == nil {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		filtered = append(filtered, n)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"aliases": filtered,
	})
}
