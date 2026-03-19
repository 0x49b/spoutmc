package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"spoutmc/internal/log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var logger = log.GetLogger(log.ModuleStorage)

// Connect connects to the SQLite database.
func Connect(ctx context.Context, sqlitePath string) (*gorm.DB, error) {
	_ = ctx // reserved for future use
	if sqlitePath == "" {
		sqlitePath = "data/spoutmc.db"
	}
	if dir := filepath.Dir(sqlitePath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}
	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}
	logger.Info("Connected to SQLite database")
	return db, nil
}
