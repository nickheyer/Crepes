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

	// GET DEFAULT TEMPLATE EXAMPLES
	router.HandleFunc("/templates/examples", func(w http.ResponseWriter, r *http.Request) {
		examples := map[string]interface{}{
			"basic_scraper": getBasicScraperTemplate(),
			"image_scraper": getImageScraperTemplate(),
			"pagination":    getPaginationTemplate(),
			"articles":      getArticleScraperTemplate(),
			"e-commerce":    getEcommerceTemplate(),
		}
		utils.RespondWithJSON(w, http.StatusOK, examples)
	}).Methods("GET")

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

// BASIC SCRAPER TEMPLATE
func getBasicScraperTemplate() models.Template {
	return models.Template{
		Name:        "Basic Web Scraper",
		Description: "A simple template for scraping a website. Extracts images and follows links.",
		BaseURL:     "https://example.com",
		Selectors: []interface{}{
			map[string]interface{}{
				"id":              "img-selector",
				"name":            "Images",
				"type":            "image",
				"value":           "img",
				"attributeSource": "src",
				"purpose":         "assets",
				"description":     "Extract all images from the page",
				"priority":        1,
				"isOptional":      false,
			},
			map[string]interface{}{
				"id":              "link-selector",
				"name":            "Links",
				"type":            "link",
				"value":           "a",
				"attributeSource": "href",
				"purpose":         "links",
				"description":     "Follow all links on the page",
				"priority":        1,
				"isOptional":      false,
			},
		},
		Rules: map[string]interface{}{
			"maxDepth":       2,
			"sameDomainOnly": true,
			"requestDelay":   1000,
		},
		Processing: map[string]interface{}{
			"thumbnails":    true,
			"metadata":      true,
			"deduplication": true,
		},
		Tags: []interface{}{"basic", "general"},
	}
}

// IMAGE SCRAPER TEMPLATE
func getImageScraperTemplate() models.Template {
	return models.Template{
		Name:        "Image Scraper",
		Description: "Specialized template for extracting images, optimized for image galleries and photo sites.",
		BaseURL:     "https://example.com",
		Selectors: []interface{}{
			map[string]interface{}{
				"id":              "main-images",
				"name":            "Main Images",
				"type":            "image",
				"value":           "img.main-image, img.gallery-image",
				"attributeSource": "src",
				"purpose":         "assets",
				"description":     "Extract main images from the page",
				"priority":        1,
				"isOptional":      false,
			},
			map[string]interface{}{
				"id":              "lazy-images",
				"name":            "Lazy-loaded Images",
				"type":            "image",
				"value":           "img[data-src]",
				"attributeSource": "data-src",
				"purpose":         "assets",
				"description":     "Extract lazy-loaded images",
				"priority":        2,
				"isOptional":      true,
			},
			map[string]interface{}{
				"id":              "gallery-links",
				"name":            "Gallery Links",
				"type":            "link",
				"value":           "a.gallery-link, a.photo-link",
				"attributeSource": "href",
				"purpose":         "links",
				"description":     "Follow links to other galleries or images",
				"priority":        1,
				"isOptional":      false,
			},
		},
		Rules: map[string]interface{}{
			"maxDepth":       3,
			"maxAssets":      100,
			"sameDomainOnly": true,
			"requestDelay":   500,
		},
		Processing: map[string]interface{}{
			"thumbnails":    true,
			"metadata":      true,
			"deduplication": true,
		},
		Tags: []interface{}{"images", "photos", "gallery"},
	}
}

// PAGINATION TEMPLATE
func getPaginationTemplate() models.Template {
	return models.Template{
		Name:        "Pagination Scraper",
		Description: "Template for scraping sites with pagination. Follows next page links to extract content across multiple pages.",
		BaseURL:     "https://example.com",
		Selectors: []interface{}{
			map[string]interface{}{
				"id":              "content-images",
				"name":            "Content Images",
				"type":            "image",
				"value":           ".content img, .post img",
				"attributeSource": "src",
				"purpose":         "assets",
				"description":     "Extract images from content areas",
				"priority":        1,
				"isOptional":      false,
			},
			map[string]interface{}{
				"id":              "next-page",
				"name":            "Next Page Link",
				"type":            "link",
				"value":           "a.next, .pagination a.next, .pagination a[rel=next]",
				"attributeSource": "href",
				"purpose":         "pagination",
				"description":     "Follow link to next page",
				"priority":        1,
				"isOptional":      false,
			},
		},
		Rules: map[string]interface{}{
			"maxDepth":       10,
			"sameDomainOnly": true,
			"requestDelay":   1000,
		},
		Processing: map[string]interface{}{
			"thumbnails":    true,
			"metadata":      true,
			"deduplication": true,
		},
		Tags: []interface{}{"pagination", "multi-page"},
	}
}

