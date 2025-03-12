package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"
	"gorm.io/gorm"
)

func RegisterTemplateHandlers(router *mux.Router, db *gorm.DB) {
	// GET ALL TEMPLATES
	router.HandleFunc("/templates", func(w http.ResponseWriter, r *http.Request) {
		var templates []models.Template
		result := db.Order("created_at DESC").Find(&templates)
		if result.Error != nil {
			log.Printf("Failed to fetch templates: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch templates")
			return
		}
		// FIX EMPTY ARRAYS
		for i := range templates {
			if templates[i].Selectors == nil {
				templates[i].Selectors = []interface{}{}
			}
			if templates[i].Filters == nil {
				templates[i].Filters = []interface{}{}
			}
			if templates[i].Rules == nil {
				templates[i].Rules = map[string]interface{}{}
			}
			if templates[i].Processing == nil {
				templates[i].Processing = map[string]interface{}{
					"thumbnails":    true,
					"metadata":      true,
					"deduplication": true,
				}
			}
			if templates[i].Tags == nil {
				templates[i].Tags = []interface{}{}
			}
		}
		utils.RespondWithJSON(w, http.StatusOK, templates)
	}).Methods("GET")

	// GET TEMPLATE BY ID
	router.HandleFunc("/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		var template models.Template
		result := db.First(&template, "id = ?", id)
		if result.Error != nil {
			log.Printf("Template not found: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Template not found")
			return
		}
		// FIX EMPTY ARRAYS
		if template.Selectors == nil {
			template.Selectors = []interface{}{}
		}
		if template.Filters == nil {
			template.Filters = []interface{}{}
		}
		if template.Rules == nil {
			template.Rules = map[string]interface{}{}
		}
		if template.Processing == nil {
			template.Processing = map[string]interface{}{
				"thumbnails":    true,
				"metadata":      true,
				"deduplication": true,
			}
		}
		if template.Tags == nil {
			template.Tags = []interface{}{}
		}
		utils.RespondWithJSON(w, http.StatusOK, template)
	}).Methods("GET")

	// CREATE TEMPLATE
	router.HandleFunc("/templates", func(w http.ResponseWriter, r *http.Request) {
		var template models.Template
		if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
			log.Printf("Invalid request payload: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		// GENERATE ID IF NOT PROVIDED
		if template.ID == "" {
			template.ID = utils.GenerateID("tpl")
		}
		// SET TIMESTAMPS
		template.CreatedAt = time.Now()
		template.UpdatedAt = time.Now()
		// ENSURE ARRAYS ARE INITIALIZED
		if template.Selectors == nil {
			template.Selectors = []any{}
		}
		if template.Filters == nil {
			template.Filters = []any{}
		}
		if template.Rules == nil {
			template.Rules = map[string]any{}
		}
		if template.Processing == nil {
			template.Processing = map[string]any{
				"thumbnails":    true,
				"metadata":      true,
				"deduplication": true,
			}
		}
		if template.Tags == nil {
			template.Tags = []any{}
		}
		// SAVE TEMPLATE TO DATABASE
		if result := db.Create(&template); result.Error != nil {
			log.Printf("Failed to create template: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create template")
			return
		}
		utils.RespondWithJSON(w, http.StatusCreated, template)
	}).Methods("POST")

	// UPDATE TEMPLATE
	router.HandleFunc("/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		// CHECK IF TEMPLATE EXISTS
		var existingTemplate models.Template
		result := db.First(&existingTemplate, "id = ?", id)
		if result.Error != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Template not found")
			return
		}
		// PARSE UPDATED TEMPLATE DATA
		var updatedTemplate models.Template
		if err := json.NewDecoder(r.Body).Decode(&updatedTemplate); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		// UPDATE FIELDS
		updatedTemplate.ID = id
		updatedTemplate.UpdatedAt = time.Now()
		updatedTemplate.CreatedAt = existingTemplate.CreatedAt
		// SAVE UPDATED TEMPLATE TO DATABASE
		if err := db.Model(&existingTemplate).Updates(updatedTemplate).Error; err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update template")
			return
		}
		utils.RespondWithJSON(w, http.StatusOK, updatedTemplate)
	}).Methods("PUT")

	// DELETE TEMPLATE
	router.HandleFunc("/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		// DELETE TEMPLATE FROM DATABASE
		result := db.Delete(&models.Template{}, "id = ?", id)
		if result.Error != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete template")
			return
		}
		// CHECK IF TEMPLATE WAS FOUND
		if result.RowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "Template not found")
			return
		}
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Template deleted successfully",
		})
	}).Methods("DELETE")

	// CREATE JOB FROM TEMPLATE
	router.HandleFunc("/templates/{id}/create-job", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// GET TEMPLATE
		var template models.Template
		if err := db.First(&template, "id = ?", id).Error; err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "Template not found")
			return
		}

		// CREATE NEW JOB FROM TEMPLATE
		job := models.Job{
			ID:          utils.GenerateID("job"),
			Name:        template.Name + " (from template)",
			BaseURL:     template.BaseURL,
			Description: template.Description,
			Status:      "idle",
			Selectors:   template.Selectors,
			Filters:     template.Filters,
			Rules:       template.Rules,
			Processing:  template.Processing,
			Tags:        template.Tags,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// SAVE JOB TO DATABASE
		if err := db.Create(&job).Error; err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create job from template")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, job)
	}).Methods("POST")
}
