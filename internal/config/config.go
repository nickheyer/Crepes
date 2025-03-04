package config

import (
	"math/rand"
	"os"
	"time"

	"github.com/nickheyer/Crepes/internal/models"
)

var (
	AppConfig  models.AppConfig
	UserAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:90.0) Gecko/20100101 Firefox/90.0",
	}
)

// INITCONFIG INITIALIZES THE APPLICATION CONFIGURATION
func InitConfig() {
	// LOAD CONFIG
	AppConfig = models.AppConfig{
		Port:           8080,
		StoragePath:    "./storage",
		ThumbnailsPath: "./thumbnails",
		DataPath:       "./data",
		MaxConcurrent:  5,
		LogFile:        "scraper.log",
		DefaultTimeout: 5 * time.Minute,
	}

	// ENSURE DIRECTORIES EXIST
	os.MkdirAll(AppConfig.StoragePath, 0755)
	os.MkdirAll(AppConfig.ThumbnailsPath, 0755)
	os.MkdirAll(AppConfig.DataPath, 0755)
}

// GETRANDOMUSERAGENT RETURNS A RANDOM USER AGENT FROM THE LIST
func GetRandomUserAgent() string {
	return UserAgents[rand.Intn(len(UserAgents))]
}
