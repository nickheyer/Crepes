package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"

	fcmp "github.com/MontFerret/ferret/pkg/compiler"
	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	fhttp "github.com/MontFerret/ferret/pkg/drivers/http"
	"gorm.io/gorm"
)

// SCRAPER ENGINE
type Engine struct {
	db                *gorm.DB
	config            *config.Config
	runningJobs       map[string]context.CancelFunc
	jobProgress       map[string]int
	jobDurations      map[string]time.Duration
	jobStartTimes     map[string]time.Time
	mu                sync.Mutex
	ferretCompiler    *fcmp.Compiler
	activeDrivers     map[string]drivers.Driver
	downloadSemaphore chan struct{} // LIMIT CONCURRENT DOWNLOADS
}

// JOB RESULT FOR ASSETS
type JobResult struct {
	URL         string
	Type        string
	Title       string
	Description string
	Metadata    map[string]interface{}
}

// CREATE NEW SCRAPER ENGINE
func NewEngine(db *gorm.DB, config *config.Config) *Engine {
	// SET UP FERRET COMPILER
	compiler := fcmp.New()

	return &Engine{
		db:                db,
		config:            config,
		runningJobs:       make(map[string]context.CancelFunc),
		jobProgress:       make(map[string]int),
		jobDurations:      make(map[string]time.Duration),
		jobStartTimes:     make(map[string]time.Time),
		mu:                sync.Mutex{},
		ferretCompiler:    compiler,
		activeDrivers:     make(map[string]drivers.Driver),
		downloadSemaphore: make(chan struct{}, config.MaxConcurrent),
	}
}

// RUN A JOB
func (e *Engine) RunJob(jobID string) error {
	log.Printf("Starting job: %s", jobID)

	// CHECK IF JOB IS ALREADY RUNNING
	e.mu.Lock()
	if _, exists := e.runningJobs[jobID]; exists {
		e.mu.Unlock()
		return errors.New("job is already running")
	}
	e.mu.Unlock()

	// GET JOB FROM DATABASE
	var job models.Job
	if err := e.db.First(&job, "id = ?", jobID).Error; err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// CREATE CANCELLABLE CONTEXT
	ctx, cancel := context.WithCancel(context.Background())

	// UPDATE JOB STATUS TO RUNNING
	job.Status = "running"
	job.LastRun = time.Now()
	e.db.Save(&job)

	// REGISTER JOB AS RUNNING
	e.mu.Lock()
	e.runningJobs[jobID] = cancel
	e.jobProgress[jobID] = 0
	e.jobStartTimes[jobID] = time.Now()
	e.mu.Unlock()

	// RUN JOB IN GOROUTINE
	go func() {
		// CLEANUP WHEN DONE
		defer func() {
			// REMOVE FROM RUNNING JOBS
			e.mu.Lock()
			delete(e.runningJobs, jobID)
			duration := time.Since(e.jobStartTimes[jobID])
			e.jobDurations[jobID] = duration
			e.mu.Unlock()

			// UPDATE JOB STATUS BASED ON CONTEXT CANCELLATION
			var newStatus string
			if ctx.Err() == context.Canceled {
				newStatus = "stopped"
			} else {
				newStatus = "completed"
			}

			if err := e.db.Model(&models.Job{}).Where("id = ?", jobID).Updates(map[string]interface{}{
				"status": newStatus,
			}).Error; err != nil {
				log.Printf("Failed to update job status: %v", err)
			}

			log.Printf("Job %s finished with status: %s, duration: %v", jobID, newStatus, duration)
		}()

		// RUN THE SCRAPER BASED ON JOB TYPE
		if job.Template != "" {
			// RUN USING CUSTOM FERRET TEMPLATE
			err := e.runFerretTemplate(ctx, &job)
			if err != nil && ctx.Err() != context.Canceled {
				log.Printf("Job %s failed: %v", jobID, err)
				e.db.Model(&models.Job{}).Where("id = ?", jobID).Updates(map[string]interface{}{
					"status": "failed",
				})
			}
		} else {
			// FALLBACK TO BASIC SCRAPING USING JOB CONFIG
			err := e.runBasicScraper(ctx, &job)
			if err != nil && ctx.Err() != context.Canceled {
				log.Printf("Job %s failed: %v", jobID, err)
				e.db.Model(&models.Job{}).Where("id = ?", jobID).Updates(map[string]interface{}{
					"status": "failed",
				})
			}
		}
	}()

	return nil
}

// STOP A RUNNING JOB
func (e *Engine) StopJob(jobID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	cancel, exists := e.runningJobs[jobID]
	if !exists {
		return errors.New("job is not running")
	}

	// CANCEL THE JOB
	cancel()
	return nil
}

// GET JOB PROGRESS
func (e *Engine) GetJobProgress(jobID string) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	progress, exists := e.jobProgress[jobID]
	if !exists {
		return 0
	}
	return progress
}

// UPDATE JOB PROGRESS
func (e *Engine) updateJobProgress(jobID string, progress int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.jobProgress[jobID] = progress
}

