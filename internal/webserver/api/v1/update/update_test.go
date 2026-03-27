package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"spoutmc/internal/access"
	"spoutmc/internal/models"
	"spoutmc/internal/storage"
	updatepkg "spoutmc/internal/update"
	authmw "spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "spoutmc-test.db")
	t.Setenv("SQLITE_DB_PATH", dbPath)
	if err := storage.InitDB(context.Background()); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
}

func createUserWithRole(t *testing.T, roleName string) models.User {
	t.Helper()

	db := storage.GetDB()
	if db == nil {
		t.Fatal("db is nil")
	}

	user := models.User{
		DisplayName: "Test User",
		Email:       roleName + "@example.test",
		Password:    "not-used",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if roleName != "" {
		var role models.Role
		if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
			t.Fatalf("load role %q failed: %v", roleName, err)
		}
		if err := db.Model(&user).Association("Roles").Append(&role); err != nil {
			t.Fatalf("assign role failed: %v", err)
		}
	}
	return user
}

func issueToken(t *testing.T, user models.User, roles []string) string {
	t.Helper()
	token, err := access.GenerateToken(user.ID, user.Email, user.DisplayName, roles, nil)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	return token
}

func newUpdateEcho() *echo.Echo {
	e := echo.New()
	protected := e.Group("", authmw.JWT)
	RegisterUpdateRoutes(protected)
	return e
}

func TestUpdateStatusRequiresAuth(t *testing.T) {
	setupTestDB(t)
	t.Setenv("SPOUTMC_UPDATE_REPO", "owner/repo")
	updatepkg.Init("0.0.6")

	e := newUpdateEcho()
	req := httptest.NewRequest(http.MethodGet, "/update/status", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestUpdateStatusRequiresAdmin(t *testing.T) {
	setupTestDB(t)
	t.Setenv("SPOUTMC_UPDATE_REPO", "owner/repo")
	updatepkg.Init("0.0.6")

	user := createUserWithRole(t, "manager")
	token := issueToken(t, user, []string{"manager"})

	e := newUpdateEcho()
	req := httptest.NewRequest(http.MethodGet, "/update/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestUpdateStatusForAdmin(t *testing.T) {
	setupTestDB(t)
	t.Setenv("SPOUTMC_UPDATE_REPO", "owner/repo")
	updatepkg.Init("0.0.6")

	user := createUserWithRole(t, "admin")
	token := issueToken(t, user, []string{"admin"})

	e := newUpdateEcho()
	req := httptest.NewRequest(http.MethodGet, "/update/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if payload["currentVersion"] != "0.0.6" {
		t.Fatalf("unexpected currentVersion: %v", payload["currentVersion"])
	}
}

func TestMain(m *testing.M) {
	// Keep environment deterministic for JWT secret defaults and DB path per test.
	_ = os.Unsetenv("JWT_SECRET")
	os.Exit(m.Run())
}
