package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/assets"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/scheduler"
	"github.com/nickheyer/Crepes/internal/scraper"
	"github.com/nickheyer/Crepes/internal/storage"
)

func CreateJob(c *gin.Context) {
	var job models.ScrapingJob
	if err := c.ShouldBindJSON(&job); err != nil {
		log.Printf("JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// VALIDATE JOB
	if job.BaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "baseUrl is required"})
		return
	}
	if len(job.Selectors) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one selector is required"})
		return
	}

	// VERIFY ASSET SELECTOR EXISTS
	hasAssetSelector := false
	for _, selector := range job.Selectors {
		if selector.For == "assets" && selector.Value != "" {
			hasAssetSelector = true
			break
		}
	}

	if !hasAssetSelector {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one 'assets' selector is required"})
		return
	}

	// SET DEFAULTS
	job.ID = uuid.New().String()
	job.Status = "idle"
	job.CompletedAssets = make(map[string]bool)
	job.Mutex = &sync.Mutex{}
	job.Assets = []models.Asset{}
	job.CurrentPage = 1
	if job.Rules.UserAgent == "" {
		job.Rules.UserAgent = config.UserAgents[0]
	}

	// HANDLE TIMEOUT AS INT64 OR FLOAT64 FROM JSON
	if job.Rules.Timeout == 0 {
		// CONVERT TIMEOUT FROM SECONDS TO NANOSECONDS IF IT'S A NUMBER VALUE IN JSON
		timeoutValue := c.Request.URL.Query().Get("timeout")
		if timeoutValue != "" {
			if seconds, err := strconv.ParseFloat(timeoutValue, 64); err == nil {
				job.Rules.Timeout = time.Duration(seconds * float64(time.Second))
			}
		}
		// IF STILL ZERO, USE DEFAULT
		if job.Rules.Timeout == 0 {
			job.Rules.Timeout = config.AppConfig.DefaultTimeout
		}
	}

	// LOG THE JOB BEING CREATED FOR DEBUGGING
	jobBytes, _ := json.Marshal(job)
	log.Printf("Creating job: %s", string(jobBytes))

	// STORE JOB
	if err := storage.AddJob(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create job: %v", err)})
		return
	}

	// SCHEDULE JOB IF NEEDED
	if job.Schedule != "" {
		scheduler.ScheduleJob(&job)
	}

	c.JSON(http.StatusCreated, job)
}

func ListJobs(c *gin.Context) {
	storage.JobsMutex.Lock()
	defer storage.JobsMutex.Unlock()

	jobsList := make([]*models.ScrapingJob, 0, len(storage.Jobs))
	for _, job := range storage.Jobs {
		jobsList = append(jobsList, job)
	}

	c.JSON(http.StatusOK, jobsList)
}

func GetJob(c *gin.Context) {
	jobID := c.Param("id")
	job, exists := storage.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func DeleteJob(c *gin.Context) {
	jobID := c.Param("id")
	if exists := storage.DeleteJob(jobID); !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job deleted"})
}

func StartJob(c *gin.Context) {
	jobID := c.Param("id")
	job, exists := storage.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// START JOB ONLY IF NOT RUNNING
	if job.Status == "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job already running"})
		return
	}

	// RESET COMPLETED ASSETS IF JOB WAS PREVIOUSLY RUN
	if job.Status == "completed" || job.Status == "stopped" || job.Status == "failed" {
		job.Mutex.Lock()
		job.CompletedAssets = make(map[string]bool)
		job.CurrentPage = 1
		job.LastError = ""
		job.Mutex.Unlock()
	}

	go scraper.RunJob(job)

	c.JSON(http.StatusOK, gin.H{"message": "job started"})
}

func StopJob(c *gin.Context) {
	jobID := c.Param("id")
	job, exists := storage.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// STOP JOB ONLY IF RUNNING
	job.Mutex.Lock()
	if job.Status != "running" || job.CancelFunc == nil {
		job.Mutex.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "job not running"})
		return
	}

	// CAPTURE THE CANCEL FUNCTION AND NULL IT OUT TO PREVENT RACE CONDITIONS
	cancelFunc := job.CancelFunc
	job.CancelFunc = nil
	job.Status = "stopping" // INTERMEDIATE STATE
	job.Mutex.Unlock()

	// CALL CANCEL FUNCTION OUTSIDE OF LOCK
	cancelFunc()

	// SET FINAL STATUS
	job.Mutex.Lock()
	job.Status = "stopped"
	job.Mutex.Unlock()

	storage.SaveJobs()
	c.JSON(http.StatusOK, gin.H{"message": "job stopped"})
}

func GetJobAssets(c *gin.Context) {
	jobID := c.Param("id")
	job, exists := storage.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job.Assets)
}

