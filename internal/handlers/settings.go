package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// REGISTER SETTINGS HANDLERS
func RegisterSettingsHandlers(router *mux.Router, db *gorm.DB, cfg *config.Config) {
	// GET ALL SETTINGS
	router.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		var settings []models.Setting
		if err := db.Find(&settings).Error; err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch settings")
			return
		}

		// CONVERT SETTINGS TO MAP
		settingsMap := make(map[string]string)
		for _, setting := range settings {
			settingsMap[setting.Key] = setting.Value
		}

		// CREATE RESPONSE STRUCTURE
		response := map[string]interface{}{
			"appConfig": map[string]interface{}{
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

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    response,
		})
	}).Methods("GET")

	// UPDATE SETTINGS
	router.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		// PARSE REQUEST BODY
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// UPDATE APP CONFIG
		if appConfig, ok := request["appConfig"].(map[string]interface{}); ok {
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

			// SAVE CONFIG TO FILE
			if err := config.SaveConfig(cfg, "config.json"); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save app configuration")
				return
			}
		}

		// UPDATE USER CONFIG
		if userConfig, ok := request["userConfig"].(map[string]interface{}); ok {
			// UPDATE DATABASE SETTINGS
			for key, value := range userConfig {
				strValue, ok := value.(string)
				if !ok {
					// CONVERT NON-STRING VALUES TO STRING
					valueJSON, _ := json.Marshal(value)
					strValue = string(valueJSON)
				}

				// UPSERT SETTING
				var setting models.Setting
				if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
					// CREATE NEW SETTING
					setting = models.Setting{
						Key:   key,
						Value: strValue,
					}
					db.Create(&setting)
				} else {
					// UPDATE EXISTING SETTING
					setting.Value = strValue
					db.Save(&setting)
				}
			}
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Settings updated successfully",
		})
	}).Methods("PUT")

	// CLEAR CACHE
	router.HandleFunc("/cache/clear", func(w http.ResponseWriter, r *http.Request) {
		// THIS IS A PLACEHOLDER FOR CACHE CLEARING FUNCTIONALITY
		// IN A REAL APPLICATION, THIS WOULD CLEAR TEMPORARY FILES, ETC.

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Cache cleared successfully",
		})
	}).Methods("POST")
}
