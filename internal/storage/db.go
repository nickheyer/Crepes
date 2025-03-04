package storage

import (
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nickheyer/Crepes/internal/models"
)

var (
	// GLOBAL DB CONNECTION
	DB *sql.DB
	// JOBS CACHE
	Jobs      = make(map[string]*models.ScrapingJob)
	JobsMutex sync.Mutex
)

// INITDB INITIALIZES THE DATABASE CONNECTION AND SCHEMA
func InitDB(dbPath string) error {
	var err error

	// OPEN DATABASE CONNECTION
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// SET PRAGMAS FOR BETTER PERFORMANCE
	_, err = DB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return err
	}

	// CREATE TABLES IF THEY DON'T EXIST
	err = createTables()
	if err != nil {
		return err
	}

	// LOAD JOBS INTO MEMORY CACHE
	return LoadJobs()
}

// CREATETABLES CREATES THE NECESSARY DATABASE TABLES
func createTables() error {
	// JOBS TABLE
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		base_url TEXT NOT NULL,
		selectors TEXT NOT NULL,
		rules TEXT NOT NULL,
		schedule TEXT,
		status TEXT NOT NULL,
		last_run TIMESTAMP,
		next_run TIMESTAMP,
		current_page INTEGER NOT NULL DEFAULT 1
	)`)
	if err != nil {
		return err
	}

	// ASSETS TABLE
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS assets (
		id TEXT PRIMARY KEY,
		job_id TEXT NOT NULL,
		url TEXT NOT NULL,
		title TEXT,
		description TEXT,
		author TEXT,
		date TEXT,
		type TEXT NOT NULL,
		size INTEGER,
		local_path TEXT,
		thumbnail_path TEXT,
		metadata TEXT,
		downloaded BOOLEAN NOT NULL DEFAULT 0,
		error TEXT,
		FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
	)`)
	if err != nil {
		return err
	}

	return nil
}

// LOADJOBS LOADS ALL JOBS FROM THE DATABASE INTO MEMORY
func LoadJobs() error {
	rows, err := DB.Query(`SELECT 
		id, base_url, selectors, rules, schedule, 
		status, last_run, next_run, current_page 
		FROM jobs`)
	if err != nil {
		return err
	}
	defer rows.Close()

	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	// CLEAR EXISTING JOBS
	Jobs = make(map[string]*models.ScrapingJob)

	for rows.Next() {
		var job models.ScrapingJob
		var selectorsJSON, rulesJSON string
		var lastRunTime, nextRunTime sql.NullTime

		err := rows.Scan(
			&job.ID,
			&job.BaseURL,
			&selectorsJSON,
			&rulesJSON,
			&job.Schedule,
			&job.Status,
			&lastRunTime,
			&nextRunTime,
			&job.CurrentPage,
		)
		if err != nil {
			log.Printf("Error scanning job row: %v", err)
			continue
		}

		// PARSE SELECTORS JSON
		if err := json.Unmarshal([]byte(selectorsJSON), &job.Selectors); err != nil {
			log.Printf("Error unmarshaling selectors for job %s: %v", job.ID, err)
			continue
		}

		// PARSE RULES JSON
		if err := json.Unmarshal([]byte(rulesJSON), &job.Rules); err != nil {
			log.Printf("Error unmarshaling rules for job %s: %v", job.ID, err)
			continue
		}

		// SET TIMES IF NOT NULL
		if lastRunTime.Valid {
			job.LastRun = lastRunTime.Time
		}
		if nextRunTime.Valid {
			job.NextRun = nextRunTime.Time
		}

		// INITIALIZE REQUIRED FIELDS
		job.Assets = []models.Asset{}
		job.CompletedAssets = make(map[string]bool)
		job.Mutex = &sync.Mutex{}

		// LOAD ASSETS FOR THIS JOB
		loadAssetsForJob(&job)

		// ADD TO MEMORY CACHE
		Jobs[job.ID] = &job
	}

	log.Printf("Loaded %d jobs from database", len(Jobs))
	return nil
}

// LOADASSETSFORJOB LOADS ASSETS FOR A SPECIFIC JOB
func loadAssetsForJob(job *models.ScrapingJob) error {
	rows, err := DB.Query(`SELECT 
		id, url, title, description, author, date, 
		type, size, local_path, thumbnail_path, metadata, 
		downloaded, error 
		FROM assets WHERE job_id = ?`, job.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var assets []models.Asset

	for rows.Next() {
		var asset models.Asset
		var metadataJSON string
		var localPath, thumbnailPath, assetErr sql.NullString
		var size sql.NullInt64

		err := rows.Scan(
			&asset.ID,
			&asset.URL,
			&asset.Title,
			&asset.Description,
			&asset.Author,
			&asset.Date,
			&asset.Type,
			&size,
			&localPath,
			&thumbnailPath,
			&metadataJSON,
			&asset.Downloaded,
			&assetErr,
		)
		if err != nil {
			log.Printf("Error scanning asset row: %v", err)
			continue
		}

		// SET NULLABLE FIELDS
		if size.Valid {
			asset.Size = size.Int64
		}
		if localPath.Valid {
			asset.LocalPath = localPath.String
		}
		if thumbnailPath.Valid {
			asset.ThumbnailPath = thumbnailPath.String
		}
		if assetErr.Valid {
			asset.Error = assetErr.String
		}

		// PARSE METADATA JSON
		asset.Metadata = make(map[string]string)
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &asset.Metadata); err != nil {
				log.Printf("Error unmarshaling metadata for asset %s: %v", asset.ID, err)
			}
		}

		assets = append(assets, asset)
	}

	job.Assets = assets
	return nil
}

