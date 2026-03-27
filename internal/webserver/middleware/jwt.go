package middleware

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"spoutmc/internal/access"

	"github.com/labstack/echo/v4"
)

const claimsKey = "jwt_claims"

func JWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Path() == "/api/v1/git/webhook" || c.Request().URL.Path == "/api/v1/git/webhook" {
			// #region agent log
			debugAuthLog("run1", "H1", "internal/webserver/middleware/jwt.go:20", "JWT middleware reached webhook path", map[string]any{
				"path":             c.Request().URL.Path,
				"method":           c.Request().Method,
				"hasAuthHeader":    c.Request().Header.Get("Authorization") != "",
				"hasAccessTokenQS": c.QueryParam("access_token") != "",
			})
			// #endregion
		}

		authHeader := c.Request().Header.Get("Authorization")
		var token string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return echo.NewHTTPError(401, "Invalid authorization header")
			}
			token = parts[1]
		} else {
			token = c.QueryParam("access_token")
		}

		if token == "" {
			if c.Path() == "/api/v1/git/webhook" || c.Request().URL.Path == "/api/v1/git/webhook" {
				// #region agent log
				debugAuthLog("run1", "H1", "internal/webserver/middleware/jwt.go:43", "JWT rejected webhook request: missing auth", map[string]any{
					"path":   c.Request().URL.Path,
					"method": c.Request().Method,
				})
				// #endregion
			}
			return echo.NewHTTPError(401, "Missing authorization header")
		}

		claims, err := access.VerifyToken(token)
		if err != nil {
			return echo.NewHTTPError(401, "Invalid or expired token")
		}

		c.Set(claimsKey, claims)
		return next(c)
	}
}

func debugAuthLog(runID, hypothesisID, location, message string, data map[string]any) {
	f, err := os.OpenFile("/Users/florianthievent/workspace/private/spoutmc/.cursor/debug-87a563.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	entry := map[string]any{
		"sessionId":    "87a563",
		"runId":        runID,
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintln(f, string(b))
}

func GetClaims(c echo.Context) *access.Claims {
	cl, ok := c.Get(claimsKey).(*access.Claims)
	if !ok {
		return nil
	}
	return cl
}
