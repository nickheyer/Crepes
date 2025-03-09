package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// REGISTER ASSET HANDLERS
func RegisterAssetHandlers(router *mux.Router, db *gorm.DB, cfg *config.Config) {
	// GET ALL ASSETS WITH OPTIONAL FILTERS
	router.HandleFunc("/assets", func(w http.ResponseWriter, r *http.Request) {
		query := db.Model(&models.Asset{})

		// APPLY TYPE FILTER
		if assetType := r.URL.Query().Get("type"); assetType != "" {
			query = query.Where("type = ?", assetType)
		}

		// APPLY JOB FILTER
		if jobId := r.URL.Query().Get("jobId"); jobId != "" {
			query = query.Where("job_id = ?", jobId)
		}

		// APPLY SEARCH FILTER
		if search := r.URL.Query().Get("search"); search != "" {
			searchTerm := "%" + search + "%"
			query = query.Where("title LIKE ? OR description LIKE ? OR url LIKE ?", searchTerm, searchTerm, searchTerm)
		}

		// APPLY DATE RANGE FILTER
		if fromDate := r.URL.Query().Get("from"); fromDate != "" {
			query = query.Where("date >= ?", fromDate)
		}
		if toDate := r.URL.Query().Get("to"); toDate != "" {
			query = query.Where("date <= ?", toDate)
		}

		// APPLY SORTING
		sortBy := r.URL.Query().Get("sortBy")
		sortDirection := r.URL.Query().Get("sortDirection")
		if sortBy != "" {
			if sortDirection == "asc" {
				query = query.Order(sortBy)
			} else {
				query = query.Order(sortBy + " DESC")
			}
		} else {
			// DEFAULT SORT BY CREATED_AT DESC
			query = query.Order("created_at DESC")
		}

		var assets []models.Asset
		result := query.Find(&assets)
		if result.Error != nil {
			log.Printf("Failed to fetch assets: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch assets")
			return
		}

		// FIX EMPTY METADATA
		for i := range assets {
			if assets[i].Metadata == nil {
				assets[i].Metadata = map[string]interface{}{}
			}
		}

		// GET ASSET COUNTS FOR FILTER PANEL
		var counts struct {
			Image    int64 `json:"image"`
			Video    int64 `json:"video"`
			Audio    int64 `json:"audio"`
			Document int64 `json:"document"`
			Total    int64 `json:"total"`
		}

		db.Model(&models.Asset{}).Count(&counts.Total)
		db.Model(&models.Asset{}).Where("type = ?", "image").Count(&counts.Image)
		db.Model(&models.Asset{}).Where("type = ?", "video").Count(&counts.Video)
		db.Model(&models.Asset{}).Where("type = ?", "audio").Count(&counts.Audio)
		db.Model(&models.Asset{}).Where("type = ?", "document").Count(&counts.Document)

		// INCLUDE COUNTS IN RESPONSE
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"assets": assets,
			"counts": counts,
		})
	}).Methods("GET")

	// GET ASSET BY ID
	router.HandleFunc("/assets/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		var asset models.Asset
		result := db.First(&asset, "id = ?", id)
		if result.Error != nil {
			log.Printf("Asset not found: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Asset not found")
			return
		}

		// FIX EMPTY METADATA
		if asset.Metadata == nil {
			asset.Metadata = map[string]interface{}{}
		}

		utils.RespondWithJSON(w, http.StatusOK, asset)
	}).Methods("GET")

	// DELETE ASSET
	router.HandleFunc("/assets/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// GET ASSET FOR FILE PATH INFO
		var asset models.Asset
		result := db.First(&asset, "id = ?", id)
		if result.Error != nil {
			log.Printf("Asset not found for deletion: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Asset not found")
			return
		}

		// DELETE ASSET FILES
		if asset.LocalPath != "" {
			filePath := filepath.Join(cfg.StoragePath, asset.LocalPath)
			if err := os.Remove(filePath); err != nil {
				log.Printf("Warning: failed to delete asset file: %v", err)
			}
		}

		if asset.ThumbnailPath != "" {
			thumbPath := filepath.Join(cfg.ThumbnailsPath, asset.ThumbnailPath)
			if err := os.Remove(thumbPath); err != nil {
				log.Printf("Warning: failed to delete thumbnail file: %v", err)
			}
		}

		// DELETE FROM DATABASE
		if err := db.Delete(&asset).Error; err != nil {
			log.Printf("Failed to delete asset from DB: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete asset")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Asset deleted successfully",
		})
	}).Methods("DELETE")

	// REGENERATE THUMBNAIL
	router.HandleFunc("/assets/{id}/regenerate-thumbnail", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// GET ASSET
		var asset models.Asset
		result := db.First(&asset, "id = ?", id)
		if result.Error != nil {
			log.Printf("Asset not found for thumbnail regeneration: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Asset not found")
			return
		}

		// CHECK IF ASSET HAS A LOCAL FILE
		if asset.LocalPath == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Asset does not have a local file")
			return
		}

		// CHECK IF FILE EXISTS
		filePath := filepath.Join(cfg.StoragePath, asset.LocalPath)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("Asset file not found for thumbnail regeneration: %v", err)
			utils.RespondWithError(w, http.StatusNotFound, "Asset file not found")
			return
		}

		// GENERATE NEW THUMBNAIL FILENAME
		thumbnailFilename := "thumb_" + id + "_" + strconv.FormatInt(asset.UpdatedAt.Unix(), 10) + ".jpg"
		thumbnailPath := filepath.Join(cfg.ThumbnailsPath, thumbnailFilename)

		// DELETE OLD THUMBNAIL IF EXISTS
		if asset.ThumbnailPath != "" {
			oldThumbPath := filepath.Join(cfg.ThumbnailsPath, asset.ThumbnailPath)
			if err := os.Remove(oldThumbPath); err != nil {
				log.Printf("Warning: failed to delete old thumbnail: %v", err)
			}
		}

		// GENERATE THUMBNAIL BASED ON ASSET TYPE
		var err error
		switch {
		case strings.HasPrefix(asset.Type, "image"):
			err = utils.GenerateImageThumbnail(filePath, thumbnailPath)
		case strings.HasPrefix(asset.Type, "video"):
			err = utils.GenerateVideoThumbnail(filePath, thumbnailPath)
		case strings.HasPrefix(asset.Type, "audio"):
			err = utils.GenerateAudioThumbnail(thumbnailPath) // GENERIC AUDIO ICON
		case strings.HasPrefix(asset.Type, "document") || strings.HasPrefix(asset.Type, "application"):
			err = utils.GenerateDocumentThumbnail(thumbnailPath) // GENERIC DOCUMENT ICON
		default:
			err = utils.GenerateGenericThumbnail(thumbnailPath) // GENERIC ICON
		}

		if err != nil {
			log.Printf("Failed to generate thumbnail: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate thumbnail: "+err.Error())
			return
		}

		// UPDATE ASSET RECORD
		asset.ThumbnailPath = thumbnailFilename
		if err := db.Save(&asset).Error; err != nil {
			log.Printf("Failed to update asset with new thumbnail: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update asset")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success":       true,
			"message":       "Thumbnail regenerated successfully",
			"thumbnailPath": thumbnailFilename,
		})
	}).Methods("POST")

	// GET ASSET COUNTS BY TYPE (FOR FILTERS)
	router.HandleFunc("/assets/counts", func(w http.ResponseWriter, r *http.Request) {
		var counts struct {
			Image    int64 `json:"image"`
			Video    int64 `json:"video"`
			Audio    int64 `json:"audio"`
			Document int64 `json:"document"`
			Total    int64 `json:"total"`
		}

		db.Model(&models.Asset{}).Count(&counts.Total)
		db.Model(&models.Asset{}).Where("type = ?", "image").Count(&counts.Image)
		db.Model(&models.Asset{}).Where("type = ?", "video").Count(&counts.Video)
		db.Model(&models.Asset{}).Where("type = ?", "audio").Count(&counts.Audio)
		db.Model(&models.Asset{}).Where("type = ?", "document").Count(&counts.Document)

		utils.RespondWithJSON(w, http.StatusOK, counts)
	}).Methods("GET")

	// SERVE ASSET FILES
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/api/assets/", http.FileServer(http.Dir(cfg.StoragePath))))

	// SERVE THUMBNAIL FILES
	router.PathPrefix("/thumbnails/").Handler(http.StripPrefix("/api/thumbnails/", http.FileServer(http.Dir(cfg.ThumbnailsPath))))
}
