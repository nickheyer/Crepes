package api

import (
	"encoding/json"
	"fmt"
	"io"
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

func ErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error":   message,
	})
}

func SuccessResponse(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

func CreateJob(c *gin.Context) {
	// CHECK CONTENT TYPE
	contentType := c.Request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		ErrorResponse(c, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	// READ REQUEST BODY
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// DETERMINE IF THIS IS A PIPELINE OR LEGACY JOB
	var jobData map[string]any
	if err := json.Unmarshal(body, &jobData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	var job models.ScrapingJob
	isPipeline := false

	// CHECK IF THIS IS A PIPELINE JOB
	if _, ok := jobData["pipeline"]; ok {
		isPipeline = true
		// CREATE JOB FROM PIPELINE FORMAT
		if err := json.Unmarshal(body, &job); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline job format"})
			return
		}
	} else {
		// LEGACY FORMAT - BIND TO JOB STRUCT
		if err := json.Unmarshal(body, &job); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job format"})
			return
		}
	}

	// VALIDATE JOB
	if job.BaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "baseUrl is required"})
		return
	}

	// ENSURE JOB HAS AN ID
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	// SET DEFAULTS IF NEEDED
	if job.Status == "" {
		job.Status = "idle"
	}

	if processingConfig, ok := jobData["processing"].(map[string]any); ok {
		// STORE PROCESSING CONFIG IN METADATA
		if job.Metadata == nil {
			job.Metadata = make(map[string]any)
		}
		job.Metadata["processing"] = processingConfig
	}

	// HANDLE FILTERS
	if filters, ok := jobData["filters"].([]any); ok {
		// STORE FILTERS IN METADATA
		if job.Metadata == nil {
			job.Metadata = make(map[string]any)
		}
		job.Metadata["filters"] = filters
	}

	// HANDLE TAGS
	if tags, ok := jobData["tags"].([]any); ok {
		// STORE TAGS IN METADATA
		if job.Metadata == nil {
			job.Metadata = make(map[string]any)
		}
		job.Metadata["tags"] = tags
	}

	// INITIALIZE MUTEX
	job.Mutex = &sync.Mutex{}

	// IF PIPELINE FORMAT, VALIDATE PIPELINE
	if isPipeline {
		// STORE PIPELINE AS JSON STRING
		if _, isPipelineMap := jobData["pipeline"].(map[string]any); isPipelineMap {
			pipelineJSON, err := json.Marshal(jobData["pipeline"])
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline format"})
				return
			}
			job.Pipeline = string(pipelineJSON)
		} else if pipelineStr, isString := jobData["pipeline"].(string); isString {
			// ALREADY A JSON STRING
			job.Pipeline = pipelineStr
		}
	} else {
		// LEGACY FORMAT - INITIALIZE SELECTORS IF NOT SET
		if job.Selectors == nil {
			job.Selectors = []models.SelectorItem{}
		}

		if len(job.Selectors) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one selector is required"})
			return
		}

		// SET DEFAULTS FOR RULES
		if job.Rules.UserAgent == "" {
			job.Rules.UserAgent = config.UserAgents[0]
		}

		if job.Rules.MaxDepth <= 0 {
			job.Rules.MaxDepth = 5
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

		// GENERATE PIPELINE FROM LEGACY JOB
		pipeline := scraper.ConvertJobToPipeline(&job)
		pipelineJSON, err := json.Marshal(pipeline)
		if err != nil {
			log.Printf("Error marshaling pipeline: %v", err)
		} else {
			job.Pipeline = string(pipelineJSON)
		}
	}

	// STORE JOB
	if err := storage.AddJob(&job); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to create job: %v", err))
		return
	}

	// SCHEDULE JOB IF NEEDED
	if job.Schedule != "" {
		scheduler.ScheduleJob(&job)
	}

	SuccessResponse(c, http.StatusCreated, job)
}

func ListJobs(c *gin.Context) {
	// Create a copy of the jobs to avoid holding the mutex while serializing to JSON
	jobsList := make([]*models.ScrapingJob, 0)

	// Critical section - keep mutex locked for as short as possible
	storage.JobsMutex.Lock()
	for _, job := range storage.Jobs {
		jobsList = append(jobsList, job)
	}
	storage.JobsMutex.Unlock()

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

	// Critical section - lock only for status check and initialization
	job.Mutex.Lock()

	// START JOB ONLY IF NOT RUNNING
	if job.Status == "running" {
		job.Mutex.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "job already running"})
		return
	}

	// RESET COMPLETED ASSETS IF JOB WAS PREVIOUSLY RUN
	if job.Status == "completed" || job.Status == "stopped" || job.Status == "failed" {
		job.CompletedAssets = make(map[string]bool)
		job.CurrentPage = 1
		job.LastError = ""
	}

	// Change status to "starting" to avoid race conditions
	job.Status = "starting"
	job.Mutex.Unlock()

	// Start the job in a separate goroutine
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

