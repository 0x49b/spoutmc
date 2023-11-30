package database

import (
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"spoutmc/backend/log"
)

type DB struct {
	db *gorm.DB
}

var logger = log.New()

func Start() *DB {
	// todo need the DSN from config file
	dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	return &DB{db}
}

func (d *DB) ValidateIfExists(model interface{}, id int, name string) error {
	var exists bool
	err := d.db.Model(model).
		Select("count(*) > 0").
		Where("id = ?", id).
		Find(&exists).
		Error

	if !exists || err != nil {
		return errors.New(name + " not found")
	}
	return nil

}

// Returns wrapped database
func (d *DB) DB() *gorm.DB {
	return d.db
}