// ADDJOB ADDS A NEW JOB TO THE DATABASE AND MEMORY CACHE
func AddJob(job *models.ScrapingJob) error {
	// SERIALIZE COMPLEX FIELDS TO JSON
	selectorsJSON, err := json.Marshal(job.Selectors)
	if err != nil {
		return err
	}

	rulesJSON, err := json.Marshal(job.Rules)
	if err != nil {
		return err
	}

	// INSERT INTO DATABASE
	_, err = DB.Exec(`
	INSERT INTO jobs (
		id, base_url, selectors, rules, schedule, 
		status, last_run, next_run, current_page
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID,
		job.BaseURL,
		string(selectorsJSON),
		string(rulesJSON),
		job.Schedule,
		job.Status,
		job.LastRun,
		job.NextRun,
		job.CurrentPage,
	)
	if err != nil {
		return err
	}

	// ADD TO MEMORY CACHE
	JobsMutex.Lock()
	Jobs[job.ID] = job
	JobsMutex.Unlock()

	return nil
}

// UPDATEJOB UPDATES AN EXISTING JOB IN THE DATABASE
func UpdateJob(job *models.ScrapingJob) error {
	// SERIALIZE COMPLEX FIELDS TO JSON
	selectorsJSON, err := json.Marshal(job.Selectors)
	if err != nil {
		return err
	}

	rulesJSON, err := json.Marshal(job.Rules)
	if err != nil {
		return err
	}

	// UPDATE DATABASE
	_, err = DB.Exec(`
	UPDATE jobs SET
		base_url = ?,
		selectors = ?,
		rules = ?,
		schedule = ?,
		status = ?,
		last_run = ?,
		next_run = ?,
		current_page = ?
	WHERE id = ?`,
		job.BaseURL,
		string(selectorsJSON),
		string(rulesJSON),
		job.Schedule,
		job.Status,
		job.LastRun,
		job.NextRun,
		job.CurrentPage,
		job.ID,
	)
	return err
}

// SAVEJOBS UPDATES ALL JOBS IN THE DATABASE
func SaveJobs() {
	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	for _, job := range Jobs {
		job.Mutex.Lock()
		err := UpdateJob(job)
		if err != nil {
			log.Printf("Error updating job %s: %v", job.ID, err)
		}

		// SAVE ASSETS FOR THIS JOB
		for _, asset := range job.Assets {
			err := saveAsset(&asset, job.ID)
			if err != nil {
				log.Printf("Error saving asset %s: %v", asset.ID, err)
			}
		}
		job.Mutex.Unlock()
	}
}

// SAVEASSET SAVES A SINGLE ASSET TO THE DATABASE
func saveAsset(asset *models.Asset, jobID string) error {
	// SERIALIZE METADATA TO JSON
	metadataJSON, err := json.Marshal(asset.Metadata)
	if err != nil {
		return err
	}

	// CHECK IF ASSET EXISTS
	var exists bool
	err = DB.QueryRow("SELECT 1 FROM assets WHERE id = ?", asset.ID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err == sql.ErrNoRows {
		// INSERT NEW ASSET
		_, err = DB.Exec(`
		INSERT INTO assets (
			id, job_id, url, title, description, author,
			date, type, size, local_path, thumbnail_path,
			metadata, downloaded, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			asset.ID,
			jobID,
			asset.URL,
			asset.Title,
			asset.Description,
			asset.Author,
			asset.Date,
			asset.Type,
			asset.Size,
			asset.LocalPath,
			asset.ThumbnailPath,
			string(metadataJSON),
			asset.Downloaded,
			asset.Error,
		)
	} else {
		// UPDATE EXISTING ASSET
		_, err = DB.Exec(`
		UPDATE assets SET
			url = ?,
			title = ?,
			description = ?,
			author = ?,
			date = ?,
			type = ?,
			size = ?,
			local_path = ?,
			thumbnail_path = ?,
			metadata = ?,
			downloaded = ?,
			error = ?
		WHERE id = ?`,
			asset.URL,
			asset.Title,
			asset.Description,
			asset.Author,
			asset.Date,
			asset.Type,
			asset.Size,
			asset.LocalPath,
			asset.ThumbnailPath,
			string(metadataJSON),
			asset.Downloaded,
			asset.Error,
			asset.ID,
		)
	}

	return err
}

// GETJOB RETRIEVES A JOB BY ID
func GetJob(id string) (*models.ScrapingJob, bool) {
	JobsMutex.Lock()
	defer JobsMutex.Unlock()
	job, exists := Jobs[id]
	return job, exists
}

// DELETEJOB REMOVES A JOB FROM THE DATABASE AND MEMORY CACHE
func DeleteJob(id string) bool {
	JobsMutex.Lock()
	defer JobsMutex.Unlock()

	job, exists := Jobs[id]
	if exists {
		// STOP RUNNING JOB
		if job.Status == "running" && job.CancelFunc != nil {
			job.CancelFunc()
		}

		// DELETE FROM DATABASE
		_, err := DB.Exec("DELETE FROM jobs WHERE id = ?", id)
		if err != nil {
			log.Printf("Error deleting job %s from database: %v", id, err)
		}

		// DELETE FROM MEMORY CACHE
		delete(Jobs, id)
	}

	return exists
}

// DELETEASSET REMOVES AN ASSET FROM THE DATABASE
func DeleteAsset(assetID string) error {
	_, err := DB.Exec("DELETE FROM assets WHERE id = ?", assetID)
	return err
}

// CLOSEDB CLOSES THE DATABASE CONNECTION
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

// PERIODIC SAVE FUNCTION TO RUN IN BACKGROUND
func StartPeriodicSave(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			SaveJobs()
		}
	}()
}
