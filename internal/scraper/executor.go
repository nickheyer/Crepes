package scraper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nickheyer/Crepes/internal/assets"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/storage"
	"github.com/nickheyer/Crepes/internal/utils"
)

// JOBEXECUTOR MANAGES EXECUTION OF SCRAPING JOBS
type JobExecutor struct {
	// JOB CONFIGURATION
	job        *models.ScrapingJob
	pipeline   *Pipeline
	jobCtx     context.Context
	cancelFunc context.CancelFunc

	// EXECUTION STATE
	status        string
	startTime     time.Time
	endTime       time.Time
	processedURLs sync.Map
	completedURLs int32
	failedURLs    int32
	totalURLs     int32
	errors        []utils.ScraperError
	mu            sync.RWMutex

	// CONCURRENCY MANAGEMENT
	workerPool *utils.WorkerPool
	errorGroup *utils.ErrorGroup
	logger     *utils.Logger
}

// NEWJOBEXECUTOR CREATES A NEW JOB EXECUTOR
func NewJobExecutor(job *models.ScrapingJob) *JobExecutor {
	// CREATE ROOT CONTEXT
	ctx, cancel := context.WithCancel(context.Background())

	// CREATE ERROR GROUP
	errGroup, errCtx := utils.NewErrorGroup(ctx)

	// CREATE EXECUTOR
	executor := &JobExecutor{
		job:        job,
		jobCtx:     errCtx,
		cancelFunc: cancel,
		status:     "initializing",
		startTime:  time.Now(),
		logger:     utils.GetLogger(),
		errorGroup: errGroup,
	}

	// SET WORKER POOL SIZE
	poolSize := job.Rules.MaxConcurrent
	if poolSize <= 0 {
		poolSize = 5 // DEFAULT TO 5 WORKERS
	}

	// CREATE WORKER POOL
	executor.workerPool = utils.NewWorkerPool(poolSize)

	return executor
}

// EXECUTE RUNS THE JOB
func (e *JobExecutor) Execute() error {
	e.mu.Lock()
	e.status = "running"
	e.startTime = time.Now()
	e.mu.Unlock()

	// UPDATE JOB STATUS
	e.job.Mutex.Lock()
	e.job.Status = "running"
	e.job.LastRun = time.Now()
	e.job.CancelFunc = e.cancelFunc
	e.job.Mutex.Unlock()

	// UPDATE JOB IN DB
	storage.UpdateJob(e.job)

	// INITIALIZE PIPELINE
	pipeline, err := NewPipeline(e.job)
	if err != nil {
		// LOG ERROR
		e.logger.Error("Failed to create pipeline", map[string]any{
			"jobId": e.job.ID,
			"error": err.Error(),
		})

		// UPDATE JOB STATUS
		e.job.Mutex.Lock()
		e.job.Status = "failed"
		e.job.LastError = err.Error()
		e.job.CancelFunc = nil
		e.job.Mutex.Unlock()

		// UPDATE JOB IN DB
		storage.UpdateJob(e.job)

		return err
	}

	e.pipeline = pipeline

	// SETUP STATUS MONITORING
	go e.monitorStatus()

	// EXECUTE PIPELINE
	err = e.pipeline.Execute(e.job.BaseURL)

	// WAIT FOR ALL WORKERS TO COMPLETE
	e.workerPool.Wait()

	// WAIT FOR ERROR GROUP TO FINISH
	errGroupErr := e.errorGroup.Wait()
	if errGroupErr != nil && err == nil {
		err = errGroupErr
	}

	// UPDATE JOB STATUS
	e.job.Mutex.Lock()

	if err != nil {
		e.job.Status = "failed"
		e.job.LastError = err.Error()
	} else {
		e.job.Status = "completed"
	}

	e.job.CancelFunc = nil
	e.job.Mutex.Unlock()

	// UPDATE JOB IN DB
	storage.UpdateJob(e.job)

	// UPDATE EXECUTOR STATUS
	e.mu.Lock()
	e.status = e.job.Status
	e.endTime = time.Now()
	e.mu.Unlock()

	// LOG COMPLETION
	e.logger.Info("Job execution completed", map[string]any{
		"jobId":     e.job.ID,
		"status":    e.job.Status,
		"duration":  time.Since(e.startTime).String(),
		"processed": atomic.LoadInt32(&e.completedURLs),
		"failed":    atomic.LoadInt32(&e.failedURLs),
		"total":     atomic.LoadInt32(&e.totalURLs),
	})

	return err
}

