package config

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
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
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36 Edg/94.0.992.47",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (iPad; CPU OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
	}
)

// INITCONFIG INITIALIZES THE APPLICATION CONFIGURATION
func InitConfig() {
	// LOAD CONFIG
	AppConfig = models.AppConfig{
		Port:              8080,
		StoragePath:       "./storage",
		ThumbnailsPath:    "./thumbnails",
		DataPath:          "./data",
		LogsPath:          "./logs",
		ErrorsPath:        "./logs/errors",
		MaxConcurrent:     5,
		MaxBrowsers:       3,
		MaxBrowserTabs:    5,
		LogFile:           "crepes.log",
		DefaultTimeout:    5 * time.Minute,
		BrowserLifetime:   30 * time.Minute,
		StoreErrorDetails: true,
		DevMode:           false,
	}

	// ENSURE DIRECTORIES EXIST
	os.MkdirAll(AppConfig.StoragePath, 0755)
	os.MkdirAll(AppConfig.ThumbnailsPath, 0755)
	os.MkdirAll(AppConfig.DataPath, 0755)
	os.MkdirAll(AppConfig.LogsPath, 0755)
	os.MkdirAll(AppConfig.ErrorsPath, 0755)

	// CREATE SUBDIRECTORIES FOR BROWSER SCREENSHOTS
	os.MkdirAll(filepath.Join(AppConfig.ErrorsPath, "screenshots"), 0755)
}

// GETRANDOMUSERAGENT RETURNS A RANDOM USER AGENT FROM THE LIST
func GetRandomUserAgent() string {
	return UserAgents[rand.Intn(len(UserAgents))]
}

// LOADCONFIG LOADS CONFIGURATION FROM ENVIRONMENT VARIABLES AND CONFIG FILE
func LoadConfig() {
	// BASIC CONFIG ALREADY INITIALIZED IN INITCONFIG
	// THIS FUNCTION CAN BE EXPANDED TO LOAD FROM ENV VARS OR CONFIG FILES

	// EXAMPLE: OVERRIDE FROM ENVIRONMENT VARIABLES
	if portStr := os.Getenv("CREPES_PORT"); portStr != "" {
		if port, err := parseInt(portStr); err == nil && port > 0 {
			AppConfig.Port = port
		}
	}

	if storagePath := os.Getenv("CREPES_STORAGE_PATH"); storagePath != "" {
		AppConfig.StoragePath = storagePath
	}

	if thumbnailsPath := os.Getenv("CREPES_THUMBNAILS_PATH"); thumbnailsPath != "" {
		AppConfig.ThumbnailsPath = thumbnailsPath
	}

	if dataPath := os.Getenv("CREPES_DATA_PATH"); dataPath != "" {
		AppConfig.DataPath = dataPath
	}

	if logsPath := os.Getenv("CREPES_LOGS_PATH"); logsPath != "" {
		AppConfig.LogsPath = logsPath
	}

	if maxConcurrentStr := os.Getenv("CREPES_MAX_CONCURRENT"); maxConcurrentStr != "" {
		if maxConcurrent, err := parseInt(maxConcurrentStr); err == nil && maxConcurrent > 0 {
			AppConfig.MaxConcurrent = maxConcurrent
		}
	}

	if maxBrowsersStr := os.Getenv("CREPES_MAX_BROWSERS"); maxBrowsersStr != "" {
		if maxBrowsers, err := parseInt(maxBrowsersStr); err == nil && maxBrowsers > 0 {
			AppConfig.MaxBrowsers = maxBrowsers
		}
	}

	if maxBrowserTabsStr := os.Getenv("CREPES_MAX_BROWSER_TABS"); maxBrowserTabsStr != "" {
		if maxBrowserTabs, err := parseInt(maxBrowserTabsStr); err == nil && maxBrowserTabs > 0 {
			AppConfig.MaxBrowserTabs = maxBrowserTabs
		}
	}

	if devModeStr := os.Getenv("CREPES_DEV_MODE"); devModeStr == "true" || devModeStr == "1" {
		AppConfig.DevMode = true
	}

	// ENSURE DIRECTORIES EXIST AFTER CONFIG CHANGES
	os.MkdirAll(AppConfig.StoragePath, 0755)
	os.MkdirAll(AppConfig.ThumbnailsPath, 0755)
	os.MkdirAll(AppConfig.DataPath, 0755)
	os.MkdirAll(AppConfig.LogsPath, 0755)
	os.MkdirAll(AppConfig.ErrorsPath, 0755)
}

// PARSEINT PARSES A STRING TO INT
func parseInt(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
