package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nickheyer/Crepes/internal/models"
)

var (
	// GLOBAL DB CONNECTION
	DB *sql.DB
	// JOBS CACHE
	Jobs      = make(map[string]*models.ScrapingJob)
	JobsMutex sync.Mutex
	// SETTINGS CACHE
	AppSettings     *models.Settings
	SettingsMutex   sync.Mutex
	DefaultSettings = models.Settings{
		AppConfig: models.AppConfig{
			Port:           8080,
			StoragePath:    "./storage",
			ThumbnailsPath: "./thumbnails",
			DataPath:       "./data",
			MaxConcurrent:  5,
			DefaultTimeout: 300000,
		},
		UserConfig: models.UserConfig{
			Theme:                "default",
			DefaultView:          "grid",
			NotificationsEnabled: true,
		},
	}
)

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

	// CREATE ALL TABLES DIRECTLY
	err = createTables()
	if err != nil {
		return err
	}

	// LOAD JOBS INTO MEMORY CACHE
	err = LoadJobs()
	if err != nil {
		return err
	}

	// LOAD TEMPLATES INTO MEMORY CACHE
	err = LoadTemplates()
	if err != nil {
		log.Printf("Warning: Error loading templates: %v", err)
	}

	// LOAD SETTINGS INTO MEMORY CACHE
	err = LoadSettings()
	if err != nil {
		log.Printf("Warning: Error loading settings: %v", err)
		// INITIALIZE WITH DEFAULT SETTINGS IF NONE EXIST
		AppSettings = &DefaultSettings
		SaveSettings()
	}

	return nil
}