func GetAllAssets(c *gin.Context) {
	// GET QUERY PARAMETERS FOR FILTERING
	typeFilter := c.Query("type")
	searchTerm := c.Query("search")
	jobIDFilter := c.Query("jobId")

	storage.JobsMutex.Lock()
	var allAssets []models.Asset
	for jobID, job := range storage.Jobs {
		if jobIDFilter != "" && jobID != jobIDFilter {
			continue
		}

		for _, asset := range job.Assets {
			// ADD JOB ID TO ASSET FOR REFERENCE
			asset.JobID = jobID

			// APPLY FILTERS
			if typeFilter != "" && asset.Type != typeFilter {
				continue
			}

			if searchTerm != "" {
				searchLower := strings.ToLower(searchTerm)
				titleLower := strings.ToLower(asset.Title)
				descLower := strings.ToLower(asset.Description)

				if !strings.Contains(titleLower, searchLower) &&
					!strings.Contains(descLower, searchLower) {
					continue
				}
			}

			allAssets = append(allAssets, asset)
		}
	}
	storage.JobsMutex.Unlock()

	c.JSON(http.StatusOK, allAssets)
}

func GetAsset(c *gin.Context) {
	assetID := c.Param("id")
	var foundAsset *models.Asset
	var foundJob *models.ScrapingJob

	storage.JobsMutex.Lock()
	for _, job := range storage.Jobs {
		for i, asset := range job.Assets {
			if asset.ID == assetID {
				foundAsset = &job.Assets[i] // USE POINTER TO ACTUAL ASSET
				foundJob = job
				break
			}
		}
		if foundAsset != nil {
			break
		}
	}
	storage.JobsMutex.Unlock()

	if foundAsset == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}

	// ADD JOB ID FOR REFERENCE
	assetCopy := *foundAsset
	assetCopy.JobID = foundJob.ID

	c.JSON(http.StatusOK, assetCopy)
}

func DeleteAsset(c *gin.Context) {
	assetID := c.Param("id")
	var foundIndex int = -1
	var foundJob *models.ScrapingJob

	storage.JobsMutex.Lock()
	for _, job := range storage.Jobs {
		for i, asset := range job.Assets {
			if asset.ID == assetID {
				foundIndex = i
				foundJob = job
				break
			}
		}
		if foundIndex >= 0 {
			break
		}
	}

	if foundIndex >= 0 && foundJob != nil {
		// GET THE ASSET TO DELETE ITS FILES
		asset := foundJob.Assets[foundIndex]

		// REMOVE ASSET FROM JOB
		foundJob.Assets = append(foundJob.Assets[:foundIndex], foundJob.Assets[foundIndex+1:]...)

		// RELEASE THE LOCK BEFORE DELETING FILES
		storage.JobsMutex.Unlock()

		// DELETE ASSET FROM DATABASE
		if err := storage.DeleteAsset(assetID); err != nil {
			log.Printf("Error deleting asset from database: %v", err)
		}

		// DELETE ASSET FILES
		if asset.LocalPath != "" {
			fullPath := filepath.Join(config.AppConfig.StoragePath, asset.LocalPath)
			os.Remove(fullPath)
		}

		if asset.ThumbnailPath != "" {
			fullThumbPath := filepath.Join(config.AppConfig.ThumbnailsPath, asset.ThumbnailPath)
			os.Remove(fullThumbPath)
		}

		// UPDATE JOB IN DATABASE
		storage.SaveJobs()

		c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
		return
	}

	storage.JobsMutex.Unlock()
	c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
}

func RegenerateThumbnail(c *gin.Context) {
	assetID := c.Param("id")
	var foundAsset *models.Asset

	storage.JobsMutex.Lock()
	for _, job := range storage.Jobs {
		for i, asset := range job.Assets {
			if asset.ID == assetID {
				foundAsset = &job.Assets[i] // USE POINTER TO ACTUAL ASSET
				break
			}
		}
		if foundAsset != nil {
			break
		}
	}
	storage.JobsMutex.Unlock()

	if foundAsset == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}

	// DELETE EXISTING THUMBNAIL IF IT EXISTS
	if foundAsset.ThumbnailPath != "" {
		fullThumbPath := filepath.Join(config.AppConfig.ThumbnailsPath, foundAsset.ThumbnailPath)
		os.Remove(fullThumbPath)
	}

	// REGENERATE THUMBNAIL
	newThumbPath, err := assets.GenerateThumbnail(foundAsset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to regenerate thumbnail: %v", err)})
		return
	}

	// UPDATE ASSET WITH NEW THUMBNAIL PATH
	storage.JobsMutex.Lock()
	foundAsset.ThumbnailPath = newThumbPath
	storage.JobsMutex.Unlock()

	// SAVE TO DATABASE
	storage.SaveJobs()

	c.JSON(http.StatusOK, gin.H{"message": "thumbnail regenerated", "thumbnailPath": newThumbPath})
}
