package access

import "gorm.io/gorm"

var dbProvider func() *gorm.DB

func SetDBProvider(provider func() *gorm.DB) {
	dbProvider = provider
}

func resolveDB(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	if dbProvider != nil {
		return dbProvider()
	}
	return nil
}
