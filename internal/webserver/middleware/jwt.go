package middleware

import (
	"strings"

	"spoutmc/internal/auth"

	"github.com/labstack/echo/v4"
)

const claimsKey = "jwt_claims"

// JWT validates the Bearer token and sets claims in context
func JWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(401, "Missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return echo.NewHTTPError(401, "Invalid authorization header")
		}

		claims, err := auth.VerifyToken(parts[1])
		if err != nil {
			return echo.NewHTTPError(401, "Invalid or expired token")
		}

		c.Set(claimsKey, claims)
		return next(c)
	}
}

// GetClaims returns the JWT claims from context (nil if not authenticated)
func GetClaims(c echo.Context) *auth.Claims {
	cl, ok := c.Get(claimsKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return cl
}