func createTables() error {
	// JOBS TABLE
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		name TEXT,
		base_url TEXT NOT NULL,
		selectors TEXT NOT NULL,
		custom_selectors TEXT DEFAULT '[]',
		unified_selectors TEXT DEFAULT '[]',
		rules TEXT NOT NULL,
		schedule TEXT,
		status TEXT NOT NULL,
		last_run TIMESTAMP,
		next_run TIMESTAMP,
		current_page INTEGER NOT NULL DEFAULT 1,
		pipeline TEXT,
		metadata TEXT,
		last_error TEXT
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
		source TEXT,
		FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
	)`)

	if err != nil {
		return err
	}

	// ERROR LOGS TABLE
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS error_logs (
		id TEXT PRIMARY KEY,
		job_id TEXT,
		stage_id TEXT,
		stage_name TEXT,
		url TEXT,
		message TEXT,
		status_code INTEGER,
		timestamp TIMESTAMP,
		metadata TEXT,
		html_snippet TEXT,
		screenshot_path TEXT,
		FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
	)`)

	if err != nil {
		return err
	}

	// TEMPLATES TABLE
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		base_url TEXT NOT NULL,
		selectors TEXT NOT NULL,
		rules TEXT NOT NULL,
		schedule TEXT,
		pipeline TEXT,
		tags TEXT,
		metadata TEXT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	)`)

	if err != nil {
		return err
	}

	// SETTINGS TABLE
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS settings (
		id TEXT PRIMARY KEY DEFAULT 'global',
		app_config TEXT NOT NULL,
		user_config TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`)

	if err != nil {
		return err
	}

	// SCHEMA VERSION TABLE
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL
	)`)

	if err != nil {
		return err
	}

	// RECORD CURRENT VERSION
	_, err = DB.Exec(`
	INSERT OR REPLACE INTO schema_version (version, applied_at)
	VALUES (7, datetime('now'))`)

	return err
}

func LoadJobs() error {
	rows, err := DB.Query(`SELECT 
		id, name, base_url, selectors, custom_selectors, unified_selectors, rules, schedule, 
		status, last_run, next_run, current_page, pipeline, metadata, last_error 
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
		var selectorsJSON, customSelectorsJSON, unifiedSelectorsJSON, rulesJSON, pipelineJSON, metadataJSON, lastError sql.NullString
		var name sql.NullString
		var lastRunTime, nextRunTime sql.NullTime

		err := rows.Scan(
			&job.ID,
			&name,
			&job.BaseURL,
			&selectorsJSON,
			&customSelectorsJSON,
			&unifiedSelectorsJSON,
			&rulesJSON,
			&job.Schedule,
			&job.Status,
			&lastRunTime,
			&nextRunTime,
			&job.CurrentPage,
			&pipelineJSON,
			&metadataJSON,
			&lastError,
		)

		if err != nil {
			log.Printf("Error scanning job row: %v", err)
			continue
		}

		// SET NAME IF PRESENT
		if name.Valid {
			job.Name = name.String
		}

		// LOAD PIPELINE IF PRESENT
		if pipelineJSON.Valid && pipelineJSON.String != "" && pipelineJSON.String != "null" {
			job.Pipeline = pipelineJSON.String
		}

		// LOAD METADATA IF PRESENT
		if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "null" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &job.Metadata); err != nil {
				log.Printf("Error unmarshaling metadata for job %s: %v", job.ID, err)
				job.Metadata = make(map[string]any)
			}
		} else {
			job.Metadata = make(map[string]any)
		}

		// SET LAST ERROR IF PRESENT
		if lastError.Valid {
			job.LastError = lastError.String
		}

		// FIRST TRY TO LOAD UNIFIED SELECTORS
		if unifiedSelectorsJSON.Valid && unifiedSelectorsJSON.String != "" && unifiedSelectorsJSON.String != "null" {
			if err := json.Unmarshal([]byte(unifiedSelectorsJSON.String), &job.Selectors); err != nil {
				log.Printf("Error unmarshaling unified selectors for job %s: %v", job.ID, err)
				job.Selectors = []models.SelectorItem{}
			}
		}

		// PARSE RULES JSON
		if rulesJSON.Valid && rulesJSON.String != "" && rulesJSON.String != "null" {
			if err := json.Unmarshal([]byte(rulesJSON.String), &job.Rules); err != nil {
				log.Printf("Error unmarshaling rules for job %s: %v", job.ID, err)
				continue
			}
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

// LOADSETTTINGS LOADS APP SETTINGS FROM THE DATABASE
func LoadSettings() error {
	var appConfigJSON, userConfigJSON string
	var updatedAt time.Time

	err := DB.QueryRow(`
		SELECT app_config, user_config, updated_at 
		FROM settings 
		WHERE id = 'global'`).Scan(&appConfigJSON, &userConfigJSON, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			// RETURN DEFAULT SETTINGS IF NONE EXIST
			AppSettings = &DefaultSettings
			return nil
		}
		return err
	}

	var settings models.Settings

	// PARSE APP CONFIG
	if err := json.Unmarshal([]byte(appConfigJSON), &settings.AppConfig); err != nil {
		log.Printf("Error unmarshaling app config: %v", err)
		settings.AppConfig = DefaultSettings.AppConfig
	}

	// PARSE USER CONFIG
	if err := json.Unmarshal([]byte(userConfigJSON), &settings.UserConfig); err != nil {
		log.Printf("Error unmarshaling user config: %v", err)
		settings.UserConfig = DefaultSettings.UserConfig
	}

	// SET SETTINGS CACHE
	SettingsMutex.Lock()
	AppSettings = &settings
	SettingsMutex.Unlock()

	log.Printf("Loaded settings, last updated: %v", updatedAt)
	return nil
}

