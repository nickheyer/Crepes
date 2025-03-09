package database

import (
	"log"
	"path/filepath"

	"github.com/nickheyer/Crepes/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SETUP DATABASE CONNECTION
func SetupDatabase(dataPath string) (*gorm.DB, error) {
	// CREATE DB PATH
	dbPath := filepath.Join(dataPath, "crepes.db")
	log.Printf("Using database at: %s", dbPath)

	// OPEN DATABASE CONNECTION
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// ENSURE DEFAULT SETTINGS EXIST
func EnsureDefaultSettings(db *gorm.DB) {
	var count int64
	db.Model(&models.Setting{}).Count(&count)

	// IF NO SETTINGS EXIST, CREATE DEFAULT ONES
	if count == 0 {
		log.Println("Creating default settings...")
		defaultSettings := []models.Setting{
			{Key: "theme", Value: "default"},
			{Key: "defaultView", Value: "grid"},
			{Key: "notificationsEnabled", Value: "true"},
		}

		for _, setting := range defaultSettings {
			if err := db.Create(&setting).Error; err != nil {
				log.Printf("Failed to create default setting %s: %v", setting.Key, err)
			}
		}
	}
}
