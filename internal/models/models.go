package models

import (
	"context"
	"sync"
	"time"
)

// SETTINGS DATA MODELS
type Settings struct {
	AppConfig  AppConfig  `json:"appConfig"`
	UserConfig UserConfig `json:"userConfig"`
}

type AppConfig struct {
	Port              int           `json:"port"`
	StoragePath       string        `json:"storagePath"`
	ThumbnailsPath    string        `json:"thumbnailsPath"`
	DataPath          string        `json:"dataPath"`
	LogsPath          string        `json:"logsPath"`
	ErrorsPath        string        `json:"errorsPath"`
	MaxConcurrent     int           `json:"maxConcurrent"`
	MaxBrowsers       int           `json:"maxBrowsers"`
	MaxBrowserTabs    int           `json:"maxBrowserTabs"`
	LogFile           string        `json:"logFile"`
	DefaultTimeout    time.Duration `json:"defaultTimeout"`
	BrowserLifetime   time.Duration `json:"browserLifetime"`
	StoreErrorDetails bool          `json:"storeErrorDetails"`
	DevMode           bool          `json:"devMode"`
}

type UserConfig struct {
	Theme                string `json:"theme"`
	DefaultView          string `json:"defaultView"`
	NotificationsEnabled bool   `json:"notificationsEnabled"`
}

// SCRAPINGJOB REPRESENTS A SCRAPING JOB CONFIGURATION
type ScrapingJob struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	BaseURL             string             `json:"baseUrl"`
	Selectors           []SelectorItem     `json:"selectors"` // UNIFIED SELECTORS
	Rules               ScrapingRules      `json:"rules"`
	Schedule            string             `json:"schedule"`
	Status              string             `json:"status"`
	LastRun             time.Time          `json:"lastRun"`
	NextRun             time.Time          `json:"nextRun"`
	Assets              []Asset            `json:"assets"`
	Pipeline            string             `json:"pipeline"` // JSON REPRESENTATION OF PIPELINE
	CompletedAssets     map[string]bool    `json:"-"`
	Metadata            map[string]any     `json:"metadata"`
	Mutex               *sync.Mutex        `json:"-"`
	CancelFunc          context.CancelFunc `json:"-"`
	DownloadsInProgress int32              `json:"-"`
	CurrentPage         int                `json:"currentPage"`
	LastError           string             `json:"lastError,omitempty"`
}

// JOBSTATUS REPRESENTS THE CURRENT STATUS OF A JOB
type JobStatus struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	Progress      float64   `json:"progress"`
	ProcessedURLs int       `json:"processedUrls"`
	FailedURLs    int       `json:"failedUrls"`
	TotalURLs     int       `json:"totalUrls"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime,omitempty"`
	Duration      string    `json:"duration"`
	AssetCount    int       `json:"assetCount"`
	Errors        any       `json:"errors,omitempty"`
}

