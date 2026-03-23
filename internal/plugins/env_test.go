package plugins

import (
	"strings"
	"testing"

	"spoutmc/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMergePluginsEnv_SystemAndUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.UserPlugin{}, &models.UserPluginServer{}); err != nil {
		t.Fatal(err)
	}
	p := models.UserPlugin{Name: "Test", URL: "https://example.com/a.jar"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.UserPluginServer{UserPluginID: p.ID, ServerName: "s1"}).Error; err != nil {
		t.Fatal(err)
	}

	s := models.SpoutServer{Name: "s1", Lobby: true, Env: map[string]string{}}
	out := MergePluginsEnv(db, s)
	merged := out["PLUGINS"]
	if merged == "" {
		t.Fatal("expected PLUGINS")
	}
	if !strings.Contains(merged, "servertap") || !strings.Contains(merged, "example.com/a.jar") {
		t.Fatalf("unexpected PLUGINS: %q", merged)
	}
}

func TestMergePluginsEnv_YamlAppend(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.UserPlugin{}, &models.UserPluginServer{}); err != nil {
		t.Fatal(err)
	}

	s := models.SpoutServer{
		Name: "s1",
		Env: map[string]string{
			"PLUGINS": "https://example.com/extra.jar",
		},
	}
	out := MergePluginsEnv(db, s)
	merged := out["PLUGINS"]
	if !strings.Contains(merged, "extra.jar") {
		t.Fatalf("expected yaml plugin preserved: %q", merged)
	}
}
