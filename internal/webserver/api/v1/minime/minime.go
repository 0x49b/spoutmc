package minime

import (
	"net/http"
	"strings"

	"spoutmc/internal/log"
	"spoutmc/internal/minecraft"
	"spoutmc/internal/minime/processor"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleWebserver)

const (
	defaultOutputSize = 512
	minOutputSize     = 64
	maxOutputSize     = 1024
)

func RegisterMinimeRoutes(g *echo.Group) {
	g.POST("/minime", generateMinime)
}

type generateMinimeRequest struct {
	MinecraftName string `json:"minecraftName"`
	Username      string `json:"username"`
	UUID          string `json:"uuid"`
	Size          int    `json:"size"`
	Model         string `json:"model"`
}

type generateMinimeResponse struct {
	UUID           uuid.UUID `json:"uuid"`
	MinecraftName  string    `json:"minecraftName,omitempty"`
	Size           int       `json:"size"`
	Model          string    `json:"model"`
	ImageBase64    string    `json:"imageBase64"`
	SkinTextureURL string    `json:"skinTextureUrl,omitempty"`
}

func generateMinime(c echo.Context) error {
	var req generateMinimeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	identifier := strings.TrimSpace(req.UUID)
	if identifier == "" {
		identifier = strings.TrimSpace(req.MinecraftName)
	}
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}
	if identifier == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Provide minecraftName, username, or uuid",
		})
	}

	slim := false
	switch strings.ToLower(strings.TrimSpace(req.Model)) {
	case "", "normal", "classic", "wide":
		slim = false
	case "slim", "alex":
		slim = true
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": `model must be "normal" or "slim"`,
		})
	}

	size := req.Size
	if size == 0 {
		size = defaultOutputSize
	}
	if size < minOutputSize || size > maxOutputSize {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "size must be between 32 and 512",
		})
	}

	playerUUID, mcName, skinURL, err := minecraft.GetPlayerProfile(identifier)
	if err != nil {
		logger.Info("minime: player lookup failed", zap.String("identifier", identifier), zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Could not resolve Minecraft player or skin",
		})
	}

	img, err := processor.ProcessSkin(skinURL, true, slim, size)
	if err != nil {
		logger.Error("minime: process skin failed", zap.Error(err))
		return c.JSON(http.StatusBadGateway, map[string]string{
			"error": "Failed to generate minime from skin",
		})
	}

	b64, err := processor.EncodeToBase64(img)
	if err != nil {
		logger.Error("minime: encode failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode image"})
	}

	resp := generateMinimeResponse{
		UUID:           playerUUID,
		MinecraftName:  mcName,
		Size:           size,
		Model:          map[bool]string{true: "slim", false: "normal"}[slim],
		ImageBase64:    b64,
		SkinTextureURL: skinURL,
	}

	return c.JSON(http.StatusOK, resp)
}
