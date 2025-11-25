package infrastructure

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

const (
	passwordLength = 32 // Length of generated passwords in bytes (will be base64 encoded)
	passwordsFile  = ".db-passwords"
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

// GetOrGeneratePasswords reads or generates database passwords
// Returns map with keys: MARIADB_ROOT_PASSWORD, MARIADB_PASSWORD
func GetOrGeneratePasswords(workingDir string, logger *zap.Logger) (map[string]string, error) {
	passwordsPath := filepath.Join(workingDir, passwordsFile)

	// Try to read existing passwords
	passwords, err := readPasswordsFile(passwordsPath)
	if err == nil && len(passwords) == 2 {
		logger.Info("Loaded existing database passwords from .db-passwords")
		return passwords, nil
	}

	// Generate new passwords
	logger.Info("Generating new database passwords")
	rootPassword, err := GenerateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate root password: %w", err)
	}

	userPassword, err := GenerateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user password: %w", err)
	}

	passwords = map[string]string{
		"MARIADB_ROOT_PASSWORD": rootPassword,
		"MARIADB_PASSWORD":      userPassword,
	}

	// Write passwords to file
	if err := writePasswordsFile(passwordsPath, passwords); err != nil {
		return nil, fmt.Errorf("failed to write passwords file: %w", err)
	}

	logger.Info("Generated and saved new database passwords to .db-passwords")
	return passwords, nil
}

// readPasswordsFile reads the .db-passwords file
func readPasswordsFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	passwords := make(map[string]string)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			passwords[key] = value
		}
	}

	return passwords, nil
}

// writePasswordsFile writes passwords to .db-passwords file
func writePasswordsFile(path string, passwords map[string]string) error {
	content := "# Database passwords - DO NOT COMMIT TO GIT\n"
	content += "# Generated automatically by SpoutMC\n\n"

	for key, value := range passwords {
		content += fmt.Sprintf("%s=%s\n", key, value)
	}

	return os.WriteFile(path, []byte(content), 0600) // Read/write for owner only
}
