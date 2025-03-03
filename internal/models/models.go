package models

import (
	"context"
	"sync"
	"time"
)

type ScrapingJob struct {
	ID                  string             `json:"id"`
	BaseURL             string             `json:"baseUrl"`
	Selectors           []Selector         `json:"selectors"`
	Rules               ScrapingRules      `json:"rules"`
	Schedule            string             `json:"schedule"`
	Status              string             `json:"status"`
	LastRun             time.Time          `json:"lastRun"`
	NextRun             time.Time          `json:"nextRun"`
	Assets              []Asset            `json:"assets"`
	CompletedAssets     map[string]bool    `json:"-"`
	Mutex               *sync.Mutex        `json:"-"`
	CancelFunc          context.CancelFunc `json:"-"`
	DownloadsInProgress int32              `json:"-"`
	CurrentPage         int                `json:"currentPage"`
}

type Selector struct {
	Type  string `json:"type"` // CSS OR XPATH
	Value string `json:"value"`
	For   string `json:"for"` // LINKS, ASSETS, TITLE, DESCRIPTION, AUTHOR, DATE, PAGINATION
}

type ScrapingRules struct {
	MaxDepth           int           `json:"maxDepth"`
	MaxAssets          int           `json:"maxAssets"`
	IncludeURLPattern  string        `json:"includeUrlPattern"`
	ExcludeURLPattern  string        `json:"excludeUrlPattern"`
	Timeout            time.Duration `json:"timeout"`
	UserAgent          string        `json:"userAgent"`
	RequestDelay       time.Duration `json:"requestDelay"`
	RandomizeDelay     bool          `json:"randomizeDelay"`
	PaginationSelector string        `json:"paginationSelector"`
}

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
}

type AppConfig struct {
	Port           int           `json:"port"`
	StoragePath    string        `json:"storagePath"`
	ThumbnailsPath string        `json:"thumbnailsPath"`
	MaxConcurrent  int           `json:"maxConcurrent"`
	LogFile        string        `json:"logFile"`
	DefaultTimeout time.Duration `json:"defaultTimeout"`
}
