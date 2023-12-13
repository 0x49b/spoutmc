package config

import (
	"errors"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"os"
	"spoutmc/backend/dbcontext"
	"spoutmc/backend/models"
	"strconv"
)

type Config struct {
	port string
	env  string
}

type Application struct {
	Db   *dbcontext.DB
	Port string
}

func New(port string, env string) *Config {
	return &Config{
		port,
		env,
	}
}

func (c *Config) Bootstrap() *Application {
	dbctx := c.bootstrapDB()
	port, _ := c.bootstrapAPI()

	return &Application{
		dbctx,
		port,
	}
}

func (c *Config) bootstrapDB() *dbcontext.DB {
	confPath, _ := os.UserConfigDir()
	appDir := confPath + "/spout"
	sqliteFile := "/spout.db"

	// Check SQLite db if exists
	if _, err := os.Stat(appDir + sqliteFile); err != nil {
		os.MkdirAll(appDir, 0700)
		os.Create(appDir + sqliteFile)
	}

	sqlite := sqlite.Open(appDir + sqliteFile)

	db, err := gorm.Open(sqlite, &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})

	db.AutoMigrate(&models.Project{}, &models.Task{})

	if err != nil {
		panic("failed to connect to database")
	}

	return dbcontext.New(db)
}

func (c *Config) bootstrapAPI() (port string, err error) {
	if c.port == "" {
		return "8081", nil
	}
	if _, err := strconv.Atoi(c.port); err != nil {
		return "8081", errors.New("PORT is not a number")
	}
	return c.port, nil
}
