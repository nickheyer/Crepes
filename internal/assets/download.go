package assets

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"
	"golang.org/x/net/publicsuffix"
)

// DOWNLOADASSET DOWNLOADS AN ASSET AND SAVES IT TO DISK
func DownloadAsset(ctx context.Context, job *models.ScrapingJob, asset *models.Asset) error {
	// EXTRACT CONTENT TYPE FROM URL IF PRESENT
	contentType := ""
	originalURL := asset.URL
	if hashIndex := strings.LastIndex(asset.URL, "#content-type="); hashIndex != -1 {
		contentType = asset.URL[hashIndex+len("#content-type="):]
		// CLEAN URL BY REMOVING CONTENT TYPE
		asset.URL = asset.URL[:hashIndex]
	}

	// SET BETTER ASSET TYPE BASED ON CONTENT TYPE IF AVAILABLE
	if contentType != "" {
		if strings.Contains(contentType, "video/") {
			asset.Type = "video"
		} else if strings.Contains(contentType, "audio/") {
			asset.Type = "audio"
		} else if strings.Contains(contentType, "image/") {
			asset.Type = "image"
		} else if strings.Contains(contentType, "application/json") {
			asset.Type = "document"
		}
	}

	// VALIDATE URL BEFORE PROCEEDING
	if strings.HasPrefix(asset.URL, "blob:") {
		return fmt.Errorf("cannot download blob URL directly: %s", asset.URL)
	}

	// CREATE DIRECTORY IF NOT EXISTS
	assetDir := filepath.Join(config.AppConfig.StoragePath, job.ID)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return err
	}

	// DETERMINE FILE EXTENSION
	ext := filepath.Ext(asset.URL)
	if ext == "" {
		ext = GetExtensionByType(asset.Type)
	}

	// CREATE FILE PATH
	fileName := fmt.Sprintf("%s%s", asset.ID, ext)
	filePath := filepath.Join(assetDir, fileName)
	asset.LocalPath = filepath.Join(job.ID, fileName)

	// CREATE SEPARATE BACKGROUND CONTEXT FOR DOWNLOAD
	baseCtx := context.Background()
	dlCtx, dlCancel := context.WithTimeout(baseCtx, 60*time.Minute) // 1 HOUR TIMEOUT FOR LARGE DOWNLOADS

	// MONITOR THE PARENT CONTEXT FOR CANCELLATION BUT NOT TIMEOUT
	go func() {
		select {
		case <-ctx.Done():
			// ONLY CANCEL IF IT'S A MANUAL CANCELLATION, NOT A TIMEOUT
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				dlCancel()
			}
		case <-dlCtx.Done():
			// THIS WILL HAPPEN WHEN DLCTX TIMES OUT OR IS CANCELED
		}
	}()

	defer dlCancel()

	req, err := http.NewRequestWithContext(dlCtx, "GET", asset.URL, nil)
	if err != nil {
		return err
	}

	// SET HEADERS TO MIMIC BROWSER
	req.Header.Set("User-Agent", config.GetRandomUserAgent())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", job.BaseURL)

	// ADD CONTENT TYPE TO METADATA
	if asset.Metadata == nil {
		asset.Metadata = make(map[string]string)
	}
	asset.Metadata["original_url"] = originalURL
	if contentType != "" {
		asset.Metadata["content_type"] = contentType
	}

	// CREATE COOKIE JAR FOR SESSION HANDLING
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// CREATE TRANSPORT WITH RELAXED SECURITY
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableCompression:  false,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// CREATE CLIENT WITH COOKIE SUPPORT AND REASONABLE TIMEOUT
	client := &http.Client{
		Jar:       jar,
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	var resp *http.Response
	var lastErr error

	// TRY UP TO 3 TIMES WITH BACKOFF
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break // SUCCESS OR CLIENT ERROR
		}

		if resp != nil {
			resp.Body.Close()
		}

		lastErr = err
		if err == nil {
			lastErr = fmt.Errorf("server returned status: %d", resp.StatusCode)
		}

		// EXPONENTIAL BACKOFF
		backoffTime := time.Duration(attempt+1) * 2 * time.Second
		select {
		case <-dlCtx.Done():
			if errors.Is(dlCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("download timeout after multiple attempts: %v", lastErr)
			}
			return dlCtx.Err()
		case <-time.After(backoffTime):
			// CONTINUE AFTER WAITING
			log.Printf("Retrying download for %s (attempt %d of 3)", asset.URL, attempt+2)
		}
	}

	if resp == nil {
		return fmt.Errorf("failed after 3 attempts: %v", lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status: %d %s", resp.StatusCode, resp.Status)
	}

	// CHECK RESPONSE CONTENT TYPE AND UPDATE ASSET TYPE
	if respContentType := resp.Header.Get("Content-Type"); respContentType != "" {
		asset.Metadata["actual_content_type"] = respContentType

		// UPDATE ASSET TYPE BASED ON ACTUAL CONTENT
		if strings.Contains(respContentType, "video/") {
			asset.Type = "video"
		} else if strings.Contains(respContentType, "audio/") {
			asset.Type = "audio"
		} else if strings.Contains(respContentType, "image/") {
			asset.Type = "image"
		} else if strings.Contains(respContentType, "application/json") ||
			strings.Contains(respContentType, "text/") {
			asset.Type = "document"
		}

		// IF WE DOWNLOADED A TEXT FILE BUT THOUGHT IT WAS VIDEO, RENAME TO PROPER EXTENSION
		if (asset.Type == "document") && (ext != ".txt" && ext != ".json") {
			// REDEFINE FILE PATH WITH CORRECT EXTENSION
			if strings.Contains(respContentType, "application/json") {
				ext = ".json"
			} else {
				ext = ".txt"
			}

			// UPDATE FILE PATH
			fileName = fmt.Sprintf("%s%s", asset.ID, ext)
			filePath = filepath.Join(assetDir, fileName)
			asset.LocalPath = filepath.Join(job.ID, fileName)
		}
	}

	// DETERMINE SIZE
	asset.Size = resp.ContentLength

	// CREATE FILE
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// COPY RESPONSE BODY TO FILE WITH PROGRESS REPORTING
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				info, err := out.Stat()
				if err == nil && asset.Size > 0 {
					log.Printf("Downloading %s: %d/%d bytes (%.1f%%)",
						asset.URL,
						info.Size(),
						asset.Size,
						float64(info.Size())/float64(asset.Size)*100,
					)
				} else if err == nil {
					log.Printf("Downloading %s: %d bytes (unknown size)",
						asset.URL,
						info.Size(),
					)
				}
			case <-done:
				return
			case <-dlCtx.Done():
				return
			}
		}
	}()

	written, err := io.Copy(out, resp.Body)
	close(done)

	if err != nil {
		return err
	}

	log.Printf("Downloaded %s: %d bytes", asset.URL, written)

	// CHECK THAT THE DOWNLOADED FILE IS VALID (E.G., NOT EMPTY OR TOO SMALL)
	if written < 1000 && asset.Type == "video" {
		// READ CONTENT TO CHECK IF IT'S JSON/TEXT
		out.Seek(0, 0) // REWIND FILE
		content, err := io.ReadAll(out)
		if err != nil {
			log.Printf("Error reading downloaded file: %v", err)
		} else {
			// LOG CONTENT FOR DEBUGGING
			log.Printf("Downloaded file content appears to be text, not video: %s", utils.TruncateString(string(content), 200))

			// CHECK IF IT'S JSON
			if strings.HasPrefix(string(content), "{") || strings.HasPrefix(string(content), "[") {
				asset.Type = "document"
				asset.Error = "Downloaded API response instead of video content"
			}
		}
	}

	return nil
}
