package storage

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/nickheyer/Crepes/internal/models"
)

var (
	// GLOBAL JOBS STORAGE
	Jobs      = make(map[string]*models.ScrapingJob)
	JobsMutex sync.Mutex
)

// SAVEJOBS SERIALIZES AND SAVES JOBS TO DISK
func SaveJobs() {
	// SERIALIZE JOBS WITHOUT MUTEX AND CANCELFUNC
	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	serializedJobs := make(map[string]models.ScrapingJob)
	for id, job := range Jobs {
		// CREATE A COPY WITHOUT MUTEX AND CANCELFUNC
		serializedJob := *job
		serializedJob.Mutex = nil
		serializedJob.CancelFunc = nil
		serializedJob.CompletedAssets = nil
		serializedJobs[id] = serializedJob
	}

	// SAVE TO FILE
	data, err := json.MarshalIndent(serializedJobs, "", "  ")
	if err != nil {
		log.Printf("Error serializing jobs: %v", err)
		return
	}

	// WRITE TO TEMP FILE FIRST, THEN RENAME FOR ATOMIC OPERATION
	tempFile := "jobs.json.tmp"
	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		log.Printf("Error saving jobs to temp file: %v", err)
		return
	}

	err = os.Rename(tempFile, "jobs.json")
	if err != nil {
		log.Printf("Error renaming temp file to jobs.json: %v", err)
	}
}

// LOADJOBS LOADS JOBS FROM DISK
func LoadJobs() {
	data, err := os.ReadFile("jobs.json")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading jobs: %v", err)
		}
		return
	}

	var serializedJobs map[string]models.ScrapingJob
	err = json.Unmarshal(data, &serializedJobs)
	if err != nil {
		log.Printf("Error parsing jobs.json: %v", err)
		return
	}

	// RESTORE JOBS WITH PROPER MUTEX AND MAPS
	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	for id, serializedJob := range serializedJobs {
		// CREATE A NEW JOB WITH PROPER MUTEX
		loadedJob := serializedJob
		loadedJob.Mutex = &sync.Mutex{} // CREATE NEW MUTEX
		loadedJob.CancelFunc = nil
		loadedJob.CompletedAssets = make(map[string]bool)

		// STORE AS POINTER
		Jobs[id] = &loadedJob

		// RESCHEDULE JOB IF NEEDED
		if loadedJob.Schedule != "" && loadedJob.Status != "running" {
			// SCHEDULE THE JOB (IMPLEMENTATION WILL BE IN SCHEDULER PACKAGE)
		}
	}

	log.Printf("Loaded %d jobs", len(Jobs))
}

// GETJOB RETRIEVES A JOB BY ID
func GetJob(id string) (*models.ScrapingJob, bool) {
	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	job, exists := Jobs[id]
	return job, exists
}

// ADDJOB ADDS A NEW JOB TO STORAGE
func AddJob(job *models.ScrapingJob) {
	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	Jobs[job.ID] = job
	SaveJobs()
}

// DELETEJOB REMOVES A JOB FROM STORAGE
func DeleteJob(id string) bool {
	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	job, exists := Jobs[id]
	if exists {
		// STOP RUNNING JOB
		if job.Status == "running" && job.CancelFunc != nil {
			job.CancelFunc()
		}
		delete(Jobs, id)
		SaveJobs()
	}

	return exists
}
