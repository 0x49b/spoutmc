package middleware

import (
	"strings"

	"spoutmc/internal/access"

	"github.com/labstack/echo/v4"
)

const claimsKey = "jwt_claims"

// JWT validates the Bearer token and sets claims in context.
// Browsers cannot set headers on EventSource; for SSE, the same JWT may be passed as access_token.
func JWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
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

// GetClaims returns the JWT claims from context (nil if not authenticated)
func GetClaims(c echo.Context) *access.Claims {
	cl, ok := c.Get(claimsKey).(*access.Claims)
	if !ok {
		return nil
	}
	return cl
}