// GET JOB DURATION
func (e *Engine) GetJobDuration(jobID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	// CHECK IF JOB IS RUNNING
	if startTime, running := e.jobStartTimes[jobID]; running {
		return utils.FormatDuration(time.Since(startTime))
	}

	// RETURN STORED DURATION FOR COMPLETED JOBS
	if duration, exists := e.jobDurations[jobID]; exists {
		return utils.FormatDuration(duration)
	}

	return "0s"
}

// RUN FERRET TEMPLATE FOR A JOB
func (e *Engine) runFerretTemplate(ctx context.Context, job *models.Job) error {
	log.Printf("Running job %s using Ferret template", job.ID)

	// COMPILE FERRET QUERY
	program, err := e.ferretCompiler.Compile(job.Template)
	if err != nil {
		return fmt.Errorf("failed to compile Ferret template: %w", err)
	}

	// SET UP DRIVERS
	httpDriver := fhttp.NewDriver()
	defer httpDriver.Close()

	cdpDriver := cdp.NewDriver()
	defer cdpDriver.Close()

	// CREATE EXECUTION CONTEXT
	execCtx := drivers.WithContext(ctx, httpDriver)
	execCtx = drivers.WithContext(execCtx, cdpDriver)

	// EXECUTE FERRET QUERY
	out, err := program.Run(execCtx)
	if err != nil {
		return fmt.Errorf("failed to execute Ferret template: %w", err)
	}

	// PARSE RESULTS
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		return fmt.Errorf("failed to parse Ferret results: %w", err)
	}

	// PROCESS RESULTS
	log.Printf("Job %s found %d items", job.ID, len(results))
	totalItems := len(results)
	processedItems := 0

	// CHECK IF MAX ASSETS IS SET AND VALID
	maxAssets := 0
	if max, ok := job.Rules["maxAssets"].(float64); ok && max > 0 {
		maxAssets = int(max)
	}

	// PROCESS RESULTS
	for _, result := range results {
		// CHECK FOR CANCELLATION
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// CHECK IF MAX ASSETS LIMIT REACHED
		if maxAssets > 0 && processedItems >= maxAssets {
			log.Printf("Job %s reached max assets limit (%d)", job.ID, maxAssets)
			break
		}

		// GET URL FROM RESULT
		url, ok := result["url"].(string)
		if !ok || url == "" {
			continue
		}

		// TRANSFORM RELATIVE URL TO ABSOLUTE
		if !strings.HasPrefix(url, "http") {
			baseURL := job.BaseURL
			url = utils.ResolveURL(baseURL, url)
		}

		// CREATE ASSET FROM RESULT
		asset := models.Asset{
			ID:        utils.GenerateID("ast"),
			JobID:     job.ID,
			URL:       url,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Date:      time.Now(),
		}

		// SET ASSET METADATA
		metadata := make(map[string]interface{})
		for k, v := range result {
			if k != "url" {
				metadata[k] = v
			}
		}

		// SET ASSET METADATA
		asset.Metadata = metadata

		// SET TITLE IF AVAILABLE
		if title, ok := result["title"].(string); ok {
			asset.Title = title
		}

		// SET DESCRIPTION IF AVAILABLE
		if desc, ok := result["description"].(string); ok {
			asset.Description = desc
		}

		// DOWNLOAD ASSET
		if e.shouldDownloadAsset(url, job) {
			e.downloadSemaphore <- struct{}{} // ACQUIRE SEMAPHORE
			go func(asset models.Asset) {
				defer func() { <-e.downloadSemaphore }() // RELEASE SEMAPHORE

				// DOWNLOAD THE ASSET
				if err := e.downloadAsset(ctx, &asset, job); err != nil {
					log.Printf("Failed to download asset %s: %v", asset.URL, err)
				}

				// SAVE ASSET TO DATABASE
				e.db.Create(&asset)
			}(asset)
		} else {
			// SAVE ASSET TO DATABASE WITHOUT DOWNLOADING
			e.db.Create(&asset)
		}

		// UPDATE PROGRESS
		processedItems++
		if totalItems > 0 {
			progress := (processedItems * 100) / totalItems
			e.updateJobProgress(job.ID, progress)
		}
	}

	return nil
}

// RUN BASIC SCRAPER USING JOB CONFIG
func (e *Engine) runBasicScraper(ctx context.Context, job *models.Job) error {
	log.Printf("Running job %s using basic scraper", job.ID)

	// GENERATE FERRET TEMPLATE FROM JOB CONFIG
	template := GenerateFerretTemplate(job)

	// UPDATE JOB WITH GENERATED TEMPLATE
	e.db.Model(job).Update("template", template)

	// RUN THE GENERATED TEMPLATE
	return e.runFerretTemplate(ctx, job)
}

