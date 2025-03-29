package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

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

type Setting struct {
	Key       string `json:"key" gorm:"primaryKey"`
	Value     string `json:"value"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

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

type Filter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Pattern     string `json:"pattern"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type Stage struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Condition   Condition         `json:"condition"`
	Parallelism ParallelismConfig `json:"parallelism"`
	Tasks       []Task            `json:"tasks"`
	Config      map[string]any    `json:"config"`
}

type Condition struct { // CONDITION DEFINES WHEN A STAGE OR TASK SHOULD EXECUTE
	Type   string         `json:"type"` // always, never, javascript, comparison
	Config map[string]any `json:"config"`
}

type ParallelismConfig struct { // PARALLELISM CONFIG DEFINES HOW TASKS ARE EXECUTED
	Mode       string `json:"mode"` // sequential, parallel, worker-per-item
	MaxWorkers int    `json:"maxWorkers"`
}

type Task struct { // TASK DEFINES A SINGLE OPERATION IN THE PIPELINE
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"` // Type of task (navigate, click, extract, etc.)
	Description string         `json:"description"`
	Config      map[string]any `json:"config"`
	InputRefs   []string       `json:"inputRefs"` // References to outputs from other tasks
	Condition   Condition      `json:"condition"`
	RetryConfig RetryConfig    `json:"retryConfig"`
}

type RetryConfig struct { // RETRY CONFIG DEFINES HOW TASK RETRIES ARE HANDLED
	MaxRetries  int     `json:"maxRetries"`
	DelayMS     int     `json:"delayMS"`
	BackoffRate float64 `json:"backoffRate"`
}

type Job struct { // UPDATE JOB MODEL TO INCLUDE PIPELINE FIELD
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
	Pipeline    string    `json:"pipeline" gorm:"type:text"` // JSON STRING CONTAINING PIPELINE STAGES
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Assets      []Asset   `json:"assets,omitempty" gorm:"foreignKey:JobID"`
}

type JobConfig struct { // JOB CONFIG PROVIDES DEFAULT SETTINGS FOR A JOB
	BrowserSettings   BrowserSettings   `json:"browserSettings"`
	ScraperSettings   ScraperSettings   `json:"scraperSettings"`
	ResourceSettings  ResourceSettings  `json:"resourceSettings"`
	DefaultHeaders    map[string]string `json:"defaultHeaders"`
	RateLimiting      RateLimitSettings `json:"rateLimiting"`
	ProxyConfig       ProxyConfig       `json:"proxyConfig"`
	CaptchaConfig     CaptchaConfig     `json:"captchaConfig"`
	RetrySettings     RetrySettings     `json:"retrySettings"`
	DownloadSettings  DownloadSettings  `json:"downloadSettings"`
	ExtractorSettings ExtractorSettings `json:"extractorSettings"`
}

type BrowserSettings struct {
	Headless        bool                `json:"headless"`
	UserAgent       string              `json:"userAgent"`
	ViewportWidth   int                 `json:"viewportWidth"`
	ViewportHeight  int                 `json:"viewportHeight"`
	Locale          string              `json:"locale"`
	Timezone        string              `json:"timezone"`
	Cookies         []map[string]string `json:"cookies"`
	BrowserArgs     []string            `json:"browserArgs"`
	DefaultTimeout  int                 `json:"defaultTimeout"`
	ExtraSettings   map[string]any      `json:"extraSettings"`
	RecordVideo     bool                `json:"recordVideo"`
	RecordSnapshots bool                `json:"recordSnapshots"`
}

type ScraperSettings struct { // SCRAPER SETTINGS CONFIGURE GENERAL SCRAPER BEHAVIOR
	MaxDepth              int    `json:"maxDepth"`
	MaxPages              int    `json:"maxPages"`
	MaxAssets             int    `json:"maxAssets"`
	MaxConcurrentRequests int    `json:"maxConcurrentRequests"`
	DefaultNavigationMode string `json:"defaultNavigationMode"` // load, domcontentloaded, networkidle
	FollowRedirects       bool   `json:"followRedirects"`
	SameDomainOnly        bool   `json:"sameDomainOnly"`
	IncludeSubdomains     bool   `json:"includeSubdomains"`
	IncludeUrlPattern     string `json:"includeUrlPattern"`
	ExcludeUrlPattern     string `json:"excludeUrlPattern"`
	TrackUrlHistory       bool   `json:"trackUrlHistory"`
}