// JOBTEMPLATE REPRESENTS A SAVED JOB TEMPLATE
type JobTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	BaseURL     string         `json:"baseUrl"`
	Selectors   []SelectorItem `json:"selectors"`
	Rules       ScrapingRules  `json:"rules"`
	Schedule    string         `json:"schedule,omitempty"`
	Pipeline    string         `json:"pipeline,omitempty"` // JSON REPRESENTATION OF PIPELINE
	Tags        []string       `json:"tags,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// SELECTORITEM REPRESENTS A SELECTOR CONFIGURATION
type SelectorItem struct {
	ID              string `json:"id"`              // UNIQUE IDENTIFIER
	Name            string `json:"name"`            // USER-FRIENDLY NAME
	Type            string `json:"type"`            // CSS OR XPATH
	Value           string `json:"value"`           // THE SELECTOR VALUE
	AttributeSource string `json:"attributeSource"` // THE SOURCE ATTRIBUTE (HREF, SRC, ETC)
	Attribute       string `json:"attribute"`       // THE ATTRIBUTE TO EXTRACT
	Description     string `json:"description"`     // DESCRIPTION OF THE SELECTOR
	Priority        int    `json:"priority"`        // SELECTOR EXECUTION PRIORITY
	IsOptional      bool   `json:"isOptional"`      // IF TRUE, FAILURE WONT STOP PROCESSING
	URLPattern      string `json:"urlPattern"`      // URL PATTERN TO MATCH
	Purpose         string `json:"purpose"`         // PURPOSE OF THE SELECTOR (LINKS, ASSETS, METADATA, ETC)
}

// STAGE REPRESENTS A PIPELINE STAGE
type Stage struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	NextStages  []string       `json:"nextStages"`
	Position    Position       `json:"position,omitempty"`
	Concurrency int            `json:"concurrency"`
}

// POSITION REPRESENTS A STAGE POSITION IN THE EDITOR
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// CONNECTION REPRESENTS A CONNECTION BETWEEN STAGES
type Connection struct {
	ID           string `json:"id"`
	SourceID     string `json:"sourceId"`
	TargetID     string `json:"targetId"`
	SourceOutput string `json:"sourceOutput"`
	TargetInput  string `json:"targetInput"`
}

// PIPELINE REPRESENTS A COMPLETE PIPELINE
type Pipeline struct {
	Stages      map[string]*Stage `json:"stages"`
	Connections []Connection      `json:"connections"`
	EntryPoints []string          `json:"entryPoints"`
}

// SCRAPINGRULES DEFINES RULES FOR SCRAPING
type ScrapingRules struct {
	MaxDepth                int           `json:"maxDepth"`
	MaxAssets               int           `json:"maxAssets"`
	MaxPages                int           `json:"maxPages"`
	MaxConcurrent           int           `json:"maxConcurrent"`
	IncludeURLPattern       string        `json:"includeUrlPattern"`
	ExcludeURLPattern       string        `json:"excludeUrlPattern"`
	Timeout                 time.Duration `json:"timeout"`
	UserAgent               string        `json:"userAgent"`
	RequestDelay            time.Duration `json:"requestDelay"`
	RandomizeDelay          bool          `json:"randomizeDelay"`
	PaginationSelector      string        `json:"paginationSelector"`
	VideoExtractionHeadless *bool         `json:"videoExtractionHeadless"`
	MaxSize                 int64         `json:"maxSize"`    // MAXIMUM FILE SIZE TO DOWNLOAD IN BYTES
	RetryCount              int           `json:"retryCount"` // NUMBER OF TIMES TO RETRY FAILED REQUESTS
}

// ASSET REPRESENTS A DOWNLOADED ASSET
type Asset struct {
	ID            string            `json:"id"`
	URL           string            `json:"url"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Author        string            `json:"author"`
	Date          string            `json:"date"`
	Type          string            `json:"type"`
	Size          int64             `json:"size"`
	LocalPath     string            `json:"localPath"`
	ThumbnailPath string            `json:"thumbnailPath"`
	JobID         string            `json:"jobId"`
	Metadata      map[string]string `json:"metadata"`
	Downloaded    bool              `json:"downloaded"`
	Error         string            `json:"error,omitempty"`
	Source        string            `json:"source,omitempty"` // TRACKS WHICH SELECTOR CREATED THIS
}

// ERRORCONTEXT PROVIDES ADDITIONAL CONTEXT FOR ERRORS
type ErrorContext struct {
	URL           string         `json:"url"`
	JobID         string         `json:"jobId"`
	StageID       string         `json:"stageId"`
	StageName     string         `json:"stageName"`
	ItemID        string         `json:"itemId"`
	StatusCode    int            `json:"statusCode,omitempty"`
	HTML          string         `json:"html,omitempty"`
	Screenshot    []byte         `json:"screenshot,omitempty"`
	ScreenshotURL string         `json:"screenshotUrl,omitempty"`
	StackTrace    string         `json:"stackTrace,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}
