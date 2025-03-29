package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/scraper"
	"github.com/nickheyer/Crepes/internal/utils"
	"gorm.io/gorm"
)

func GetAllJobs(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jobs []models.Job
		result := db.Model(&models.Job{}).
			Preload("Assets").
			Order("created_at DESC").
			Find(&jobs)
		if result.Error != nil {
			log.Printf("Failed to fetch jobs: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch jobs")
			return
		}
		for i := range jobs {
			if jobs[i].Selectors == nil {
				jobs[i].Selectors = []any{}
			}
			if jobs[i].Filters == nil {
				jobs[i].Filters = []any{}
			}
			if jobs[i].Rules == nil {
				jobs[i].Rules = map[string]any{}
			}
			if jobs[i].Processing == nil {
				jobs[i].Processing = map[string]any{
					"thumbnails":    true,
					"metadata":      true,
					"deduplication": true,
				}
			}
			if jobs[i].Tags == nil {
				jobs[i].Tags = []any{}
			}
		}
		utils.RespondWithJSON(w, http.StatusOK, jobs)
	}
}

func GetJobByID(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		var job models.Job
		result := db.Preload("Assets").First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		if job.Selectors == nil {
			job.Selectors = []any{}
		}
		if job.Filters == nil {
			job.Filters = []any{}
		}
		if job.Rules == nil {
			job.Rules = map[string]any{}
		}
		if job.Processing == nil {
			job.Processing = map[string]any{
				"thumbnails":    true,
				"metadata":      true,
				"deduplication": true,
			}
		}
		if job.Tags == nil {
			job.Tags = []any{}
		}
		utils.RespondWithJSON(w, http.StatusOK, job)
	}
}

func CreateJob(db *gorm.DB, scheduler *scraper.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var job models.Job
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			log.Printf("Invalid request payload: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		if job.ID == "" {
			job.ID = utils.GenerateID("job")
		}
		job.CreatedAt = time.Now()
		job.UpdatedAt = time.Now()
		if job.Status == "" {
			job.Status = "idle"
		}
		if result := db.Create(&job); result.Error != nil {
			log.Printf("Failed to create job: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create job")
			return
		}
		if job.Schedule != "" {
			scheduler.ScheduleJob(&job)
		}
		utils.RespondWithJSON(w, http.StatusCreated, job)
	}
}

func UpdateJob(db *gorm.DB, scheduler *scraper.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		var existingJob models.Job
		result := db.First(&existingJob, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for update: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		var updatedJob models.Job
		if err := json.NewDecoder(r.Body).Decode(&updatedJob); err != nil {
			log.Printf("Invalid request payload for update: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		updatedJob.ID = id
		updatedJob.UpdatedAt = time.Now()
		updatedJob.CreatedAt = existingJob.CreatedAt
		if err := db.Model(&existingJob).Updates(updatedJob).Error; err != nil {
			log.Printf("Failed to update job: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update job")
			return
		}
		oldSchedule := existingJob.Schedule
		newSchedule := updatedJob.Schedule
		if oldSchedule != newSchedule {
			if oldSchedule != "" {
				scheduler.RemoveJob(id)
			}
			if newSchedule != "" {
				db.First(&updatedJob, "id = ?", id)
				scheduler.ScheduleJob(&updatedJob)
			}
		}
		var finalJob models.Job
		db.Preload("Assets").First(&finalJob, "id = ?", id)
		if finalJob.Selectors == nil {
			finalJob.Selectors = []any{}
		}
		if finalJob.Filters == nil {
			finalJob.Filters = []any{}
		}
		if finalJob.Rules == nil {
			finalJob.Rules = map[string]any{}
		}
		if finalJob.Processing == nil {
			finalJob.Processing = map[string]any{
				"thumbnails":    true,
				"metadata":      true,
				"deduplication": true,
			}
		}
		if finalJob.Tags == nil {
			finalJob.Tags = []any{}
		}
		utils.RespondWithJSON(w, http.StatusOK, finalJob)
	}
}

func DeleteJob(db *gorm.DB, engine *scraper.Engine, scheduler *scraper.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		scheduler.RemoveJob(id)
		engine.StopJob(id)
		result := db.Delete(&models.Job{}, "id = ?", id)
		if result.Error != nil {
			log.Printf("Failed to delete job: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete job")
			return
		}
		if result.RowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Job deleted successfully",
		})
	}
}

func StartJob(db *gorm.DB, engine *scraper.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		var job models.Job
		result := db.First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for start: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		go func() {
			err := engine.RunJob(id)
			if err != nil {
				log.Printf("Error starting job %s: %v", id, err)
			}
		}()
		db.Model(&models.Job{}).Where("id = ?", id).Update("status", "running")
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Job started successfully",
		})
	}
}

func StopJob(db *gorm.DB, engine *scraper.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		var job models.Job
		result := db.First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for stop: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		engine.StopJob(id)
		db.Model(&models.Job{}).Where("id = ?", id).Update("status", "stopped")
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Job stopped successfully",
		})
	}
}

func GetJobAssets(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		var assets []models.Asset
		result := db.Where("job_id = ?", id).Order("created_at DESC").Find(&assets)
		if result.Error != nil {
			log.Printf("Failed to fetch job assets: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch job assets")
			return
		}
		for i := range assets {
			if assets[i].Metadata == nil {
				assets[i].Metadata = map[string]any{}
			}
		}
		utils.RespondWithJSON(w, http.StatusOK, assets)
	}
}

func GetJobStatistics(db *gorm.DB, engine *scraper.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		var job models.Job
		result := db.First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for statistics: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		var totalAssets int64
		db.Model(&models.Asset{}).Where("job_id = ?", id).Count(&totalAssets)
		var assets []models.Asset
		db.Select("type").Where("job_id = ?", id).Find(&assets)
		assetTypes := make(map[string]int)
		for _, asset := range assets {
			assetTypes[asset.Type]++
		}
		jobProgress, err := engine.GetJobProgress(id)
		if err != nil {
			jobProgress.CompletedTasks = 0
			jobProgress.TotalTasks = 0
		}
		jobDuration, err := engine.GetJobDuration(id)
		if err != nil {
			jobDuration = 0
		}
		stats := map[string]any{
			"totalAssets": totalAssets,
			"assetTypes":  assetTypes,
			"progress":    jobProgress,
			"duration":    jobDuration,
		}
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    stats,
		})
	}
}
