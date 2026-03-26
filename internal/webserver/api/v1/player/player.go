package player

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"spoutmc/internal/access"
	"spoutmc/internal/minecraft"
	"spoutmc/internal/models"
	playerpkg "spoutmc/internal/player"
	"spoutmc/internal/sse"
	"spoutmc/internal/storage"
	"spoutmc/internal/webserver/middleware"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/teris-io/shortid"
	"gorm.io/gorm"
)

func RegisterPlayerRoutes(g *echo.Group) {
	playerGroup := g.Group("/player")

	playerGroup.GET("", listPlayers)
	playerGroup.GET("/stream", streamPlayers)
	playerGroup.GET("/ban-durations", getBanDurations)
	// Player detail / conversations (UUID-keyed).
	playerGroup.GET("/:uuid", getPlayerSummary)
	playerGroup.GET("/:uuid/bans", listPlayerBanHistory)
	playerGroup.GET("/:uuid/kicks", listPlayerKickHistory)
	playerGroup.GET("/:uuid/journal", listPlayerJournalEntries)
	playerGroup.POST("/:uuid/journal", addPlayerJournalEntry)
	playerGroup.GET("/:uuid/aliases", listPlayerAliases)
	playerGroup.GET("/:uuid/conversations", listPlayerConversations)
	playerGroup.GET("/:uuid/conversations/:conversationId/messages", getConversationMessages)
	playerGroup.POST("/:uuid/conversations/:conversationId/close", closePlayerConversation)
	playerGroup.GET("/:name/chat", getPlayerChat)
	playerGroup.POST("/:name/message", messagePlayer)
	playerGroup.POST("/:name/kick", kickPlayer)
	playerGroup.POST("/:name/ban", banPlayer)
	playerGroup.POST("/:name/unban", unbanPlayer)
}

var bridgeClient = playerpkg.NewBridgeClientFromEnv()

func listPlayers(c echo.Context) error {
	players, err := bridgeClient.ListPlayers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}

	// Persist the bridge snapshot into SQLite so the player detail page can load.
	// This is "best effort": if a field can't be parsed, we still store what we can.
	db := storage.GetDB()
	if db != nil {
		for _, p := range players {
			playerUUID, parseErr := uuid.Parse(strings.TrimSpace(p.UUID))
			if parseErr != nil {
				continue
			}

			var existing models.Player
			err := db.Where("minecraft_uuid = ?", playerUUID).First(&existing).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					_ = db.Create(&models.Player{
						MinecraftUUID: playerUUID,
						MinecraftName: p.Name,
						AvatarDataURL: p.AvatarDataURL,
						CurrentServer: p.CurrentServer,
						ClientBrand:   p.ClientBrand,
						ClientMods:    models.StringSlice(p.ClientMods),
					}).Error
				}
				continue
			}

			updates := map[string]any{
				"minecraft_name":  p.Name,
				"avatar_data_url": p.AvatarDataURL,
				"current_server":  p.CurrentServer,
				"client_brand":    p.ClientBrand,
				"client_mods":     models.StringSlice(p.ClientMods),
			}
			if p.LastLoggedInAt != nil && strings.TrimSpace(*p.LastLoggedInAt) != "" {
				if t, err2 := time.Parse(time.RFC3339Nano, strings.TrimSpace(*p.LastLoggedInAt)); err2 == nil {
					updates["last_logged_in_at"] = &t
				} else if t, err2 := time.Parse(time.RFC3339, strings.TrimSpace(*p.LastLoggedInAt)); err2 == nil {
					updates["last_logged_in_at"] = &t
				}
			}
			if p.LastLoggedOutAt != nil && strings.TrimSpace(*p.LastLoggedOutAt) != "" {
				if t, err2 := time.Parse(time.RFC3339Nano, strings.TrimSpace(*p.LastLoggedOutAt)); err2 == nil {
					updates["last_logged_out_at"] = &t
				} else if t, err2 := time.Parse(time.RFC3339, strings.TrimSpace(*p.LastLoggedOutAt)); err2 == nil {
					updates["last_logged_out_at"] = &t
				}
			}

			_ = db.Model(&existing).Updates(updates).Error
		}
	}

	return c.JSON(http.StatusOK, players)
}

