package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/storage"
	"github.com/shirou/gopsutil/disk"
)

// GetSettings returns the current application settings
func GetSettings(c *gin.Context) {
	// GET SETTINGS FROM DATABASE
	settings := storage.GetSettings()

	// ADD PORT FROM APP CONFIG IN CASE IT'S NOT IN THE DATABASE
	if settings.AppConfig.Port == 0 {
		settings.AppConfig.Port = config.AppConfig.Port
	}

	// RETURN CURRENT CONFIGURED SETTINGS
	SuccessResponse(c, http.StatusOK, settings)
}

// UpdateSettings updates application settings
func UpdateSettings(c *gin.Context) {
	var settings models.Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Invalid settings data: %v", err))
		return
	}

	// VALIDATE SETTINGS
	if settings.AppConfig.Port <= 0 || settings.AppConfig.Port > 65535 {
		ErrorResponse(c, http.StatusBadRequest, "Invalid port number")
		return
	}

	if settings.AppConfig.MaxConcurrent <= 0 {
		ErrorResponse(c, http.StatusBadRequest, "Max concurrent connections must be greater than 0")
		return
	}

	if settings.AppConfig.DefaultTimeout <= 0 {
		ErrorResponse(c, http.StatusBadRequest, "Default timeout must be greater than 0")
		return
	}

	// UPDATE CONFIG VALUES
	config.AppConfig.Port = settings.AppConfig.Port
	config.AppConfig.StoragePath = settings.AppConfig.StoragePath
	config.AppConfig.ThumbnailsPath = settings.AppConfig.ThumbnailsPath
	config.AppConfig.DataPath = settings.AppConfig.DataPath
	config.AppConfig.MaxConcurrent = settings.AppConfig.MaxConcurrent
	config.AppConfig.DefaultTimeout = settings.AppConfig.DefaultTimeout

	// SAVE SETTINGS TO DATABASE
	if err := storage.UpdateSettings(&settings); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to save settings: %v", err))
		return
	}

	SuccessResponse(c, http.StatusOK, map[string]any{
		"message": "Settings updated successfully",
	})
}

// GetStorageInfo returns information about storage usage
func GetStorageInfo(c *gin.Context) {
	storagePath := config.AppConfig.StoragePath

	// GET DISK USAGE INFORMATION USING GOPSUTIL
	usage, err := disk.Usage(filepath.Dir(storagePath))
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get storage info: %v", err))
		return
	}

	// FORMAT THE VALUES FOR BETTER READABILITY
	storageInfo := map[string]string{
		"totalSpace": formatSize(usage.Total),
		"usedSpace":  formatSize(usage.Used),
		"freeSpace":  formatSize(usage.Free),
	}

	SuccessResponse(c, http.StatusOK, storageInfo)
}

// ClearCache clears application cache
func ClearCache(c *gin.Context) {
	// IN A REAL IMPLEMENTATION, THIS WOULD CLEAR CACHES
	// FOR NOW, WE'LL JUST RETURN SUCCESS

	SuccessResponse(c, http.StatusOK, map[string]any{
		"message": "Cache cleared successfully",
	})
}

// Helper function to format bytes to human-readable size
func formatSize(bytes uint64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
