package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"spoutmc/internal/config"
	gitops "spoutmc/internal/git"
	"spoutmc/internal/log"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleGit)

func RegisterGitRoutes(g *echo.Group) {
	git := g.Group("/git")

	git.GET("/status", handleGitStatus)
	git.POST("/webhook", handleWebhook)
	git.POST("/sync", handleManualSync)
}

func handleGitStatus(c echo.Context) error {
	status := gitops.GetSyncStatus()
	status.Enabled = config.IsGitOpsEnabled()
	if !status.Enabled && status.State == "" {
		status.State = "disabled"
	}
	return c.JSON(http.StatusOK, status)
}

func handleWebhook(c echo.Context) error {
	// #region agent log
	debugWebhookRouteLog("run1", "H2", "internal/webserver/api/v1/git/git.go:36", "handleWebhook reached", map[string]any{
		"path":   c.Request().URL.Path,
		"method": c.Request().Method,
	})
	// #endregion

	handler := gitops.GetWebhookHandler()
	if handler == nil {
		logger.Warn("Webhook received but handler not initialized")
		// #region agent log
		debugWebhookRouteLog("run1", "H3", "internal/webserver/api/v1/git/git.go:45", "handleWebhook missing handler", map[string]any{
			"path": c.Request().URL.Path,
		})
		// #endregion
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "GitOps not enabled or webhook handler not initialized",
		})
	}

	return handler.HandleWebhook(c)
}

func debugWebhookRouteLog(runID, hypothesisID, location, message string, data map[string]any) {
	f, err := os.OpenFile("/Users/florianthievent/workspace/private/spoutmc/.cursor/debug-87a563.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	entry := map[string]any{
		"sessionId":    "87a563",
		"runId":        runID,
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintln(f, string(b))
}

func handleManualSync(c echo.Context) error {
	logger.Info("Manual sync triggered via API", zap.String("remote_addr", c.RealIP()))

	if err := gitops.TriggerManualSync(c.Request().Context()); err != nil {
		logger.Error("Failed to trigger manual sync", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Configuration synced successfully",
	})
}