func streamPlayers(c echo.Context) error {
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastPayload string

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-ticker.C:
			players, err := bridgeClient.ListPlayers(c.Request().Context())
			if err != nil {
				continue
			}

			data, err := json.Marshal(players)
			if err != nil {
				return err
			}
			payload := string(data)
			if payload == lastPayload {
				continue
			}
			lastPayload = payload

			id, _ := shortid.Generate()
			event := sse.Event{
				ID:        []byte(id),
				Data:      []byte(payload),
				Timestamp: time.Now().Unix(),
			}
			if err := event.MarshalTo(w); err != nil {
				return err
			}
			w.Flush()
		}
	}
}

type playerMessageBody struct {
	Message         string `json:"message"`
	NewConversation bool   `json:"newConversation"`
	ConversationID  *uint  `json:"conversationId,omitempty"`
}

func messagePlayer(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	var body playerMessageBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	msg := strings.TrimSpace(body.Message)
	if msg == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "message is required"})
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	user, err := playerpkg.LoadUserWithRoles(cl.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	sender := playerpkg.StaffChatSenderLabel(user)
	roleLabel := playerpkg.PrimaryRoleDisplay(user.Roles)
	playerIdentifier := c.Param("name")

	mcPlayerName := playerIdentifier
	var playerUUIDParsed *uuid.UUID

	if u, err := uuid.Parse(playerIdentifier); err == nil {
		playerUUIDParsed = &u
		if playerUUID, username, skinURL, err2 := minecraft.GetPlayerProfile(playerIdentifier); err2 == nil {
			_ = skinURL
			mcPlayerName = username
			var p models.Player
			if err3 := db.Where("minecraft_uuid = ?", playerUUID).First(&p).Error; err3 != nil {
				if errors.Is(err3, gorm.ErrRecordNotFound) {
					_ = db.Create(&models.Player{
						MinecraftUUID: playerUUID,
						MinecraftName: username,
						AvatarDataURL: skinURL,
					}).Error
				}
			} else if strings.TrimSpace(p.MinecraftName) == "" && strings.TrimSpace(username) != "" {
				_ = db.Model(&p).Updates(map[string]any{
					"minecraft_name":  username,
					"avatar_data_url": skinURL,
				}).Error
			}
		}
	} else {
		resolvedUUID, name, skinURL, err2 := minecraft.GetPlayerProfile(playerIdentifier)
		if err2 != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err2.Error()})
		}
		playerUUIDParsed = &resolvedUUID
		mcPlayerName = name
		if strings.TrimSpace(name) != "" {
			var p models.Player
			if err3 := db.Where("minecraft_uuid = ?", resolvedUUID).First(&p).Error; err3 != nil {
				if errors.Is(err3, gorm.ErrRecordNotFound) {
					_ = db.Create(&models.Player{
						MinecraftUUID: resolvedUUID,
						MinecraftName: name,
						AvatarDataURL: skinURL,
					}).Error
				}
			} else if strings.TrimSpace(p.MinecraftName) == "" {
				_ = db.Model(&p).Updates(map[string]any{
					"minecraft_name":  name,
					"avatar_data_url": skinURL,
				}).Error
			}
		}
	}

	legacyName := playerpkg.NormalizeMcPlayerName(mcPlayerName)

	var conv *models.PlayerSupportConversation
	if body.NewConversation {
		if playerUUIDParsed == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "could not resolve player UUID"})
		}
		if err := playerpkg.CloseOpenConversationsForStaffPlayer(db, *playerUUIDParsed, cl.UserID, legacyName); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		created, err := playerpkg.CreateSupportConversation(mcPlayerName, playerUUIDParsed, cl.UserID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		conv = created
	} else if body.ConversationID != nil {
		c0, err := playerpkg.GetConversationByID(*body.ConversationID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "conversation not found"})
		}
		if err := playerpkg.ValidateConversationForOutgoingMessage(c0, playerUUIDParsed, mcPlayerName, cl.UserID); err != nil {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		if c0.ClosedAt != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "conversation is closed"})
		}
		conv = c0
	} else {
		if playerUUIDParsed != nil {
			open, err := playerpkg.FindOpenConversationForStaffPlayer(*playerUUIDParsed, cl.UserID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			if open != nil {
				conv = open
			}
		}
		if conv == nil {
			openName, err := playerpkg.FindOpenConversationForStaffPlayerByName(mcPlayerName, cl.UserID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			if openName != nil {
				conv = openName
			}
		}
		if conv == nil {
			created, err := playerpkg.CreateSupportConversation(mcPlayerName, playerUUIDParsed, cl.UserID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			conv = created
		}
	}

	if err := bridgeClient.MessagePlayerWithMeta(c.Request().Context(), playerIdentifier, msg, sender, roleLabel, cl.UserID, body.NewConversation); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := playerpkg.AppendSupportChatMessage(conv.ID, mcPlayerName, playerUUIDParsed, cl.UserID, "outgoing", sender, roleLabel, msg, time.Now().UTC()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to persist chat"})
	}

	return c.JSON(http.StatusAccepted, map[string]any{
		"status":         "message sent",
		"conversationId": conv.ID,
	})
}

func getPlayerChat(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	if storage.GetDB() == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerName := c.Param("name")
	scope := strings.ToLower(strings.TrimSpace(c.QueryParam("scope")))

	var messages []playerpkg.PlayerChatMessage
	var err error
	if scope == "all" {
		if !chatArchiveAllowed(cl) {
			return echo.NewHTTPError(http.StatusForbidden, "archive scope requires admin or manager")
		}
		messages, err = playerpkg.ListSupportChatAllForPlayer(playerName)
	} else {
		messages, err = playerpkg.ListSupportChatForStaff(playerName, cl.UserID)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, messages)
}

func chatArchiveAllowed(cl *access.Claims) bool {
	return access.ClaimsHasPermission(cl, "player.conversations.view_all")
}

func kickPlayer(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	var cmd playerpkg.PlayerCommand
	if err := c.Bind(&cmd); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	playerIdentifier := c.Param("name")

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, err := uuid.Parse(playerIdentifier)
	var mcPlayerName string
	var skinURL string
	if err != nil {
		// Fallback for legacy clients passing a username instead of a UUID.
		var resolvedUUID uuid.UUID
		resolvedUUID, mcPlayerName, skinURL, err = minecraft.GetPlayerProfile(playerIdentifier)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		playerUUID = resolvedUUID
	} else {
		// Best-effort: also store the player name/avatar for UUID interactions.
		if _, username, resolvedSkinURL, err2 := minecraft.GetPlayerProfile(playerIdentifier); err2 == nil {
			mcPlayerName = username
			skinURL = resolvedSkinURL
		}
	}

	now := time.Now().UTC()

	// Upsert player metadata so the detail page can resolve names.
	if mcPlayerName != "" {
		var p models.Player
		if err3 := db.Where("minecraft_uuid = ?", playerUUID).First(&p).Error; err3 != nil {
			if errors.Is(err3, gorm.ErrRecordNotFound) {
				_ = db.Create(&models.Player{
					MinecraftUUID: playerUUID,
					MinecraftName: mcPlayerName,
					AvatarDataURL: skinURL,
				}).Error
			}
		} else {
			if strings.TrimSpace(p.MinecraftName) == "" {
				_ = db.Model(&p).Updates(map[string]any{
					"minecraft_name":  mcPlayerName,
					"avatar_data_url": skinURL,
				}).Error
			}
		}
	}

	// Record the kick for staff audit/history.
	if err := db.Create(&models.PlayerKick{
		MinecraftUUID: playerUUID,
		Reason:        cmd.Reason,
		StaffUserID:   cl.UserID,
		OccurredAt:    now,
	}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to persist kick"})
	}

	if err := bridgeClient.KickPlayer(c.Request().Context(), playerIdentifier, cmd.Reason); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "player kicked"})
}