func GetJobStatistics(c *gin.Context) {
	jobID := c.Param("id")
	job, exists := storage.GetJob(jobID)
	if !exists {
		ErrorResponse(c, http.StatusNotFound, "Job not found")
		return
	}

	// CALCULATE STATISTICS
	stats := map[string]any{
		"totalAssets": len(job.Assets),
		"assetTypes":  map[string]int{},
		"status":      job.Status,
		"lastRun":     job.LastRun,
		"nextRun":     job.NextRun,
		"duration":    "0s",
	}

	// COUNT ASSETS BY TYPE
	typeCount := map[string]int{}
	for _, asset := range job.Assets {
		typeCount[asset.Type]++
	}
	stats["assetTypes"] = typeCount

	// CALCULATE DURATION IF JOB HAS RUN
	if !job.LastRun.IsZero() {
		var endTime time.Time
		if job.Status == "running" {
			endTime = time.Now()
		} else {
			// USE METADATA TO GET END TIME IF AVAILABLE
			if job.Metadata != nil {
				if endTimeStr, ok := job.Metadata["endTime"].(string); ok {
					if parsedTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
						endTime = parsedTime
					}
				}
			}
			// DEFAULT TO NOW IF NO END TIME FOUND
			if endTime.IsZero() {
				endTime = time.Now()
			}
		}
		stats["duration"] = endTime.Sub(job.LastRun).String()
	}

	// ADD SUCCESS/FAILURE COUNTS IF AVAILABLE
	if job.Metadata != nil {
		if succeeded, ok := job.Metadata["succeeded"].(float64); ok {
			stats["succeeded"] = int(succeeded)
		}
		if failed, ok := job.Metadata["failed"].(float64); ok {
			stats["failed"] = int(failed)
		}
		if total, ok := job.Metadata["total"].(float64); ok {
			stats["total"] = int(total)
		}
		if progress, ok := job.Metadata["progress"].(float64); ok {
			stats["progress"] = progress
		}
	}

	SuccessResponse(c, http.StatusOK, stats)
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
	fromDate := c.Query("fromDate")
	toDate := c.Query("toDate")

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

			// APPLY DATE FILTERS IF PRESENT
			if fromDate != "" || toDate != "" {
				if asset.Date == "" {
					continue // SKIP ASSETS WITHOUT DATE IF DATE FILTER APPLIED
				}

				assetDate, err := time.Parse(time.RFC3339, asset.Date)
				if err != nil {
					continue // SKIP IF DATE CANNOT BE PARSED
				}

				if fromDate != "" {
					from, err := time.Parse("2006-01-02", fromDate)
					if err == nil && assetDate.Before(from) {
						continue
					}
				}

				if toDate != "" {
					to, err := time.Parse("2006-01-02", toDate)
					if err == nil {
						// ADD A DAY TO INCLUDE THE END DATE
						to = to.Add(24 * time.Hour)
						if assetDate.After(to) {
							continue
						}
					}
				}
			}

			allAssets = append(allAssets, asset)
		}
	}
	storage.JobsMutex.Unlock()

	SuccessResponse(c, http.StatusOK, allAssets)
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

func UpdateJob(c *gin.Context) {
	id := c.Param("id")

	// Get existing job
	job, exists := storage.GetJob(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Don't update running jobs
	if job.Status == "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot update a running job. Stop the job first."})
		return
	}

	// Bind updated job data
	var updatedJob models.ScrapingJob
	if err := c.ShouldBindJSON(&updatedJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update only allowed fields while keeping the ID and other fields the same
	// Only update baseURL and selectors
	job.BaseURL = updatedJob.BaseURL
	job.Selectors = updatedJob.Selectors
	job.Rules = updatedJob.Rules
	job.Schedule = updatedJob.Schedule

	// Validate the selectors
	hasLinksSelector := false
	hasAssetsSelector := false

	for _, selector := range job.Selectors {
		if selector.Purpose == "links" && !selector.IsOptional {
			hasLinksSelector = true
		}
		if selector.Purpose == "assets" && !selector.IsOptional {
			hasAssetsSelector = true
		}
	}

	if !hasLinksSelector || !hasAssetsSelector {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jobs must have at least one non-optional links selector and one non-optional assets selector"})
		return
	}

	// Save the updated job
	err := storage.UpdateJob(job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job updated successfully", "id": job.ID})
}
