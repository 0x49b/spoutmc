package plugins

import (
	"bufio"
	"strings"

	"spoutmc/internal/models"

	"gorm.io/gorm"
)

// MergePluginsEnv returns a copy of server.Env with PLUGINS set to the merged list of
// system-managed URLs, user-registry URLs for this server, and any URLs from the
// original env PLUGINS value (advanced users). Order: system, user DB, then YAML PLUGINS.
func MergePluginsEnv(db *gorm.DB, s models.SpoutServer) map[string]string {
	out := make(map[string]string)
	for k, v := range s.Env {
		out[k] = v
	}
	kind := KindFromSpoutServer(s.Proxy, s.Lobby)
	merged := buildMergedPluginURLs(db, s.Name, kind, out["PLUGINS"])
	if merged == "" {
		delete(out, "PLUGINS")
	} else {
		out["PLUGINS"] = merged
	}
	return out
}

func buildMergedPluginURLs(db *gorm.DB, serverName string, kind ServerKind, yamlPlugins string) string {
	seen := make(map[string]struct{})
	var ordered []string

	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		ordered = append(ordered, u)
	}

	for _, u := range SystemURLsForKind(kind) {
		add(u)
	}

	if db != nil {
		var rows []models.UserPluginServer
		if err := db.Preload("UserPlugin").Where("server_name = ?", serverName).Find(&rows).Error; err == nil {
			for _, row := range rows {
				if row.UserPlugin.URL != "" {
					add(row.UserPlugin.URL)
				}
			}
		}
	}

	for _, u := range parsePluginsEnvList(yamlPlugins) {
		add(u)
	}

	return strings.Join(ordered, "\n")
}

func parsePluginsEnvList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, part := range strings.Split(line, ",") {
			part = strings.TrimSpace(part)
			if part != "" && !strings.HasPrefix(part, "#") {
				out = append(out, part)
			}
		}
	}
	return out
}