// STOP STOPS THE JOB
func (e *JobExecutor) Stop() {
	e.mu.Lock()
	oldStatus := e.status
	e.status = "stopping"
	e.mu.Unlock()

	// ONLY CANCEL IF RUNNING
	if oldStatus == "running" {
		// CANCEL CONTEXT
		if e.cancelFunc != nil {
			e.cancelFunc()
		}

		// UPDATE JOB STATUS
		e.job.Mutex.Lock()
		e.job.Status = "stopped"
		e.job.CancelFunc = nil
		e.job.Mutex.Unlock()

		// UPDATE JOB IN DB
		storage.UpdateJob(e.job)

		// WAIT FOR WORKER POOL TO DRAIN
		e.workerPool.Stop()

		// UPDATE STATUS
		e.mu.Lock()
		e.status = "stopped"
		e.endTime = time.Now()
		e.mu.Unlock()

		// LOG STOP
		e.logger.Info("Job execution stopped", map[string]any{
			"jobId":     e.job.ID,
			"duration":  time.Since(e.startTime).String(),
			"processed": atomic.LoadInt32(&e.completedURLs),
			"failed":    atomic.LoadInt32(&e.failedURLs),
			"total":     atomic.LoadInt32(&e.totalURLs),
		})
	}
}

// MONITORSTATUS MONITORS AND UPDATES JOB STATUS
func (e *JobExecutor) monitorStatus() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// UPDATE JOB STATUS
			e.job.Mutex.Lock()
			e.job.LastRun = e.startTime
			completedURLs := atomic.LoadInt32(&e.completedURLs)
			failedURLs := atomic.LoadInt32(&e.failedURLs)
			totalURLs := atomic.LoadInt32(&e.totalURLs)

			// CALCULATE PROGRESS
			progress := 0.0
			if totalURLs > 0 {
				progress = float64(completedURLs+failedURLs) / float64(totalURLs) * 100
			}

			// UPDATE JOB METADATA
			if e.job.Metadata == nil {
				e.job.Metadata = make(map[string]any)
			}

			e.job.Metadata["progress"] = progress
			e.job.Metadata["processedUrls"] = completedURLs
			e.job.Metadata["failedUrls"] = failedURLs
			e.job.Metadata["totalUrls"] = totalURLs
			e.job.Metadata["duration"] = time.Since(e.startTime).String()

			// SAVE JOB STATUS
			storage.UpdateJob(e.job)
			e.job.Mutex.Unlock()

			// CHECK IF JOB IS STILL RUNNING
			e.mu.RLock()
			status := e.status
			e.mu.RUnlock()

			if status != "running" {
				return
			}

		case <-e.jobCtx.Done():
			// JOB CONTEXT CANCELED
			return
		}
	}
}

// ADDERROR ADDS AN ERROR TO THE ERROR LIST
func (e *JobExecutor) AddError(err *utils.ScraperError) {
	// ADD JOB ID TO ERROR
	err.JobID = e.job.ID

	// LOG ERROR
	e.logger.LogScraperError(err)

	// INCREMENT FAILED COUNT
	atomic.AddInt32(&e.failedURLs, 1)

	// APPEND TO ERROR LIST
	e.mu.Lock()
	e.errors = append(e.errors, *err)
	e.mu.Unlock()
}

