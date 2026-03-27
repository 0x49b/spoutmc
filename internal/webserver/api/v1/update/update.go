package update

import (
	"context"
	"net/http"
	"time"

	updatepkg "spoutmc/internal/update"
	"spoutmc/internal/webserver/guards"

	"github.com/labstack/echo/v4"
)

func RegisterUpdateRoutes(g *echo.Group) {
	grp := g.Group("/update")
	grp.GET("/status", getUpdateStatus)
	grp.POST("/check", triggerUpdateCheck)
	grp.POST("/start", startUpdate)
}

func getUpdateStatus(c echo.Context) error {
	if err := guards.RequireAdmin(c); err != nil {
		return err
	}

	mgr := updatepkg.Get()
	if mgr == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Update service not initialized"})
	}

	return c.JSON(http.StatusOK, mgr.GetStatus())
}

func triggerUpdateCheck(c echo.Context) error {
	if err := guards.RequireAdmin(c); err != nil {
		return err
	}

	mgr := updatepkg.Get()
	if mgr == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Update service not initialized"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()
	status, err := mgr.CheckNow(ctx)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]interface{}{
			"error":  err.Error(),
			"status": status,
		})
	}

	return c.JSON(http.StatusOK, status)
}

func startUpdate(c echo.Context) error {
	if err := guards.RequireAdmin(c); err != nil {
		return err
	}

	mgr := updatepkg.Get()
	if mgr == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Update service not initialized"})
	}

	if err := mgr.StartUpdate(); err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "Update process started",
	})
}
