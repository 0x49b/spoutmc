package guards

import (
	"net/http"
	"spoutmc/internal/auth"
	"spoutmc/internal/authz"
	"spoutmc/internal/storage"
	"spoutmc/internal/webserver/middleware"

	"github.com/labstack/echo/v4"
)

func RequireClaims(c echo.Context) (*auth.Claims, error) {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}
	return cl, nil
}

func RequireAdmin(c echo.Context) error {
	cl, err := RequireClaims(c)
	if err != nil {
		return err
	}

	db := storage.GetDB()
	if db == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Database not available")
	}

	if !authz.UserHasRole(db, cl.UserID, authz.AdminRoleName) {
		return echo.NewHTTPError(http.StatusForbidden, "admin only")
	}
	return nil
}
