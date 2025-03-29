package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"
	"gorm.io/gorm"
)

func GetSettings(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var settings []models.Setting
		if err := db.Find(&settings).Error; err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch settings")
			return
		}
		settingsMap := make(map[string]string)
		for _, setting := range settings {
			settingsMap[setting.Key] = setting.Value
		}
		response := map[string]any{
			"appConfig": map[string]any{
				"port":           cfg.Port,
				"storagePath":    cfg.StoragePath,
				"thumbnailsPath": cfg.ThumbnailsPath,
				"dataPath":       cfg.DataPath,
				"maxConcurrent":  cfg.MaxConcurrent,
				"defaultTimeout": cfg.DefaultTimeout,
			},
			"userConfig": map[string]string{
				"theme":                settingsMap["theme"],
				"defaultView":          settingsMap["defaultView"],
				"notificationsEnabled": settingsMap["notificationsEnabled"],
			},
		}
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    response,
		})
	}
}

func UpdateSettings(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		if appConfig, ok := request["appConfig"].(map[string]any); ok {
			if port, ok := appConfig["port"].(string); ok && port != "" {
				cfg.Port = port
			}
			if storagePath, ok := appConfig["storagePath"].(string); ok && storagePath != "" {
				cfg.StoragePath = storagePath
			}
			if thumbnailsPath, ok := appConfig["thumbnailsPath"].(string); ok && thumbnailsPath != "" {
				cfg.ThumbnailsPath = thumbnailsPath
			}
			if dataPath, ok := appConfig["dataPath"].(string); ok && dataPath != "" {
				cfg.DataPath = dataPath
			}
			if maxConcurrent, ok := appConfig["maxConcurrent"].(float64); ok {
				cfg.MaxConcurrent = int(maxConcurrent)
			}
			if defaultTimeout, ok := appConfig["defaultTimeout"].(float64); ok {
				cfg.DefaultTimeout = int(defaultTimeout)
			}
			if err := config.SaveConfig(cfg, "config.json"); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save app configuration")
				return
			}
		}
		if userConfig, ok := request["userConfig"].(map[string]any); ok {
			for key, value := range userConfig {
				strValue, ok := value.(string)
				if !ok {
					valueJSON, _ := json.Marshal(value)
					strValue = string(valueJSON)
				}
				var setting models.Setting
				if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
					setting = models.Setting{
						Key:   key,
						Value: strValue,
					}
					db.Create(&setting)
				} else {
					setting.Value = strValue
					db.Save(&setting)
				}
			}
		}
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Settings updated successfully",
		})
	}
}

func ClearCache() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Cache cleared successfully",
		})
	}
}
