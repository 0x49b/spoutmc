package player

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"spoutmc/internal/config"
	"spoutmc/internal/models"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestIngestPlayerChatReplyReturnsServiceUnavailableWithoutForwardingSecret(t *testing.T) {
	originalCfg := config.All()
	t.Cleanup(func() {
		config.UpdateConfiguration(originalCfg)
	})

	tempDir := t.TempDir()
	config.UpdateConfiguration(models.SpoutConfiguration{
		Storage: &models.StorageConfig{DataPath: tempDir},
		Servers: []models.SpoutServer{
			{Name: "proxy", Proxy: true},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/player/chat-ingest", bytes.NewBufferString("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := ingestPlayerChatReply(c); err != nil {
		t.Fatalf("ingest handler returned error: %v", err)
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestIngestPlayerChatReplyAuthWithForwardingSecret(t *testing.T) {
	originalCfg := config.All()
	t.Cleanup(func() {
		config.UpdateConfiguration(originalCfg)
	})

	tempDir := t.TempDir()
	proxySecretDir := filepath.Join(tempDir, "proxy", "server")
	if err := os.MkdirAll(proxySecretDir, 0o755); err != nil {
		t.Fatalf("failed to create secret directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proxySecretDir, "forwarding.secret"), []byte("super-secret"), 0o600); err != nil {
		t.Fatalf("failed to write forwarding secret: %v", err)
	}

	config.UpdateConfiguration(models.SpoutConfiguration{
		Storage: &models.StorageConfig{DataPath: tempDir},
		Servers: []models.SpoutServer{
			{Name: "proxy", Proxy: true},
		},
	})

	wrongReq := httptest.NewRequest(http.MethodPost, "/api/v1/player/chat-ingest", bytes.NewBufferString("{}"))
	wrongReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	wrongReq.Header.Set("X-Spout-Chat-Ingest", "wrong-secret")
	wrongRec := httptest.NewRecorder()
	wrongCtx := echo.New().NewContext(wrongReq, wrongRec)

	if err := ingestPlayerChatReply(wrongCtx); err != nil {
		t.Fatalf("ingest handler returned error for wrong secret: %v", err)
	}
	if wrongRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d for wrong secret, got %d", http.StatusUnauthorized, wrongRec.Code)
	}

	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/player/chat-ingest", bytes.NewBufferString("{}"))
	okReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	okReq.Header.Set("X-Spout-Chat-Ingest", "super-secret")
	okRec := httptest.NewRecorder()
	okCtx := echo.New().NewContext(okReq, okRec)

	if err := ingestPlayerChatReply(okCtx); err != nil {
		t.Fatalf("ingest handler returned error for matching secret: %v", err)
	}
	if okRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d after auth with invalid payload, got %d", http.StatusBadRequest, okRec.Code)
	}
}
