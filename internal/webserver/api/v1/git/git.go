package git

import (
	"net/http"
	"spoutmc/internal/config"
	gitops "spoutmc/internal/git"
	"spoutmc/internal/log"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleGit)

// RegisterGitRoutes registers Git-related API routes
func RegisterGitRoutes(g *echo.Group) {
	git := g.Group("/git")

	git.GET("/status", handleGitStatus)
	git.POST("/webhook", handleWebhook)
	git.POST("/sync", handleManualSync)
}

// @Summary Get GitOps sync status
// @Description Returns current GitOps mode and synchronization status metadata
// @Tags git
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /git/status [get]
func handleGitStatus(c echo.Context) error {
	status := gitops.GetSyncStatus()
	status.Enabled = config.IsGitOpsEnabled()
	if !status.Enabled && status.State == "" {
		status.State = "disabled"
	}
	return c.JSON(http.StatusOK, status)
}

// @Summary Handle Git webhook
// @Description Receives and processes webhook requests from Git providers (GitHub, GitLab)
// @Tags git
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /git/webhook [post]
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

// @Summary Trigger manual Git sync
// @Description Manually triggers synchronization with the GitOps repository
// @Tags git
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /git/sync [post]
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
