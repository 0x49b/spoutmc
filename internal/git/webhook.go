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

type WebhookHandler struct {
	poller *Poller
	secret string
}

func NewWebhookHandler(poller *Poller, secret string) *WebhookHandler {
	if strings.TrimSpace(secret) == "" {
		logger.Warn("Webhook secret is empty; signature verification is disabled. Do not use this in production.")
	}

	return &WebhookHandler{
		poller: poller,
		secret: secret,
	}
}

func (h *WebhookHandler) HandleWebhook(c echo.Context) error {
	webhookType := h.detectWebhookType(c)
	eventName := h.extractEventName(c, webhookType)

	logger.Info("Received webhook request",
		zap.String("type", webhookType),
		zap.String("event", eventName),
		zap.String("remote_addr", c.RealIP()))

	if h.secret != "" {
		if err := h.verifySignature(c, webhookType); err != nil {
			logger.Warn("Webhook signature verification failed", zap.Error(err))
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid signature",
			})
		}
	} else {
		logger.Warn("Processing webhook without signature verification because webhook secret is not configured")
	}

	if !h.shouldProcessEvent(webhookType, eventName) {
		logger.Info("Ignoring non-push webhook event",
			zap.String("type", webhookType),
			zap.String("event", eventName))
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ignored",
			"event":  eventName,
		})
	}

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

func (h *WebhookHandler) detectWebhookType(c echo.Context) string {
	if c.Request().Header.Get("X-GitHub-Event") != "" {
		return "github"
	}

	if c.Request().Header.Get("X-Gitlab-Event") != "" {
		return "gitlab"
	}

	return "unknown"
}

func (h *WebhookHandler) extractEventName(c echo.Context, webhookType string) string {
	switch webhookType {
	case "github":
		return strings.TrimSpace(strings.ToLower(c.Request().Header.Get("X-GitHub-Event")))
	case "gitlab":
		return strings.TrimSpace(strings.ToLower(c.Request().Header.Get("X-Gitlab-Event")))
	default:
		return "unknown"
	}
}

func (h *WebhookHandler) shouldProcessEvent(webhookType string, eventName string) bool {
	switch webhookType {
	case "github":
		return eventName == "push"
	case "gitlab":
		return strings.Contains(eventName, "push")
	default:
		return true
	}
}

func (h *WebhookHandler) verifySignature(c echo.Context, webhookType string) error {
	var signature string
	var payload []byte
	var err error

	payload, err = io.ReadAll(c.Request().Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	c.Request().Body = io.NopCloser(bytes.NewReader(payload))

	switch webhookType {
	case "github":
		signature = c.Request().Header.Get("X-Hub-Signature-256")
		if signature == "" {
			return fmt.Errorf("missing X-Hub-Signature-256 header")
		}

		signature = strings.TrimPrefix(signature, "sha256=")

		if err := h.verifyHMACHexSignature(payload, signature); err != nil {
			return err
		}

	case "gitlab":
		token := c.Request().Header.Get("X-Gitlab-Token")
		if token == "" {
			return fmt.Errorf("missing X-Gitlab-Token header")
		}

		if !hmac.Equal([]byte(token), []byte(h.secret)) {
			return fmt.Errorf("token mismatch")
		}

	default:
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

func (h *WebhookHandler) computeHMAC(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

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
