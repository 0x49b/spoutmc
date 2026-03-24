package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"spoutmc/internal/access"
	cfgpkg "spoutmc/internal/config"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/models"
	"spoutmc/internal/plugins"
	"spoutmc/internal/storage"
	"spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var logger = log.GetLogger(log.ModuleAPI)

// RegisterPluginRoutes registers plugin registry routes (JWT required).
func RegisterPluginRoutes(g *echo.Group) {
	g.GET("/plugin", listPlugins)
	g.POST("/plugin", createPlugin)
	g.PUT("/plugin/:id", updatePlugin)
	g.DELETE("/plugin/:id", deletePlugin)
}

type pluginDTO struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	Description   string   `json:"description,omitempty"`
	SystemManaged bool     `json:"systemManaged"`
	ServerNames   []string `json:"serverNames"`
	Kinds         []string `json:"kinds,omitempty"`
}

type createUpdatePluginRequest struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	ServerNames []string `json:"serverNames"`
}

func listPlugins(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}
	if !access.ClaimsHasPermission(claims, "server.list.read") {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	db := storage.GetDB()
	if db == nil {
		return c.JSON(http.StatusOK, []pluginDTO{})
	}

	var out []pluginDTO

	for _, e := range plugins.SystemPlugins {
		kinds := make([]string, len(e.Kinds))
		for i, k := range e.Kinds {
			kinds[i] = string(k)
		}
		out = append(out, pluginDTO{
			ID:            "system:" + e.ID,
			Name:          e.Name,
			URL:           e.URL,
			Description:   e.Description,
			SystemManaged: true,
			ServerNames:   serverNamesMatchingKinds(e.Kinds),
			Kinds:         kinds,
		})
	}

	var userPlugins []models.UserPlugin
	if err := db.Preload("Servers").Order("name").Find(&userPlugins).Error; err != nil {
		logger.Error("list plugins", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list plugins"})
	}

	for _, p := range userPlugins {
		names := make([]string, 0, len(p.Servers))
		for _, a := range p.Servers {
			names = append(names, a.ServerName)
		}
		out = append(out, pluginDTO{
			ID:            strconv.FormatUint(uint64(p.ID), 10),
			Name:          p.Name,
			URL:           p.URL,
			Description:   p.Description,
			SystemManaged: false,
			ServerNames:   names,
		})
	}

	return c.JSON(http.StatusOK, out)
}

func serverNamesMatchingKinds(kinds []plugins.ServerKind) []string {
	var want map[plugins.ServerKind]struct{}
	if len(kinds) > 0 {
		want = make(map[plugins.ServerKind]struct{})
		for _, k := range kinds {
			want[k] = struct{}{}
		}
	}
	var names []string
	for _, s := range cfgpkg.All().Servers {
		k := plugins.KindFromSpoutServer(s.Proxy, s.Lobby)
		if want != nil {
			if _, ok := want[k]; !ok {
				continue
			}
		}
		names = append(names, s.Name)
	}
	return names
}

func createPlugin(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if !requireManagePlugins(c, claims) {
		return nil
	}

	var req createUpdatePluginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if err := validatePluginPayload(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	for _, sn := range req.ServerNames {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			continue
		}
		if !cfgpkg.IsValidServerName(sn) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Unknown server: %s", sn)})
		}
	}

	db := storage.GetDB()
	p := models.UserPlugin{
		Name:        strings.TrimSpace(req.Name),
		URL:         strings.TrimSpace(req.URL),
		Description: strings.TrimSpace(req.Description),
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		for _, sn := range req.ServerNames {
			sn = strings.TrimSpace(sn)
			if sn == "" {
				continue
			}
			row := models.UserPluginServer{UserPluginID: p.ID, ServerName: sn}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("create plugin", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create plugin"})
	}

	recreateServersForPluginChange(c.Request().Context(), req.ServerNames)

	return c.JSON(http.StatusCreated, pluginDTO{
		ID:            strconv.FormatUint(uint64(p.ID), 10),
		Name:          p.Name,
		URL:           p.URL,
		Description:   p.Description,
		SystemManaged: false,
		ServerNames:   normalizeServerNames(req.ServerNames),
	})
}

func updatePlugin(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if !requireManagePlugins(c, claims) {
		return nil
	}

	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid plugin id"})
	}

	var req createUpdatePluginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if err := validatePluginPayload(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	for _, sn := range req.ServerNames {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			continue
		}
		if !cfgpkg.IsValidServerName(sn) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Unknown server: %s", sn)})
		}
	}

	db := storage.GetDB()
	var p models.UserPlugin
	if err := db.Preload("Servers").First(&p, uint(id64)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Plugin not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load plugin"})
	}

	oldNames := make([]string, 0, len(p.Servers))
	for _, a := range p.Servers {
		oldNames = append(oldNames, a.ServerName)
	}

	p.Name = strings.TrimSpace(req.Name)
	p.URL = strings.TrimSpace(req.URL)
	p.Description = strings.TrimSpace(req.Description)
	if err := db.Save(&p).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update plugin"})
	}

	if err := db.Where("user_plugin_id = ?", p.ID).Delete(&models.UserPluginServer{}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update assignments"})
	}

	for _, sn := range req.ServerNames {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			continue
		}
		row := models.UserPluginServer{UserPluginID: p.ID, ServerName: sn}
		if err := db.Create(&row).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to assign server"})
		}
	}

	affected := append([]string(nil), oldNames...)
	affected = append(affected, req.ServerNames...)
	recreateServersForPluginChange(c.Request().Context(), affected)

	return c.JSON(http.StatusOK, pluginDTO{
		ID:            strconv.FormatUint(uint64(p.ID), 10),
		Name:          p.Name,
		URL:           p.URL,
		Description:   p.Description,
		SystemManaged: false,
		ServerNames:   normalizeServerNames(req.ServerNames),
	})
}

