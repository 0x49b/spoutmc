package infrastructure

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

const (
	passwordLength = 32 // Length of generated passwords in bytes (will be base64 encoded)
)

// GenerateSecurePassword generates a cryptographically secure random password
func GenerateSecurePassword() (string, error) {
	bytes := make([]byte, passwordLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random password: %w", err)
	}
	// Base64 encode and remove padding for cleaner passwords
	password := base64.URLEncoding.EncodeToString(bytes)
	password = strings.TrimRight(password, "=")
	return password, nil
}

// GetOrGeneratePasswords checks if passwords need to be generated and returns them
// Returns map with keys: MARIADB_ROOT_PASSWORD, MARIADB_PASSWORD
// Returns a boolean indicating if new passwords were generated
func GetOrGeneratePasswords(infraContainers []InfrastructureContainer, logger *zap.Logger) (map[string]string, bool, error) {
	// Check if any infrastructure container needs password generation
	needsPasswords := false
	for _, container := range infraContainers {
		for key, value := range container.Env {
			if value == "changeme" && (key == "MARIADB_ROOT_PASSWORD" || key == "MARIADB_PASSWORD") {
				needsPasswords = true
				break
			}
		}
		if needsPasswords {
			break
		}
	}

	// If no passwords are needed, return empty map
	if !needsPasswords {
		return nil, false, nil
	}

	// Generate new passwords
	logger.Info("Generating new database passwords")
	rootPassword, err := GenerateSecurePassword()
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate root password: %w", err)
	}

	userPassword, err := GenerateSecurePassword()
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate user password: %w", err)
	}

	passwords := map[string]string{
		"MARIADB_ROOT_PASSWORD": rootPassword,
		"MARIADB_PASSWORD":      userPassword,
	}

	return passwords, true, nil
}

// PrintPasswordsToConsole displays generated passwords in a formatted box
func PrintPasswordsToConsole(passwords map[string]string) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                           ║")
	fmt.Println("║              Your Database Passwords                      ║")
	fmt.Println("║                                                           ║")

	// Print root password
	if rootPass, exists := passwords["MARIADB_ROOT_PASSWORD"]; exists {
		line := fmt.Sprintf("║  Root Password: %-42s║", rootPass)
		fmt.Println(line)
	}

	// Print user password
	if userPass, exists := passwords["MARIADB_PASSWORD"]; exists {
		line := fmt.Sprintf("║  User Password: %-42s║", userPass)
		fmt.Println(line)
	}

	fmt.Println("║                                                           ║")
	fmt.Println("║        Make sure to keep your passwords safe!            ║")
	fmt.Println("║                                                           ║")
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println()
}
