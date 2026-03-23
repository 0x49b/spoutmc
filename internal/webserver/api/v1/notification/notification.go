package notification

import (
	"errors"
	"net/http"
	"strconv"

	"spoutmc/internal/log"
	"spoutmc/internal/notifications"
	"spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var logger = log.GetLogger(log.ModuleWebserver)

func RegisterNotificationRoutes(g *echo.Group) {
	grp := g.Group("/notification")
	grp.GET("", listNotifications)
	grp.POST("/:id/dismiss", dismissNotification)
}

func listNotifications(c echo.Context) error {
	entries, err := notifications.ListOpen()
	if err != nil {
		logger.Error("Failed to list notifications", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list notifications",
		})
	}

	out := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]interface{}{
			"id":          e.ID,
			"key":         e.Key,
			"severity":    e.Severity,
			"title":       e.Title,
			"message":     e.Message,
			"source":      e.Source,
			"isOpen":      e.IsOpen,
			"createdAt":   e.CreatedAt,
			"updatedAt":   e.UpdatedAt,
			"dismissedAt": e.DismissedAt,
		})
	}
	return c.JSON(http.StatusOK, out)
}

func dismissNotification(c echo.Context) error {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	idValue := c.Param("id")
	id, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid notification ID"})
	}

	if err := notifications.Dismiss(uint(id), cl.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Notification not found"})
		}
		logger.Error("Failed to dismiss notification", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to dismiss notification"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Notification dismissed",
	})
}
