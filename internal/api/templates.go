package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/storage"
)

// CREATE TEMPLATE FROM JOB OR TEMPLATE DATA
func CreateTemplate(c *gin.Context) {
	var templateData models.JobTemplate
	if err := c.ShouldBindJSON(&templateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid template data: %v", err)})
		return
	}

	// ENSURE TEMPLATE HAS AN ID
	if templateData.ID == "" {
		templateData.ID = uuid.New().String()
	}

	// STORE TEMPLATE
	if err := storage.AddTemplate(&templateData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create template: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, templateData)
}

// LIST TEMPLATES
func ListTemplates(c *gin.Context) {
	templates, err := storage.GetTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch templates: %v", err)})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GET TEMPLATE BY ID
func GetTemplate(c *gin.Context) {
	templateID := c.Param("id")
	template, exists := storage.GetTemplate(templateID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// UPDATE TEMPLATE
func UpdateTemplate(c *gin.Context) {
	templateID := c.Param("id")
	_, exists := storage.GetTemplate(templateID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	var updatedTemplate models.JobTemplate
	if err := c.ShouldBindJSON(&updatedTemplate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid template data: %v", err)})
		return
	}

	// MAINTAIN SAME ID
	updatedTemplate.ID = templateID

	// UPDATE TEMPLATE
	if err := storage.UpdateTemplate(&updatedTemplate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update template: %v", err)})
		return
	}

	c.JSON(http.StatusOK, updatedTemplate)
}

// DELETE TEMPLATE
func DeleteTemplate(c *gin.Context) {
	templateID := c.Param("id")
	if exists := storage.DeleteTemplate(templateID); !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted"})
}

// CREATE JOB FROM TEMPLATE
func CreateJobFromTemplate(c *gin.Context) {
	templateID := c.Param("id")
	template, exists := storage.GetTemplate(templateID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// CREATE JOB FROM TEMPLATE DATA
	job := models.ScrapingJob{
		ID:              uuid.New().String(),
		Name:            template.Name,
		BaseURL:         template.BaseURL,
		Selectors:       template.Selectors,
		Rules:           template.Rules,
		Schedule:        template.Schedule,
		Status:          "idle",
		CompletedAssets: make(map[string]bool),
	}

	// STORE JOB
	if err := storage.AddJob(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create job from template: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, job)
}
