package setup

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"spoutmc/internal/access"
	"spoutmc/internal/config"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

var lock = sync.Mutex{}
var logger = log.GetLogger(log.ModuleSetup)

// RegisterSetupRoutes registers setup-related API endpoints.
func RegisterSetupRoutes(g *echo.Group) {
	g.POST("/setup/complete", completeSetup)
}

// SetupRequest represents the request body for completing setup
type SetupRequest struct {
	DataPath         string `json:"dataPath" binding:"required"`
	AcceptEula       bool   `json:"acceptEula" binding:"required"`
	AdminEmail       string `json:"adminEmail"`
	AdminPassword    string `json:"adminPassword"`
	AdminDisplayName string `json:"adminDisplayName"`
}

func completeSetup(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	var req SetupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.DataPath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Data path is required",
		})
	}

	if !req.AcceptEula {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "EULA must be accepted",
		})
	}

	logger.Info("Completing setup",
		zap.String("dataPath", req.DataPath),
		zap.Bool("eulaAccepted", req.AcceptEula))

	// Get current configuration
	currentConfig := config.All()

	// Update storage configuration
	currentConfig.Storage = &models.StorageConfig{
		DataPath: req.DataPath,
	}

	// Update EULA configuration
	currentConfig.EULA = &models.EULAConfig{
		Accepted:   req.AcceptEula,
		AcceptedOn: time.Now(),
	}

	// Save to config file
	if err := saveConfigToFile(currentConfig); err != nil {
		logger.Error("Failed to save configuration", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to save configuration: %v", err),
		})
	}

	// Reload configuration
	if err := config.ReadConfiguration(); err != nil {
		logger.Error("Failed to reload configuration", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to reload configuration: %v", err),
		})
	}

	// Create initial admin user if credentials provided and no users exist
	if req.AdminEmail != "" && req.AdminPassword != "" && len(req.AdminPassword) >= 6 {
		db := storage.GetDB()
		if db != nil {
			var count int64
			if db.Model(&models.User{}).Count(&count); count == 0 {
				hashedPassword, err := access.Hash(req.AdminPassword)
				if err != nil {
					logger.Warn("Failed to hash admin password", zap.Error(err))
				} else {
					adminUser := models.User{
						Email:       req.AdminEmail,
						Password:    hashedPassword,
						DisplayName: req.AdminDisplayName,
					}
					if adminUser.DisplayName == "" {
						adminUser.DisplayName = "Admin"
					}
					if err := db.Transaction(func(tx *gorm.DB) error {
						if err := tx.Create(&adminUser).Error; err != nil {
							return err
						}
						var adminRole models.Role
						if err := tx.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
							return err
						}
						return tx.Model(&adminUser).Association("Roles").Append(&adminRole)
					}); err != nil {
						logger.Warn("Failed to create initial admin user", zap.Error(err))
					} else {
						logger.Info("Created initial admin user", zap.String("email", req.AdminEmail))
					}
				}
			}
		}
	}

	logger.Info("Setup completed successfully")

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Setup completed successfully",
	})
}

// saveConfigToFile saves only storage and eula configuration to config/spoutmc.yaml
// This preserves existing git config, servers, and versions without overwriting them
func saveConfigToFile(cfg models.SpoutConfiguration) error {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Try both .yaml and .yml
	configPaths := []string{
		filepath.Join(wd, "config", "spoutmc.yaml"),
		filepath.Join(wd, "config", "spoutmc.yml"),
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		// Default to .yaml if neither exists
		configPath = configPaths[0]
	}

	// Read existing config file
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	// Parse existing config as a generic map to preserve structure and comments
	var existingConfig map[string]interface{}
	if err := yaml.Unmarshal(existingData, &existingConfig); err != nil {
		return fmt.Errorf("failed to parse existing config: %w", err)
	}

	// Update only storage and eula sections
	if cfg.Storage != nil {
		existingConfig["storage"] = map[string]interface{}{
			"data_path": cfg.Storage.DataPath,
		}
	}

	if cfg.EULA != nil {
		existingConfig["eula"] = map[string]interface{}{
			"accepted":    cfg.EULA.Accepted,
			"accepted_on": cfg.EULA.AcceptedOn,
		}
	}

	// Marshal back to YAML
	yamlData, err := yaml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info("Storage and EULA configuration saved",
		zap.String("path", configPath),
		zap.String("dataPath", cfg.Storage.DataPath),
		zap.Bool("eulaAccepted", cfg.EULA.Accepted))

	return nil
}