func banPlayer(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	type banPlayerBody struct {
		Reason    string  `json:"reason"`
		UntilAt   *string `json:"untilAt,omitempty"`
		Permanent *bool   `json:"permanent,omitempty"`
	}

	var body banPlayerBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	playerIdentifier := c.Param("name")

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, err := uuid.Parse(playerIdentifier)
	var mcPlayerName string
	var skinURL string
	if err != nil {
		// Fallback for legacy clients passing a username instead of a UUID.
		var resolvedUUID uuid.UUID
		resolvedUUID, mcPlayerName, skinURL, err = minecraft.GetPlayerProfile(playerIdentifier)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		playerUUID = resolvedUUID
	} else {
		// Best-effort: also store the player name/avatar for UUID interactions.
		if _, username, resolvedSkinURL, err2 := minecraft.GetPlayerProfile(playerIdentifier); err2 == nil {
			mcPlayerName = username
			skinURL = resolvedSkinURL
		}
	}

	now := time.Now().UTC()

	var untilAt *time.Time
	if body.Permanent != nil && *body.Permanent {
		untilAt = nil
	} else if body.UntilAt != nil && strings.TrimSpace(*body.UntilAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.UntilAt))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid untilAt, expected RFC3339"})
		}
		untilAt = &t
	} else {
		// Default: permanent ban (matches current UI which only sends a reason).
		untilAt = nil
	}

	// Replace: lift any currently-active bans before creating a new one.
	if err := db.Model(&models.PlayerBan{}).
		Where("minecraft_uuid = ? AND lifted_at IS NULL", playerUUID).
		Update("lifted_at", now).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update existing ban state"})
	}

	// Upsert player metadata so the detail page has something even if no chat exists.
	if mcPlayerName != "" {
		var p models.Player
		if err3 := db.Where("minecraft_uuid = ?", playerUUID).First(&p).Error; err3 != nil {
			if errors.Is(err3, gorm.ErrRecordNotFound) {
				_ = db.Create(&models.Player{
					MinecraftUUID: playerUUID,
					MinecraftName: mcPlayerName,
					AvatarDataURL: skinURL,
				}).Error
			}
		} else {
			if strings.TrimSpace(p.MinecraftName) == "" {
				_ = db.Model(&p).Updates(map[string]any{
					"minecraft_name":  mcPlayerName,
					"avatar_data_url": skinURL,
				}).Error
			}
		}
	}

	if err := db.Create(&models.PlayerBan{
		MinecraftUUID: playerUUID,
		Reason:        body.Reason,
		UntilAt:       untilAt,
		StaffUserID:   cl.UserID,
		// LiftedAt defaults to NULL
	}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to persist ban"})
	}

	if err := bridgeClient.BanPlayer(c.Request().Context(), playerIdentifier, body.Reason); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "player banned"})
}

