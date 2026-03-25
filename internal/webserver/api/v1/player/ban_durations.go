package player

import (
	"net/http"
	"time"

	"spoutmc/internal/config"
	"spoutmc/internal/models"

	"github.com/labstack/echo/v4"
)

type BanDurationOptionDTO struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	DurationSeconds int64  `json:"durationSeconds"`
}

type BanDurationsResponseDTO struct {
	Options []BanDurationOptionDTO `json:"options"`
}

func defaultBanDurationOptions() []models.BanDurationOption {
	return []models.BanDurationOption{
		{Key: "1h", Label: "1 hour", Duration: 1 * time.Hour},
		{Key: "5h", Label: "5 hours", Duration: 5 * time.Hour},
		{Key: "1d", Label: "1 day", Duration: 24 * time.Hour},
		{Key: "2d", Label: "2 days", Duration: 48 * time.Hour},
		{Key: "2w", Label: "2 weeks", Duration: 14 * 24 * time.Hour},
	}
}

func getBanDurations(c echo.Context) error {
	cfg := config.All()

	opts := defaultBanDurationOptions()
	if cfg.PlayerBans != nil && len(cfg.PlayerBans.BanDurations) > 0 {
		opts = cfg.PlayerBans.BanDurations
	}

	resp := BanDurationsResponseDTO{
		Options: make([]BanDurationOptionDTO, 0, len(opts)),
	}
	for _, o := range opts {
		resp.Options = append(resp.Options, BanDurationOptionDTO{
			Key:             o.Key,
			Label:           o.Label,
			DurationSeconds: int64(o.Duration.Seconds()),
		})
	}

	return c.JSON(http.StatusOK, resp)
}