// MARKURLPROCESSED MARKS A URL AS PROCESSED
func (e *JobExecutor) MarkURLProcessed(url string, success bool) {
	// CHECK IF ALREADY PROCESSED
	if _, exists := e.processedURLs.LoadOrStore(url, success); !exists {
		// INCREMENT TOTAL COUNT
		atomic.AddInt32(&e.totalURLs, 1)

		// INCREMENT SUCCESS/FAILURE COUNT
		if success {
			atomic.AddInt32(&e.completedURLs, 1)
		} else {
			atomic.AddInt32(&e.failedURLs, 1)
		}
	}
}

// SHOULDPROCESSURL CHECKS IF A URL SHOULD BE PROCESSED
func (e *JobExecutor) ShouldProcessURL(url string) bool {
	// CHECK IF ALREADY PROCESSED
	if _, exists := e.processedURLs.Load(url); exists {
		return false
	}

	// CHECK URL PATTERNS
	if e.job.Rules.IncludeURLPattern != "" {
		matched, err := MatchPattern(url, e.job.Rules.IncludeURLPattern)
		if err != nil || !matched {
			return false
		}
	}

	if e.job.Rules.ExcludeURLPattern != "" {
		matched, err := MatchPattern(url, e.job.Rules.ExcludeURLPattern)
		if err == nil && matched {
			return false
		}
	}

	return true
}

// JOBASSET REPRESENTS AN ASSET DETECTED DURING SCRAPING
type JobAsset struct {
	ID          string
	URL         string
	Type        string
	ContentType string
	Size        int64
	Title       string
	Description string
	Metadata    map[string]string
}

// ADDASSET ADDS AN ASSET TO THE JOB
func (e *JobExecutor) AddAsset(asset JobAsset) {
	// CREATE NEW ASSET MODEL
	newAsset := models.Asset{
		ID:          asset.ID,
		URL:         asset.URL,
		Title:       asset.Title,
		Description: asset.Description,
		Type:        asset.Type,
		Size:        asset.Size,
		Metadata:    asset.Metadata,
	}

	// ADD ASSET TO JOB
	e.job.Mutex.Lock()
	e.job.Assets = append(e.job.Assets, newAsset)
	e.job.Mutex.Unlock()

	// SAVE JOB PERIODICALLY
	assetsCount := len(e.job.Assets)
	if assetsCount%10 == 0 {
		storage.SaveJobs()
	}

	// LOG ASSET
	e.logger.Info("Added asset", map[string]any{
		"jobId":      e.job.ID,
		"assetId":    asset.ID,
		"assetUrl":   asset.URL,
		"assetType":  asset.Type,
		"assetCount": assetsCount,
	})
}

// DOWNLOADASSET DOWNLOADS AN ASSET
func (e *JobExecutor) DownloadAsset(asset *models.Asset) error {
	// ADD DOWNLOAD TASK TO WORKER POOL
	return e.workerPool.Submit(func() error {
		// CHECK IF ASSET ALREADY DOWNLOADED
		if asset.Downloaded {
			return nil
		}

		// DOWNLOAD ASSET
		ctx, cancel := context.WithTimeout(e.jobCtx, 10*time.Minute)
		defer cancel()

		err := assets.DownloadAsset(ctx, e.job, asset)
		if err != nil {
			asset.Error = err.Error()
			return fmt.Errorf("asset download failed: %w", err)
		}

		// MARK AS DOWNLOADED
		asset.Downloaded = true

		// GENERATE THUMBNAIL
		thumbnailPath, err := assets.GenerateThumbnail(asset)
		if err != nil {
			e.logger.Warn("Failed to generate thumbnail", map[string]any{
				"jobId":   e.job.ID,
				"assetId": asset.ID,
				"error":   err.Error(),
			})
		} else {
			asset.ThumbnailPath = thumbnailPath
		}

		return nil
	})
}