// CHECK IF ASSET SHOULD BE DOWNLOADED
func (e *Engine) shouldDownloadAsset(url string, job *models.Job) bool {
	// CHECK FILE EXTENSION/MIME TYPE
	ext := strings.ToLower(filepath.Ext(url))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff":
		return true
	case ".mp4", ".webm", ".avi", ".mov", ".mkv", ".flv":
		return true
	case ".mp3", ".wav", ".ogg", ".flac", ".aac":
		return true
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt":
		return true
	}

	// CHECK INCLUDE PATTERN
	if includePattern, ok := job.Rules["includeUrlPattern"].(string); ok && includePattern != "" {
		matched, err := regexp.MatchString(includePattern, url)
		if err != nil || !matched {
			return false
		}
	}

	// CHECK EXCLUDE PATTERN
	if excludePattern, ok := job.Rules["excludeUrlPattern"].(string); ok && excludePattern != "" {
		matched, err := regexp.MatchString(excludePattern, url)
		if err == nil && matched {
			return false
		}
	}

	// APPLY CUSTOM FILTERS
	for _, filterObj := range job.Filters {
		if filter, ok := filterObj.(map[string]interface{}); ok {
			pattern, hasPattern := filter["pattern"].(string)
			action, hasAction := filter["action"].(string)

			if hasPattern && hasAction && pattern != "" {
				matched, err := regexp.MatchString(pattern, url)
				if err == nil && matched {
					return action == "include"
				}
			}
		}
	}

	// DEFAULT BEHAVIOR FOR UNKNOWN TYPES
	return false
}

// DOWNLOAD AN ASSET
func (e *Engine) downloadAsset(ctx context.Context, asset *models.Asset, job *models.Job) error {
	// CREATE HTTP CLIENT WITH TIMEOUT
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// CREATE REQUEST
	req, err := http.NewRequestWithContext(ctx, "GET", asset.URL, nil)
	if err != nil {
		return err
	}

	// ADD COMMON HEADERS
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// SEND REQUEST
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// CHECK RESPONSE CODE
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// GET CONTENT TYPE
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// TRY TO DETECT TYPE FROM URL
		contentType = mime.TypeByExtension(filepath.Ext(asset.URL))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// SET ASSET TYPE BASED ON CONTENT TYPE
	assetType := "unknown"
	if strings.HasPrefix(contentType, "image/") {
		assetType = "image"
	} else if strings.HasPrefix(contentType, "video/") {
		assetType = "video"
	} else if strings.HasPrefix(contentType, "audio/") {
		assetType = "audio"
	} else if strings.HasPrefix(contentType, "application/pdf") ||
		strings.HasPrefix(contentType, "application/msword") ||
		strings.HasPrefix(contentType, "application/vnd.ms") ||
		strings.HasPrefix(contentType, "text/") {
		assetType = "document"
	}
	asset.Type = assetType

	// GENERATE FILENAME
	filename := utils.GenerateFilename(asset.URL, contentType)
	localPath := filepath.Join(job.ID, filename)
	fullPath := filepath.Join(e.config.StoragePath, localPath)

	// CREATE DIRECTORY IF IT DOESN'T EXIST
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// DEDUPLICATION CHECK IF ENABLED
	shouldProcess := true
	if dedup, ok := job.Processing["deduplication"].(bool); ok && dedup {
		// CHECK IF FILE WITH SAME URL ALREADY EXISTS
		var count int64
		e.db.Model(&models.Asset{}).Where("url = ? AND job_id = ?", asset.URL, job.ID).Count(&count)
		if count > 0 {
			log.Printf("Skipping duplicate asset: %s", asset.URL)
			shouldProcess = false
			return nil
		}
	}

	if shouldProcess {
		// CREATE FILE
		file, err := os.Create(fullPath)
		if err != nil {
			return err
		}
		defer file.Close()

		// DOWNLOAD FILE
		written, err := io.Copy(file, resp.Body)
		if err != nil {
			return err
		}

		// UPDATE ASSET PROPERTIES
		asset.LocalPath = localPath
		asset.Size = written

		// GENERATE THUMBNAIL IF ENABLED
		if thumbnails, ok := job.Processing["thumbnails"].(bool); ok && thumbnails {
			thumbnailFilename := fmt.Sprintf("thumb_%s.jpg", asset.ID)
			thumbnailPath := filepath.Join(e.config.ThumbnailsPath, thumbnailFilename)

			// GENERATE THUMBNAIL BASED ON ASSET TYPE
			var thumbErr error
			switch assetType {
			case "image":
				thumbErr = utils.GenerateImageThumbnail(fullPath, thumbnailPath)
			case "video":
				thumbErr = utils.GenerateVideoThumbnail(fullPath, thumbnailPath)
			case "audio":
				thumbErr = utils.GenerateAudioThumbnail(thumbnailPath) // GENERIC AUDIO ICON
			case "document":
				thumbErr = utils.GenerateDocumentThumbnail(thumbnailPath) // GENERIC DOCUMENT ICON
			default:
				thumbErr = utils.GenerateGenericThumbnail(thumbnailPath) // GENERIC ICON
			}

			if thumbErr == nil {
				asset.ThumbnailPath = thumbnailFilename
			} else {
				log.Printf("Failed to generate thumbnail for %s: %v", asset.URL, thumbErr)
			}
		}
	}

	return nil
}
