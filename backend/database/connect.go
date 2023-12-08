package database

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConnectDBThenMigrate() {
	var err error
	pw, err := GetDbPasswordIfExists() // Todo extend this wit configurtion properties or ENV Var
	if err != nil {
		logger.Error("Cannot get DatabasePassword from File")
	}

	dsn := fmt.Sprintf("spoutdbuser:%s@tcp(127.0.0.1:3306)/spout?charset=utf8mb4&parseTime=True&loc=Local", pw)
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("", zap.Error(err))
	}
	migrateDatabase()
}

func migrateDatabase() {
	err := DB.AutoMigrate(&Product{})
	if err != nil {
		logger.Error("", zap.Error(err))
	}
	logger.Info("Applied migrations to Database")
}
