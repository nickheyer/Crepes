package assets

import (
	"context"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"

	"slices"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/storage"
)

// PROCESSASSET PROCESSES AND DOWNLOADS AN ASSET FROM A PAGE
func ProcessAsset(job *models.ScrapingJob, selection *goquery.Selection, pageURL string, pageMetadata map[string]string) {
	// EXTRACT ASSET URL
	assetURL := ""

	// TRY COMMON ATTRIBUTES
	for _, attr := range []string{"src", "href", "data-src", "data-video", "data-media"} {
		if url, exists := selection.Attr(attr); exists && url != "" {
			assetURL = MakeAbsoluteURL(pageURL, url)
			break
		}
	}

	if assetURL == "" {
		return
	}

	// CHECK IF ALREADY PROCESSED
	job.Mutex.Lock()
	if job.CompletedAssets[assetURL] {
		job.Mutex.Unlock()
		return
	}
	job.CompletedAssets[assetURL] = true
	job.Mutex.Unlock()

	// CREATE NEW ASSET
	asset := models.Asset{
		ID:          uuid.New().String(),
		URL:         assetURL,
		Type:        GetAssetType(assetURL),
		Title:       pageMetadata["title"],
		Description: pageMetadata["description"],
		Author:      pageMetadata["author"],
		Date:        pageMetadata["date"],
		Metadata:    make(map[string]string),
		Downloaded:  false,
	}

	// CREATE A DETACHED CONTEXT FOR THE DOWNLOAD THAT WON'T BE CANCELED WHEN THE JOB COMPLETES
	downloadCtx := context.Background()

	// INCREMENT DOWNLOAD COUNTER
	atomic.AddInt32(&job.DownloadsInProgress, 1)

	// DOWNLOAD ASSET - USE A GOROUTINE WITH THE DETACHED CONTEXT
	go func() {
		err := DownloadAsset(downloadCtx, job, &asset)
		if err != nil {
			log.Printf("Error downloading asset %s: %v", assetURL, err)
			asset.Error = err.Error()
		} else {
			asset.Downloaded = true

			// GENERATE THUMBNAIL
			thumbnailPath, err := GenerateThumbnail(&asset)
			if err != nil {
				log.Printf("Error generating thumbnail for %s: %v", assetURL, err)
			} else {
				asset.ThumbnailPath = thumbnailPath
			}
		}

		// ADD ASSET TO JOB
		job.Mutex.Lock()
		job.Assets = append(job.Assets, asset)
		job.Mutex.Unlock()

		// SAVE PERIODICALLY AFTER ADDING ASSETS
		assetCount := len(job.Assets)
		if assetCount%5 == 0 {
			storage.SaveJobs()
		}

		// DECREMENT DOWNLOAD COUNTER
		atomic.AddInt32(&job.DownloadsInProgress, -1)
	}()
}

// GETASSETTYPE DETERMINES THE TYPE OF AN ASSET FROM ITS URL
func GetAssetType(url string) string {
	// EXTRACT CONTENT TYPE FROM URL IF PRESENT
	contentType := ""
	if hashIndex := strings.LastIndex(url, "#content-type="); hashIndex != -1 {
		contentType = url[hashIndex+len("#content-type="):]
		// CLEAN URL BY REMOVING CONTENT TYPE
		url = url[:hashIndex]
	}

	// CHECK CONTENT TYPE FIRST IF AVAILABLE
	if contentType != "" {
		if strings.Contains(contentType, "video/") {
			return "video"
		}
		if strings.Contains(contentType, "audio/") {
			return "audio"
		}
		if strings.Contains(contentType, "image/") {
			return "image"
		}
		if strings.Contains(contentType, "application/pdf") ||
			strings.Contains(contentType, "text/") {
			return "document"
		}
	}

	// THEN CHECK FILE EXTENSION
	ext := strings.ToLower(filepath.Ext(url))

	videoExts := []string{".mp4", ".webm", ".mkv", ".avi", ".mov", ".flv", ".m4v", ".mpg", ".mpeg", ".ts"}
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".ico"}
	audioExts := []string{".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma"}
	docExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".rtf", ".csv"}

	if slices.Contains(videoExts, ext) {
		return "video"
	}

	if slices.Contains(imageExts, ext) {
		return "image"
	}

	if slices.Contains(audioExts, ext) {
		return "audio"
	}

	if slices.Contains(docExts, ext) {
		return "document"
	}

	// GUESS FROM URL PATTERNS
	urlLower := strings.ToLower(url)

	if strings.Contains(urlLower, "video") ||
		strings.Contains(urlLower, "movie") ||
		strings.Contains(urlLower, "watch") ||
		strings.Contains(urlLower, "stream") ||
		strings.Contains(urlLower, "/mp4") {
		return "video"
	}

	if strings.Contains(urlLower, "image") ||
		strings.Contains(urlLower, "photo") ||
		strings.Contains(urlLower, "pic") {
		return "image"
	}

	if strings.Contains(urlLower, "audio") ||
		strings.Contains(urlLower, "music") ||
		strings.Contains(urlLower, "sound") {
		return "audio"
	}

	if strings.Contains(urlLower, "doc") ||
		strings.Contains(urlLower, "pdf") ||
		strings.Contains(urlLower, "file") {
		return "document"
	}

	// DEFAULT TO 'UNKNOWN' OR A GUESS BASED ON URL PATTERNS
	if strings.Contains(urlLower, "/api/") {
		// API ENDPOINTS OFTEN RETURN JSON, NOT MEDIA
		return "document"
	}

	return "unknown"
}

// GETEXTENSIONBYTYPE RETURNS AN APPROPRIATE FILE EXTENSION FOR A GIVEN ASSET TYPE
func GetExtensionByType(assetType string) string {
	switch assetType {
	case "video":
		return ".mp4"
	case "image":
		return ".jpg"
	case "audio":
		return ".mp3"
	case "document":
		return ".txt" // Change from .pdf to .txt for API responses that are likely text
	default:
		return ".bin"
	}
}

// MAKEABSOLUTEURL CONVERTS A RELATIVE URL TO AN ABSOLUTE URL
func MakeAbsoluteURL(base, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(refURL).String()
}

func ProcessExtractedAsset(ctx context.Context, job *models.ScrapingJob, asset *models.Asset, pageURL string) {
	// INCREMENT DOWNLOAD COUNTER
	atomic.AddInt32(&job.DownloadsInProgress, 1)

	// DOWNLOAD ASSET - USE A GOROUTINE WITH THE DETACHED CONTEXT
	err := DownloadAsset(ctx, job, asset)
	if err != nil {
		log.Printf("Error downloading extracted asset %s: %v", asset.URL, err)
		asset.Error = err.Error()
	} else {
		asset.Downloaded = true

		// GENERATE THUMBNAIL
		thumbnailPath, err := GenerateThumbnail(asset)
		if err != nil {
			log.Printf("Error generating thumbnail for %s: %v", asset.URL, err)
		} else {
			asset.ThumbnailPath = thumbnailPath
		}
	}

	// ADD ASSET TO JOB
	job.Mutex.Lock()
	job.Assets = append(job.Assets, *asset)
	job.Mutex.Unlock()

	// SAVE PERIODICALLY AFTER ADDING ASSETS
	assetCount := len(job.Assets)
	if assetCount%5 == 0 {
		storage.SaveJobs()
	}

	// DECREMENT DOWNLOAD COUNTER
	atomic.AddInt32(&job.DownloadsInProgress, -1)
}