func deletePlugin(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if !requireManagePlugins(c, claims) {
		return nil
	}

	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid plugin id"})
	}

	db := storage.GetDB()
	var p models.UserPlugin
	if err := db.Preload("Servers").First(&p, uint(id64)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Plugin not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load plugin"})
	}

	if len(p.Servers) > 0 {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Remove all server assignments before deleting this plugin",
		})
	}

	if err := db.Delete(&p).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete plugin"})
	}

	return c.NoContent(http.StatusNoContent)
}

func requireManagePlugins(c echo.Context, claims *access.Claims) bool {
	if claims == nil {
		_ = c.NoContent(http.StatusUnauthorized)
		return false
	}
	if !access.ClaimsCanManagePlugins(claims) {
		_ = c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
		return false
	}
	return true
}

func validatePluginPayload(req *createUpdatePluginRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.URL) == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(strings.TrimSpace(req.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("url must be a valid http or https URL to a plugin jar")
	}
	return nil
}

func normalizeServerNames(names []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func recreateServersForPluginChange(ctx context.Context, serverNames []string) {
	cfg := cfgpkg.All()
	dataPath := ""
	if cfg.Storage != nil {
		dataPath = cfg.Storage.DataPath
	}
	if dataPath == "" {
		logger.Warn("recreate servers for plugin change: empty data path")
		return
	}

	seen := map[string]struct{}{}
	for _, raw := range serverNames {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		var sp *models.SpoutServer
		for i := range cfg.Servers {
			if cfg.Servers[i].Name == name {
				sp = &cfg.Servers[i]
				break
			}
		}
		if sp == nil {
			continue
		}
		if !docker.ContainerExists(ctx, name) {
			continue
		}
		if err := docker.RecreateContainer(ctx, *sp, dataPath); err != nil {
			logger.Warn("Recreate container after plugin change",
				zap.String("server", name),
				zap.Error(err))
		}
	}
}
