package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/scraper"
	"github.com/nickheyer/Crepes/internal/utils"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// REGISTER JOB HANDLERS
func RegisterJobHandlers(router *mux.Router, db *gorm.DB, engine *scraper.Engine, scheduler *scraper.Scheduler) {
	// GET ALL JOBS
	router.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		var jobs []models.Job

		// PRELOAD ASSETS COUNT
		result := db.Model(&models.Job{}).
			Preload("Assets").
			Order("created_at DESC").
			Find(&jobs)

		if result.Error != nil {
			log.Printf("Failed to fetch jobs: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch jobs")
			return
		}

		// FIX EMPTY ARRAYS TO PREVENT NULL IN JSON
		for i := range jobs {
			if jobs[i].Selectors == nil {
				jobs[i].Selectors = []interface{}{}
			}
			if jobs[i].Filters == nil {
				jobs[i].Filters = []interface{}{}
			}
			if jobs[i].Rules == nil {
				jobs[i].Rules = map[string]interface{}{}
			}
			if jobs[i].Processing == nil {
				jobs[i].Processing = map[string]interface{}{
					"thumbnails":    true,
					"metadata":      true,
					"deduplication": true,
				}
			}
			if jobs[i].Tags == nil {
				jobs[i].Tags = []interface{}{}
			}
		}

		utils.RespondWithJSON(w, http.StatusOK, jobs)
	}).Methods("GET")

	// GET JOB BY ID
	router.HandleFunc("/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		var job models.Job
		result := db.Preload("Assets").First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		// FIX EMPTY ARRAYS
		if job.Selectors == nil {
			job.Selectors = []interface{}{}
		}
		if job.Filters == nil {
			job.Filters = []interface{}{}
		}
		if job.Rules == nil {
			job.Rules = map[string]interface{}{}
		}
		if job.Processing == nil {
			job.Processing = map[string]interface{}{
				"thumbnails":    true,
				"metadata":      true,
				"deduplication": true,
			}
		}
		if job.Tags == nil {
			job.Tags = []interface{}{}
		}

		utils.RespondWithJSON(w, http.StatusOK, job)
	}).Methods("GET")

	// CREATE JOB
	router.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		var job models.Job
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			log.Printf("Invalid request payload: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// GENERATE ID IF NOT PROVIDED
		if job.ID == "" {
			job.ID = utils.GenerateID("job")
		}

		// SET TIMESTAMPS
		job.CreatedAt = time.Now()
		job.UpdatedAt = time.Now()

		// ENSURE STATUS IS SET
		if job.Status == "" {
			job.Status = "idle"
		}

		// CREATE FERRET TEMPLATE FROM JOB CONFIG
		if job.Template == "" {
			job.Template = scraper.GenerateFerretTemplate(&job)
		}

		// SAVE JOB TO DATABASE
		if result := db.Create(&job); result.Error != nil {
			log.Printf("Failed to create job: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create job")
			return
		}

		// SCHEDULE JOB IF IT HAS A CRON SCHEDULE
		if job.Schedule != "" {
			scheduler.ScheduleJob(&job)
		}

		utils.RespondWithJSON(w, http.StatusCreated, job)
	}).Methods("POST")

	// UPDATE JOB
	router.HandleFunc("/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// CHECK IF JOB EXISTS
		var existingJob models.Job
		result := db.First(&existingJob, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for update: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		// PARSE UPDATED JOB DATA
		var updatedJob models.Job
		if err := json.NewDecoder(r.Body).Decode(&updatedJob); err != nil {
			log.Printf("Invalid request payload for update: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// UPDATE FIELDS
		updatedJob.ID = id
		updatedJob.UpdatedAt = time.Now()
		updatedJob.CreatedAt = existingJob.CreatedAt

		// UPDATE FERRET TEMPLATE IF JOB CONFIG CHANGED
		if updatedJob.Template == "" {
			updatedJob.Template = scraper.GenerateFerretTemplate(&updatedJob)
		}

		// SAVE UPDATED JOB TO DATABASE
		if err := db.Model(&existingJob).Updates(updatedJob).Error; err != nil {
			log.Printf("Failed to update job: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update job")
			return
		}

		// HANDLE SCHEDULE CHANGES
		oldSchedule := existingJob.Schedule
		newSchedule := updatedJob.Schedule

		if oldSchedule != newSchedule {
			// REMOVE EXISTING SCHEDULE
			if oldSchedule != "" {
				scheduler.RemoveJob(id)
			}
			// ADD NEW SCHEDULE
			if newSchedule != "" {
				db.First(&updatedJob, "id = ?", id) // REFRESH JOB DATA
				scheduler.ScheduleJob(&updatedJob)
			}
		}

		// RETURN UPDATED JOB
		var finalJob models.Job
		db.Preload("Assets").First(&finalJob, "id = ?", id)

		// FIX EMPTY ARRAYS
		if finalJob.Selectors == nil {
			finalJob.Selectors = []interface{}{}
		}
		if finalJob.Filters == nil {
			finalJob.Filters = []interface{}{}
		}
		if finalJob.Rules == nil {
			finalJob.Rules = map[string]interface{}{}
		}
		if finalJob.Processing == nil {
			finalJob.Processing = map[string]interface{}{
				"thumbnails":    true,
				"metadata":      true,
				"deduplication": true,
			}
		}
		if finalJob.Tags == nil {
			finalJob.Tags = []interface{}{}
		}

		utils.RespondWithJSON(w, http.StatusOK, finalJob)
	}).Methods("PUT")

	// DELETE JOB
	router.HandleFunc("/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// REMOVE JOB FROM SCHEDULER IF SCHEDULED
		scheduler.RemoveJob(id)

		// STOP JOB IF RUNNING
		engine.StopJob(id)

		// DELETE JOB FROM DATABASE
		result := db.Delete(&models.Job{}, "id = ?", id)
		if result.Error != nil {
			log.Printf("Failed to delete job: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete job")
			return
		}

		// CHECK IF JOB WAS FOUND
		if result.RowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Job deleted successfully",
		})
	}).Methods("DELETE")

	// START JOB
	router.HandleFunc("/jobs/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// CHECK IF JOB EXISTS
		var job models.Job
		result := db.First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for start: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		// START JOB ASYNCHRONOUSLY
		go func() {
			err := engine.RunJob(id)
			if err != nil {
				log.Printf("Error starting job %s: %v", id, err)
			}
		}()

		// UPDATE JOB STATUS IMMEDIATELY FOR UI
		db.Model(&models.Job{}).Where("id = ?", id).Update("status", "running")

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Job started successfully",
		})
	}).Methods("POST")

	// STOP JOB
	router.HandleFunc("/jobs/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// CHECK IF JOB EXISTS
		var job models.Job
		result := db.First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for stop: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		// STOP JOB
		engine.StopJob(id)

		// UPDATE JOB STATUS IMMEDIATELY FOR UI
		db.Model(&models.Job{}).Where("id = ?", id).Update("status", "stopped")

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Job stopped successfully",
		})
	}).Methods("POST")

	// GET JOB ASSETS
	router.HandleFunc("/jobs/{id}/assets", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		var assets []models.Asset
		result := db.Where("job_id = ?", id).Order("created_at DESC").Find(&assets)
		if result.Error != nil {
			log.Printf("Failed to fetch job assets: %v", result.Error)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch job assets")
			return
		}

		// FIX EMPTY METADATA
		for i := range assets {
			if assets[i].Metadata == nil {
				assets[i].Metadata = map[string]interface{}{}
			}
		}

		utils.RespondWithJSON(w, http.StatusOK, assets)
	}).Methods("GET")

	// GET JOB STATISTICS
	router.HandleFunc("/jobs/{id}/statistics", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		// CHECK IF JOB EXISTS
		var job models.Job
		result := db.First(&job, "id = ?", id)
		if result.Error != nil {
			log.Printf("Job not found for statistics: %v", result.Error)
			utils.RespondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		// COUNT TOTAL ASSETS
		var totalAssets int64
		db.Model(&models.Asset{}).Where("job_id = ?", id).Count(&totalAssets)

		// COUNT ASSETS BY TYPE
		var assets []models.Asset
		db.Select("type").Where("job_id = ?", id).Find(&assets)

		// COMPILE STATISTICS
		assetTypes := make(map[string]int)
		for _, asset := range assets {
			assetTypes[asset.Type]++
		}

		stats := map[string]interface{}{
			"totalAssets": totalAssets,
			"assetTypes":  assetTypes,
			"progress":    engine.GetJobProgress(id),
			"duration":    engine.GetJobDuration(id),
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    stats,
		})
	}).Methods("GET")
}