// SAVESETTINGS SAVES APP SETTINGS TO THE DATABASE
func SaveSettings() error {
	SettingsMutex.Lock()
	defer SettingsMutex.Unlock()

	if AppSettings == nil {
		AppSettings = &DefaultSettings
	}

	// SERIALIZE CONFIGS TO JSON
	appConfigJSON, err := json.Marshal(AppSettings.AppConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal app config: %w", err)
	}

	userConfigJSON, err := json.Marshal(AppSettings.UserConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal user config: %w", err)
	}

	// UPSERT SETTINGS
	_, err = DB.Exec(`
		INSERT OR REPLACE INTO settings (id, app_config, user_config, updated_at)
		VALUES ('global', ?, ?, datetime('now'))`,
		string(appConfigJSON),
		string(userConfigJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

// GETSETTINGS RETURNS THE CURRENT APP SETTINGS
func GetSettings() *models.Settings {
	SettingsMutex.Lock()
	defer SettingsMutex.Unlock()

	if AppSettings == nil {
		return &DefaultSettings
	}

	// RETURN A COPY TO AVOID RACE CONDITIONS
	settingsCopy := *AppSettings
	return &settingsCopy
}

// UPDATESETTINGS UPDATES THE APP SETTINGS
func UpdateSettings(settings *models.Settings) error {
	SettingsMutex.Lock()
	AppSettings = settings
	SettingsMutex.Unlock()

	return SaveSettings()
}

func AddJob(job *models.ScrapingJob) error {
	// ENSURE JOB HAS AN ID
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	// ENSURE SELECTORS HAVE IDs
	for i := range job.Selectors {
		if job.Selectors[i].ID == "" {
			job.Selectors[i].ID = uuid.New().String()
		}
	}

	// SERIALIZE SELECTORS
	unifiedSelectorsJSON, err := json.Marshal(job.Selectors)
	if err != nil {
		return err
	}

	// BACKWARD COMPATIBILITY
	emptyArray := "[]"

	// SERIALIZE RULES
	rulesJSON, err := json.Marshal(job.Rules)
	if err != nil {
		return err
	}

	// INITIALIZE REQUIRED FIELDS
	job.Assets = []models.Asset{}
	job.CompletedAssets = make(map[string]bool)
	job.Mutex = &sync.Mutex{}

	// INSERT INTO DATABASE
	_, err = DB.Exec(`
	INSERT INTO jobs (
		id, name, base_url, selectors, custom_selectors, unified_selectors, rules, schedule, 
		status, last_run, next_run, current_page, pipeline, metadata, last_error
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID,
		job.Name,
		job.BaseURL,
		emptyArray, // EMPTY ARRAY FOR BACKWARD COMPATIBILITY
		emptyArray, // EMPTY ARRAY FOR BACKWARD COMPATIBILITY
		string(unifiedSelectorsJSON),
		string(rulesJSON),
		job.Schedule,
		job.Status,
		job.LastRun,
		job.NextRun,
		job.CurrentPage,
		job.Pipeline,
		"{}",
		job.LastError,
	)

	if err != nil {
		return err
	}

	// ADD TO IN-MEMORY MAP
	JobsMutex.Lock()
	Jobs[job.ID] = job
	JobsMutex.Unlock()

	return nil
}

func UpdateJob(job *models.ScrapingJob) error {
	// CREATE A COPY OF THE DATA TO AVOID HOLDING LOCKS DURING I/O
	var unifiedSelectorsJSON, rulesJSON, metadataJSON []byte
	var err error

	// SERIALIZE SELECTORS
	unifiedSelectorsJSON, err = json.Marshal(job.Selectors)
	if err != nil {
		return err
	}

	// BACKWARD COMPATIBILITY
	emptyArray := "[]"

	// SERIALIZE RULES
	rulesJSON, err = json.Marshal(job.Rules)
	if err != nil {
		return err
	}

	// SERIALIZE METADATA
	metadataJSON, err = json.Marshal(job.Metadata)
	if err != nil {
		return err
	}

	// UPDATE DATABASE
	_, err = DB.Exec(`
	UPDATE jobs SET
		name = ?,
		base_url = ?,
		selectors = ?,
		custom_selectors = ?,
		unified_selectors = ?,
		rules = ?,
		schedule = ?,
		status = ?,
		last_run = ?,
		next_run = ?,
		current_page = ?,
		pipeline = ?,
		metadata = ?,
		last_error = ?
	WHERE id = ?`,
		job.Name,
		job.BaseURL,
		emptyArray, // BACKWARD COMPATIBILITY
		emptyArray, // BACKWARD COMPATIBILITY
		string(unifiedSelectorsJSON),
		string(rulesJSON),
		job.Schedule,
		job.Status,
		job.LastRun,
		job.NextRun,
		job.CurrentPage,
		job.Pipeline,
		string(metadataJSON),
		job.LastError,
		job.ID,
	)

	return err
}

func loadAssetsForJob(job *models.ScrapingJob) error {
	rows, err := DB.Query(`SELECT 
		id, url, title, description, author, date, 
		type, size, local_path, thumbnail_path, metadata, 
		downloaded, error, source
		FROM assets WHERE job_id = ?`, job.ID)

	if err != nil {
		return err
	}
	defer rows.Close()

	var assets []models.Asset

	for rows.Next() {
		var asset models.Asset
		var metadataJSON string
		var localPath, thumbnailPath, assetErr, source sql.NullString
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
			&source,
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
		if source.Valid {
			asset.Source = source.String
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

func SaveJobs() {
	// MAKE A COPY OF JOB IDS TO AVOID HOLDING MUTEX WHILE UPDATING
	var jobIDs []string

	JobsMutex.Lock()
	for id := range Jobs {
		jobIDs = append(jobIDs, id)
	}
	JobsMutex.Unlock()

	// UPDATE EACH JOB INDIVIDUALLY
	for _, id := range jobIDs {
		JobsMutex.Lock()
		job, exists := Jobs[id]
		JobsMutex.Unlock()

		if !exists {
			continue
		}

		// LOCK JOB FOR SAFE COPY
		job.Mutex.Lock()

		// CAPTURE ASSET LIST FOR SAVING
		assets := make([]models.Asset, len(job.Assets))
		copy(assets, job.Assets)

		// UPDATE JOB IN DATABASE
		err := UpdateJob(job)
		job.Mutex.Unlock()

		if err != nil {
			log.Printf("Error updating job %s: %v", job.ID, err)
			continue
		}

		// SAVE ASSETS WITHOUT JOB LOCK
		for _, asset := range assets {
			assetCopy := asset // CREATE COPY TO AVOID POINTER ISSUES
			err := saveAsset(&assetCopy, job.ID)
			if err != nil {
				log.Printf("Error saving asset %s: %v", asset.ID, err)
			}
		}
	}
}

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
			metadata, downloaded, error, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
			asset.Source,
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
			error = ?,
			source = ?
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
			asset.Source,
			asset.ID,
		)
	}

	return err
}

func AddAsset(jobID string, asset *models.Asset) error {
	// GET THE JOB
	job, exists := GetJob(jobID)
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// ADD ASSET TO JOB
	job.Mutex.Lock()
	job.Assets = append(job.Assets, *asset)
	job.Mutex.Unlock()

	// SAVE ASSET TO DATABASE
	return saveAsset(asset, jobID)
}

func GetJob(id string) (*models.ScrapingJob, bool) {
	JobsMutex.Lock()
	defer JobsMutex.Unlock()
	job, exists := Jobs[id]
	return job, exists
}

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

func DeleteAsset(assetID string) error {
	_, err := DB.Exec("DELETE FROM assets WHERE id = ?", assetID)
	return err
}

// TEMPLATES STORAGE
var (
	Templates      = make(map[string]*models.JobTemplate)
	TemplatesMutex sync.Mutex
)

// ADDTEMPLATE ADDS A TEMPLATE TO STORAGE
func AddTemplate(template *models.JobTemplate) error {
	// SET TIMESTAMPS
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}
	template.UpdatedAt = time.Now()

	// SERIALIZE TEMPLATE DATA
	templateJSON, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	// CHECK IF TEMPLATE EXISTS
	var exists bool
	err = DB.QueryRow("SELECT 1 FROM templates WHERE id = ?", template.ID).Scan(&exists)

	// IF TEMPLATE DOESN'T EXIST OR ERROR OCCURRED
	if err != nil || !exists {
		if err != sql.ErrNoRows && err != nil {
			return fmt.Errorf("error checking template existence: %w", err)
		}

		// INSERT NEW TEMPLATE
		_, err = DB.Exec(`
			INSERT INTO templates (
				id, name, description, base_url, selectors, rules, 
				schedule, pipeline, tags, metadata, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			template.ID,
			template.Name,
			template.Description,
			template.BaseURL,
			string(templateJSON),
			"{}",
			template.Schedule,
			template.Pipeline,
			"[]",
			"{}",
			template.CreatedAt,
			template.UpdatedAt,
		)
	} else {
		// UPDATE EXISTING TEMPLATE
		_, err = DB.Exec(`
			UPDATE templates SET
				name = ?,
				description = ?,
				base_url = ?,
				selectors = ?,
				rules = ?,
				schedule = ?,
				pipeline = ?,
				tags = ?,
				metadata = ?,
				updated_at = ?
			WHERE id = ?`,
			template.Name,
			template.Description,
			template.BaseURL,
			string(templateJSON),
			"{}",
			template.Schedule,
			template.Pipeline,
			"[]",
			"{}",
			template.UpdatedAt,
			template.ID,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	// UPDATE IN-MEMORY CACHE
	TemplatesMutex.Lock()
	Templates[template.ID] = template
	TemplatesMutex.Unlock()

	return nil
}

// GETTEMPLATES RETURNS ALL TEMPLATES
func GetTemplates() ([]*models.JobTemplate, error) {
	TemplatesMutex.Lock()
	defer TemplatesMutex.Unlock()

	templates := make([]*models.JobTemplate, 0, len(Templates))
	for _, template := range Templates {
		templates = append(templates, template)
	}

	return templates, nil
}

// GETTEMPLATE RETURNS A TEMPLATE BY ID
func GetTemplate(id string) (*models.JobTemplate, bool) {
	TemplatesMutex.Lock()
	defer TemplatesMutex.Unlock()

	template, exists := Templates[id]
	return template, exists
}

// UPDATETEMPLATE UPDATES A TEMPLATE
func UpdateTemplate(template *models.JobTemplate) error {
	template.UpdatedAt = time.Now()

	templateJSON, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	_, err = DB.Exec(`
		UPDATE templates SET
			name = ?,
			description = ?,
			base_url = ?,
			selectors = ?,
			rules = ?,
			schedule = ?,
			pipeline = ?,
			tags = ?,
			metadata = ?,
			updated_at = ?
		WHERE id = ?`,
		template.Name,
		template.Description,
		template.BaseURL,
		string(templateJSON),
		"{}",
		template.Schedule,
		template.Pipeline,
		"[]",
		"{}",
		template.UpdatedAt,
		template.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	// UPDATE IN-MEMORY CACHE
	TemplatesMutex.Lock()
	Templates[template.ID] = template
	TemplatesMutex.Unlock()

	return nil
}

// DELETETEMPLATE DELETES A TEMPLATE
func DeleteTemplate(id string) bool {
	TemplatesMutex.Lock()
	defer TemplatesMutex.Unlock()

	_, exists := Templates[id]
	if exists {
		_, err := DB.Exec("DELETE FROM templates WHERE id = ?", id)
		if err != nil {
			log.Printf("Error deleting template %s from database: %v", id, err)
		}
		delete(Templates, id)
	}

	return exists
}

// LOADTEMPLATES LOADS TEMPLATES FROM THE DATABASE
func LoadTemplates() error {
	rows, err := DB.Query(`SELECT 
		id, name, description, base_url, selectors, rules, 
		schedule, pipeline, tags, metadata, created_at, updated_at 
		FROM templates`)
	if err != nil {
		return err
	}
	defer rows.Close()

	TemplatesMutex.Lock()
	defer TemplatesMutex.Unlock()

	// CLEAR EXISTING TEMPLATES
	Templates = make(map[string]*models.JobTemplate)

	for rows.Next() {
		var template models.JobTemplate
		var selectorsJSON, rulesJSON, pipelineJSON, tagsJSON, metadataJSON string
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&template.BaseURL,
			&selectorsJSON,
			&rulesJSON,
			&template.Schedule,
			&pipelineJSON,
			&tagsJSON,
			&metadataJSON,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			log.Printf("Error scanning template row: %v", err)
			continue
		}

		// PARSE SELECTORS
		if err := json.Unmarshal([]byte(selectorsJSON), &template.Selectors); err != nil {
			log.Printf("Error unmarshaling selectors for template %s: %v", template.ID, err)
			template.Selectors = []models.SelectorItem{}
		}

		// PARSE RULES
		if err := json.Unmarshal([]byte(rulesJSON), &template.Rules); err != nil {
			log.Printf("Error unmarshaling rules for template %s: %v", template.ID, err)
		}

		// PARSE PIPELINE
		if pipelineJSON != "" && pipelineJSON != "null" {
			template.Pipeline = pipelineJSON
		}

		// PARSE TAGS
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
			log.Printf("Error unmarshaling tags for template %s: %v", template.ID, err)
			template.Tags = []string{}
		} else {
			template.Tags = tags
		}

		// PARSE METADATA
		if err := json.Unmarshal([]byte(metadataJSON), &template.Metadata); err != nil {
			log.Printf("Error unmarshaling metadata for template %s: %v", template.ID, err)
			template.Metadata = make(map[string]any)
		}

		// SET TIMESTAMPS
		if createdAt.Valid {
			template.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			template.UpdatedAt = updatedAt.Time
		}

		// ADD TO MAP
		Templates[template.ID] = &template
	}

	log.Printf("Loaded %d templates from database", len(Templates))
	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func StartPeriodicSave(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			SaveJobs()
			SaveSettings()
		}
	}()
}
