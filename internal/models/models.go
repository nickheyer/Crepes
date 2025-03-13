package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// JOB MODEL
type Job struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	BaseURL     string    `json:"baseUrl"`
	Description string    `json:"description"`
	Status      string    `json:"status" gorm:"default:'idle'"`
	LastRun     time.Time `json:"lastRun"`
	NextRun     time.Time `json:"nextRun"`
	Schedule    string    `json:"schedule"`
	Selectors   JSONArray `json:"selectors" gorm:"type:text"`
	Filters     JSONArray `json:"filters" gorm:"type:text"`
	Rules       JSONMap   `json:"rules" gorm:"type:text"`
	Processing  JSONMap   `json:"processing" gorm:"type:text"`
	Tags        JSONArray `json:"tags" gorm:"type:text"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Assets      []Asset   `json:"assets,omitempty" gorm:"foreignKey:JobID"`
}

// ASSET MODEL
type Asset struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	JobID         string    `json:"jobId"`
	URL           string    `json:"url"`
	Type          string    `json:"type"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	LocalPath     string    `json:"localPath"`
	ThumbnailPath string    `json:"thumbnailPath"`
	Size          int64     `json:"size"`
	Date          time.Time `json:"date"`
	Metadata      JSONMap   `json:"metadata" gorm:"type:text"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// SETTING MODEL
type Setting struct {
	Key       string `json:"key" gorm:"primaryKey"`
	Value     string `json:"value"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SELECTOR MODEL
type Selector struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Attribute   string `json:"attribute"`
	Description string `json:"description"`
	Purpose     string `json:"purpose"`
	Priority    int    `json:"priority"`
	IsOptional  bool   `json:"isOptional"`
	URLPattern  string `json:"urlPattern"`
}

// FILTER MODEL
type Filter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Pattern     string `json:"pattern"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// JSON ARRAY TYPE FOR STORING ARRAYS IN SQLITE
type JSONArray []any

// SCAN FROM DB VALUE
func (j *JSONArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONArray value")
	}
	if len(bytes) == 0 {
		*j = make(JSONArray, 0)
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// VALUE FOR DB STORAGE
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// JSON MAP TYPE FOR STORING OBJECTS IN SQLITE
type JSONMap map[string]interface{}

// SCAN FROM DB VALUE
func (j *JSONMap) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONMap value")
	}
	if len(bytes) == 0 {
		*j = make(JSONMap)
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// VALUE FOR DB STORAGE
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// BEFORE CREATE HOOK TO SET DEFAULTS
func (job *Job) BeforeCreate(tx *gorm.DB) (err error) {
	// SET DEFAULT VALUES IF EMPTY
	if job.Status == "" {
		job.Status = "idle"
	}
	if job.Selectors == nil {
		job.Selectors = make(JSONArray, 0)
	}
	if job.Filters == nil {
		job.Filters = make(JSONArray, 0)
	}
	if job.Rules == nil {
		job.Rules = make(JSONMap)
	}
	if job.Processing == nil {
		job.Processing = JSONMap{
			"thumbnails":    true,
			"metadata":      true,
			"deduplication": true,
			"headless":      true,
		}
	}
	if job.Tags == nil {
		job.Tags = make(JSONArray, 0)
	}
	return
}
