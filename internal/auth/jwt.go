package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultJWTSecret = "spoutmc-jwt-secret-change-in-production"

// Claims holds JWT claims for SpoutMC users
type Claims struct {
	UserID       uint     `json:"userId"`
	Email        string   `json:"email"`
	DisplayName  string   `json:"displayName"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT for the given user
func GenerateToken(userID uint, email, displayName string, roles []string, permissionKeys []string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = defaultJWTSecret
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r
	}

	perms := make([]string, len(permissionKeys))
	copy(perms, permissionKeys)

	claims := Claims{
		UserID:      userID,
		Email:       email,
		DisplayName: displayName,
		Roles:       roleNames,
		Permissions: perms,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyToken parses and validates a JWT, returning the claims
func VerifyToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = defaultJWTSecret
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