// ARTICLE SCRAPER TEMPLATE
func getArticleScraperTemplate() models.Template {
	return models.Template{
		Name:        "Article Scraper",
		Description: "Template for scraping articles from blogs and news sites.",
		BaseURL:     "https://example.com/blog",
		Selectors: []interface{}{
			map[string]interface{}{
				"id":              "article-links",
				"name":            "Article Links",
				"type":            "link",
				"value":           "article h2 a, .post-title a",
				"attributeSource": "href",
				"purpose":         "links",
				"description":     "Extract links to articles",
				"priority":        1,
				"isOptional":      false,
			},
			map[string]interface{}{
				"id":              "article-images",
				"name":            "Article Images",
				"type":            "image",
				"value":           "article img, .post-content img",
				"attributeSource": "src",
				"purpose":         "assets",
				"description":     "Extract images from articles",
				"priority":        1,
				"isOptional":      false,
			},
			map[string]interface{}{
				"id":              "featured-images",
				"name":            "Featured Images",
				"type":            "image",
				"value":           ".featured-image img, .post-thumbnail img",
				"attributeSource": "src",
				"purpose":         "assets",
				"description":     "Extract featured images",
				"priority":        2,
				"isOptional":      true,
			},
			map[string]interface{}{
				"id":              "next-page",
				"name":            "Next Page Link",
				"type":            "link",
				"value":           ".pagination a.next, a[rel=next]",
				"attributeSource": "href",
				"purpose":         "pagination",
				"description":     "Follow link to next page of articles",
				"priority":        1,
				"isOptional":      false,
			},
		},
		Rules: map[string]interface{}{
			"maxDepth":          5,
			"sameDomainOnly":    true,
			"requestDelay":      1000,
			"includeUrlPattern": "/blog/|/article/|/post/",
		},
		Processing: map[string]interface{}{
			"thumbnails":    true,
			"metadata":      true,
			"deduplication": true,
		},
		Tags: []interface{}{"blog", "articles", "news"},
	}
}

// E-COMMERCE TEMPLATE
func getEcommerceTemplate() models.Template {
	return models.Template{
		Name:        "E-Commerce Scraper",
		Description: "Template for scraping product information from e-commerce websites.",
		BaseURL:     "https://example.com/shop",
		Selectors: []interface{}{
			map[string]interface{}{
				"id":              "product-links",
				"name":            "Product Links",
				"type":            "link",
				"value":           ".product a, .product-title a",
				"attributeSource": "href",
				"purpose":         "links",
				"description":     "Extract links to product pages",
				"priority":        1,
				"isOptional":      false,
			},
			map[string]interface{}{
				"id":              "product-images",
				"name":            "Product Images",
				"type":            "image",
				"value":           ".product-image img, .product-gallery img",
				"attributeSource": "src",
				"purpose":         "assets",
				"description":     "Extract product images",
				"priority":        1,
				"isOptional":      false,
			},
			map[string]interface{}{
				"id":              "category-links",
				"name":            "Category Links",
				"type":            "link",
				"value":           ".category a, .category-menu a",
				"attributeSource": "href",
				"purpose":         "links",
				"description":     "Follow links to product categories",
				"priority":        2,
				"isOptional":      true,
			},
			map[string]interface{}{
				"id":              "next-page",
				"name":            "Next Page Link",
				"type":            "link",
				"value":           ".pagination a.next, .pages a.next",
				"attributeSource": "href",
				"purpose":         "pagination",
				"description":     "Follow link to next page of products",
				"priority":        1,
				"isOptional":      false,
			},
		},
		Rules: map[string]interface{}{
			"maxDepth":          5,
			"sameDomainOnly":    true,
			"requestDelay":      1000,
			"includeUrlPattern": "/product/|/item/|/shop/",
		},
		Processing: map[string]interface{}{
			"thumbnails":    true,
			"metadata":      true,
			"deduplication": true,
		},
		Tags: []interface{}{"ecommerce", "products", "shop"},
	}
}