func unbanPlayer(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	playerIdentifier := c.Param("name")

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
	}

	playerUUID, err := uuid.Parse(playerIdentifier)
	if err != nil {
		// Fallback for legacy clients passing a username instead of a UUID.
		var mcPlayerName string
		var skinURL string
		playerUUID, mcPlayerName, skinURL, err = minecraft.GetPlayerProfile(playerIdentifier)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Upsert player metadata if we resolved a name.
		if mcPlayerName != "" {
			var p models.Player
			if err3 := db.Where("minecraft_uuid = ?", playerUUID).First(&p).Error; err3 != nil {
				if errors.Is(err3, gorm.ErrRecordNotFound) {
					_ = db.Create(&models.Player{
						MinecraftUUID: playerUUID,
						MinecraftName: mcPlayerName,
						AvatarDataURL: skinURL,
					}).Error
				}
			} else if strings.TrimSpace(p.MinecraftName) == "" {
				_ = db.Model(&p).Updates(map[string]any{
					"minecraft_name":  mcPlayerName,
					"avatar_data_url": skinURL,
				}).Error
			}
		}
	} else {
		// If UUID is provided, best-effort enrich.
		if _, username, skinURL, err2 := minecraft.GetPlayerProfile(playerIdentifier); err2 == nil {
			if strings.TrimSpace(username) != "" {
				var p models.Player
				if err3 := db.Where("minecraft_uuid = ?", playerUUID).First(&p).Error; err3 != nil {
					if errors.Is(err3, gorm.ErrRecordNotFound) {
						_ = db.Create(&models.Player{
							MinecraftUUID: playerUUID,
							MinecraftName: username,
							AvatarDataURL: skinURL,
						}).Error
					}
				} else if strings.TrimSpace(p.MinecraftName) == "" {
					_ = db.Model(&p).Updates(map[string]any{
						"minecraft_name":  username,
						"avatar_data_url": skinURL,
					}).Error
				}
			}
		}
	}

	if err := bridgeClient.UnbanPlayer(c.Request().Context(), playerIdentifier); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	now := time.Now().UTC()
	if err := db.Model(&models.PlayerBan{}).
		Where("minecraft_uuid = ? AND lifted_at IS NULL", playerUUID).
		Update("lifted_at", now).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to mark ban as lifted"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "player unbanned"})
}
