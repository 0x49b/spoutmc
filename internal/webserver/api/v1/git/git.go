package git

import (
	"net/http"
	gitops "spoutmc/internal/git"
	"spoutmc/internal/log"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var logger = log.GetLogger()

// RegisterGitRoutes registers Git-related API routes
func RegisterGitRoutes(g *echo.Group) {
	git := g.Group("/git")

	git.POST("/webhook", handleWebhook)
	git.POST("/sync", handleManualSync)
}

// handleWebhook handles incoming webhook requests from Git providers
func handleWebhook(c echo.Context) error {
	handler := gitops.GetWebhookHandler()
	if handler == nil {
		logger.Warn("Webhook received but handler not initialized")
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "GitOps not enabled or webhook handler not initialized",
		})
	}

	return handler.HandleWebhook(c)
}

// handleManualSync manually triggers a Git sync
func handleManualSync(c echo.Context) error {
	logger.Info("Manual sync triggered via API", zap.String("remote_addr", c.RealIP()))

	if err := gitops.TriggerManualSync(); err != nil {
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
