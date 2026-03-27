package player

import (
	"context"
	"sync"
	"time"

	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func StartPlayerUnbanCron(ctx context.Context) error {
	db := storage.GetDB()
	if db == nil {
		return nil
	}

	bridgeClient := NewBridgeClientFromEnv()
	logger := log.GetLogger(log.ModuleMain)

	c := cron.New(
		cron.WithLocation(time.UTC),
	)

	var mu sync.Mutex

	if _, err := c.AddFunc("* * * * *", func() {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now().UTC()
		var due []models.PlayerBan
		if err := db.Where("lifted_at IS NULL AND until_at IS NOT NULL AND until_at <= ?", now).
			Find(&due).Error; err != nil {
			logger.Error("Failed to query due player bans", zap.Error(err))
			return
		}

		if len(due) == 0 {
			return
		}

		jobCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		for _, ban := range due {
			uuidStr := ban.MinecraftUUID.String()
			if err := bridgeClient.UnbanPlayer(jobCtx, uuidStr); err != nil {
				logger.Warn(
					"Failed to unban player (will retry next hour)",
					zap.Error(err),
					zap.String("minecraftUuid", uuidStr),
				)
				continue
			}

			liftedAt := now
			if err := db.Model(&models.PlayerBan{}).Where("id = ?", ban.ID).Update("lifted_at", liftedAt).Error; err != nil {
				logger.Warn(
					"Failed to mark ban as lifted",
					zap.Error(err),
					zap.Uint("banId", ban.ID),
				)
			}
		}
	}); err != nil {
		return err
	}

	c.Start()

	go func() {
		<-ctx.Done()
		c.Stop()
	}()

	return nil
}
