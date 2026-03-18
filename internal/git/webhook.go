package git

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// WebhookHandler handles Git webhook requests
type WebhookHandler struct {
	poller *Poller
	secret string
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(poller *Poller, secret string) *WebhookHandler {
	return &WebhookHandler{
		poller: poller,
		secret: secret,
	}
}

// HandleWebhook processes incoming webhook requests
func (h *WebhookHandler) HandleWebhook(c echo.Context) error {
	// Determine webhook type from headers
	webhookType := h.detectWebhookType(c)

	logger.Info("Received webhook request",
		zap.String("type", webhookType),
		zap.String("remote_addr", c.RealIP()))

	// Verify webhook signature if secret is configured
	if h.secret != "" {
		if err := h.verifySignature(c, webhookType); err != nil {
			logger.Warn("Webhook signature verification failed", zap.Error(err))
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid signature",
			})
		}
	}

	// Trigger sync
	if err := h.poller.TriggerSync(c.Request().Context()); err != nil {
		logger.Error("Failed to sync after webhook", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to sync configuration",
		})
	}

	logger.Info("Webhook processed successfully")
	commit := h.poller.repo.GetLastCommit()
	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
		"commit": shortCommit(commit),
	})
}

// detectWebhookType detects the type of webhook from headers
func (h *WebhookHandler) detectWebhookType(c echo.Context) string {
	// GitHub webhooks have X-GitHub-Event header
	if c.Request().Header.Get("X-GitHub-Event") != "" {
		return "github"
	}

	// GitLab webhooks have X-Gitlab-Event header
	if c.Request().Header.Get("X-Gitlab-Event") != "" {
		return "gitlab"
	}

	return "unknown"
}

// verifySignature verifies the webhook signature
func (h *WebhookHandler) verifySignature(c echo.Context, webhookType string) error {
	var signature string
	var payload []byte
	var err error

	// Read request body
	payload, err = io.ReadAll(c.Request().Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	// Reset body so it can be read again
	c.Request().Body = io.NopCloser(bytes.NewReader(payload))

	switch webhookType {
	case "github":
		// GitHub uses X-Hub-Signature-256 header
		signature = c.Request().Header.Get("X-Hub-Signature-256")
		if signature == "" {
			return fmt.Errorf("missing X-Hub-Signature-256 header")
		}

		// GitHub signature format: sha256=<signature>
		signature = strings.TrimPrefix(signature, "sha256=")

		// Verify signature
		if err := h.verifyHMACHexSignature(payload, signature); err != nil {
			return err
		}

	case "gitlab":
		// GitLab uses X-Gitlab-Token header
		token := c.Request().Header.Get("X-Gitlab-Token")
		if token == "" {
			return fmt.Errorf("missing X-Gitlab-Token header")
		}

		// GitLab uses simple token comparison
		if token != h.secret {
			return fmt.Errorf("token mismatch")
		}

	default:
		// For unknown webhook types, use generic HMAC verification
		signature = c.Request().Header.Get("X-Hub-Signature-256")
		if signature == "" {
			signature = c.Request().Header.Get("X-Signature")
		}

		if signature == "" {
			return fmt.Errorf("missing signature header")
		}

		signature = strings.TrimPrefix(signature, "sha256=")
		if err := h.verifyHMACHexSignature(payload, signature); err != nil {
			return err
		}
	}

	return nil
}

// computeHMAC computes the HMAC-SHA256 of the payload
func (h *WebhookHandler) computeHMAC(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// verifyHMACHexSignature validates a hex-encoded SHA256 HMAC signature.
func (h *WebhookHandler) verifyHMACHexSignature(payload []byte, signature string) error {
	givenMAC, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding")
	}

	expectedMAC := hmac.New(sha256.New, []byte(h.secret))
	expectedMAC.Write(payload)
	expected := expectedMAC.Sum(nil)

	if !hmac.Equal(givenMAC, expected) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}