type ResourceSettings struct { // RESOURCE SETTINGS CONFIGURE RESOURCE POOLS
	MaxBrowsers int `json:"maxBrowsers"`
	MaxPages    int `json:"maxPages"`
	MaxWorkers  int `json:"maxWorkers"`
}

type RateLimitSettings struct { // RATE LIMIT SETTINGS CONFIGURE REQUEST THROTTLING
	Enabled              bool    `json:"enabled"`
	RequestDelay         int     `json:"requestDelay"`         // MS BETWEEN REQUESTS
	RandomizeDelay       bool    `json:"randomizeDelay"`       // ADD RANDOM JITTER
	DelayVariation       float64 `json:"delayVariation"`       // PERCENTAGE OF VARIATION (0.0-1.0)
	MaxRequestsPerMinute int     `json:"maxRequestsPerMinute"` // RATE LIMITING
}

type ProxyConfig struct {
	Enabled       bool     `json:"enabled"`
	Type          string   `json:"type"` // http, socks5, etc.
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Username      string   `json:"username"`
	Password      string   `json:"password"`
	ProxyRotation bool     `json:"proxyRotation"`
	ProxyList     []string `json:"proxyList"`
}

type CaptchaConfig struct { // CAPTCHA CONFIG DEFINES CAPTCHA SOLVING SETTINGS
	Enabled      bool   `json:"enabled"`
	Service      string `json:"service"` // 2captcha, anticaptcha, etc.
	ApiKey       string `json:"apiKey"`
	SolveTimeout int    `json:"solveTimeout"` // SECONDS
}

type RetrySettings struct { // RETRY SETTINGS CONFIGURE REQUEST RETRIES
	MaxRetries       int     `json:"maxRetries"`
	InitialDelayMS   int     `json:"initialDelayMS"`
	BackoffRate      float64 `json:"backoffRate"`
	MaxDelayMS       int     `json:"maxDelayMS"`
	RetryOnTimeout   bool    `json:"retryOnTimeout"`
	RetryOnFailure   bool    `json:"retryOnFailure"`
	RetryStatusCodes []int   `json:"retryStatusCodes"` // HTTP STATUS CODES TO RETRY
}

type DownloadSettings struct { // DOWNLOAD SETTINGS CONFIGURE ASSET DOWNLOADS
	Enabled           bool     `json:"enabled"`
	DownloadDir       string   `json:"downloadDir"`
	CreateSubfolders  bool     `json:"createSubfolders"`
	MaxConcurrent     int      `json:"maxConcurrent"`
	MaxFileSizeMB     int      `json:"maxFileSizeMB"`
	AllowedExtensions []string `json:"allowedExtensions"`
	SkipExisting      bool     `json:"skipExisting"`
	OverwriteExisting bool     `json:"overwriteExisting"`
	Compress          bool     `json:"compress"`
}

// EXTRACTOR SETTINGS CONFIGURE DATA EXTRACTION
type ExtractorSettings struct {
	ExtractMetadata       bool `json:"extractMetadata"`
	GenerateThumbnails    bool `json:"generateThumbnails"`
	ExtractDocumentText   bool `json:"extractDocumentText"`
	DownloadLinkedContent bool `json:"downloadLinkedContent"`
	MaxExtractSizeMB      int  `json:"maxExtractSizeMB"`
}

// JSON ARRAY TYPE FOR STORING ARRAYS IN SQLITE
type JSONArray []any

// SCAN FROM DB VALUE
func (j *JSONArray) Scan(value any) error {
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
type JSONMap map[string]any

// SCAN FROM DB VALUE
func (j *JSONMap) Scan(value any) error {
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
