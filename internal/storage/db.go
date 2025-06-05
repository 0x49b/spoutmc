package storage

import (
	"fmt"
	"os"
	"spoutmc/internal/log"
	"spoutmc/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB
var logger = log.GetLogger()

func InitDB() error {
	var err error
	if err = godotenv.Load(); err != nil {
		return fmt.Errorf("💾 failed to load .env file: %w", err)
	}

	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		return fmt.Errorf("💾 SQLITE_DB_PATH not set in .env file")
	}

	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("💾 failed to connect to SQLite database: %w", err)
	}

	logger.Info("💾 Successfully connected to SQLite database")

	err = db.AutoMigrate(&models.User{}, &models.SpoutServer{})
	if err != nil {
		return fmt.Errorf("💾 failed to migrate database schema: %w", err)
	}

	logger.Info("💾 Successfully migrated database schema")

	return nil
}

func GetDB() *gorm.DB {
	return db
}