// GETJOBSTATUS GETS THE CURRENT JOB STATUS
func (e *JobExecutor) GetJobStatus() models.JobStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	completedURLs := atomic.LoadInt32(&e.completedURLs)
	failedURLs := atomic.LoadInt32(&e.failedURLs)
	totalURLs := atomic.LoadInt32(&e.totalURLs)

	// CALCULATE PROGRESS
	progress := 0.0
	if totalURLs > 0 {
		progress = float64(completedURLs+failedURLs) / float64(totalURLs) * 100
	}

	status := models.JobStatus{
		ID:            e.job.ID,
		Status:        e.status,
		Progress:      progress,
		ProcessedURLs: int(completedURLs),
		FailedURLs:    int(failedURLs),
		TotalURLs:     int(totalURLs),
		StartTime:     e.startTime,
		Duration:      time.Since(e.startTime).String(),
		AssetCount:    len(e.job.Assets),
	}

	if !e.endTime.IsZero() {
		status.EndTime = e.endTime
		status.Duration = e.endTime.Sub(e.startTime).String()
	}

	return status
}

// JOBMANAGER MANAGES ALL RUNNING JOBS
type JobManager struct {
	jobs   map[string]*JobExecutor
	mu     sync.RWMutex
	logger *utils.Logger
}

// GLOBAL JOB MANAGER INSTANCE
var (
	defaultJobManager *JobManager
	jobManagerOnce    sync.Once
)

// GETJOBMANAGER RETURNS THE SINGLETON JOB MANAGER INSTANCE
func GetJobManager() *JobManager {
	jobManagerOnce.Do(func() {
		defaultJobManager = &JobManager{
			jobs:   make(map[string]*JobExecutor),
			logger: utils.GetLogger(),
		}
	})

	return defaultJobManager
}

// STARTJOB STARTS A JOB
func (m *JobManager) StartJob(job *models.ScrapingJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// CHECK IF JOB IS ALREADY RUNNING
	if executor, exists := m.jobs[job.ID]; exists && executor.status == "running" {
		return fmt.Errorf("job is already running")
	}

	// CREATE JOB EXECUTOR
	executor := NewJobExecutor(job)

	// ADD TO JOBS MAP
	m.jobs[job.ID] = executor

	// START JOB IN BACKGROUND
	go func() {
		err := executor.Execute()
		if err != nil {
			m.logger.Error("Job execution failed", map[string]any{
				"jobId": job.ID,
				"error": err.Error(),
			})
		}

		// CLEAN UP JOB AFTER COMPLETION
		m.mu.Lock()
		delete(m.jobs, job.ID)
		m.mu.Unlock()
	}()

	return nil
}

// STOPJOB STOPS A RUNNING JOB
func (m *JobManager) StopJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// CHECK IF JOB EXISTS
	executor, exists := m.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	// CHECK IF JOB IS RUNNING
	if executor.status != "running" {
		return fmt.Errorf("job is not running")
	}

	// STOP JOB
	executor.Stop()

	return nil
}

// GETJOBSTATUS GETS THE STATUS OF A JOB
func (m *JobManager) GetJobStatus(jobID string) (models.JobStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// CHECK IF JOB EXISTS
	executor, exists := m.jobs[jobID]
	if !exists {
		return models.JobStatus{}, fmt.Errorf("job not found")
	}

	return executor.GetJobStatus(), nil
}

// GETALLSTATUSES GETS THE STATUS OF ALL JOBS
func (m *JobManager) GetAllStatuses() []models.JobStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]models.JobStatus, 0, len(m.jobs))

	for _, executor := range m.jobs {
		statuses = append(statuses, executor.GetJobStatus())
	}

	return statuses
}

// STOPALL STOPS ALL RUNNING JOBS
func (m *JobManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, executor := range m.jobs {
		if executor.status == "running" {
			executor.Stop()
		}
	}
}

// RUNJOB IS THE MAIN ENTRY POINT FOR STARTING A JOB
func RunJob(job *models.ScrapingJob) {
	// GET JOB MANAGER
	manager := GetJobManager()

	// START JOB
	err := manager.StartJob(job)
	if err != nil {
		// LOG ERROR
		log.Printf("Failed to start job %s: %v", job.ID, err)

		// UPDATE JOB STATUS
		job.Mutex.Lock()
		job.Status = "failed"
		job.LastError = err.Error()
		job.Mutex.Unlock()

		// UPDATE JOB IN DB
		storage.UpdateJob(job)
	}
}
