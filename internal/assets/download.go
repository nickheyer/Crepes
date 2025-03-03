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
	"time"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"golang.org/x/net/publicsuffix"
)

// DOWNLOADASSET DOWNLOADS AN ASSET AND SAVES IT TO DISK
func DownloadAsset(ctx context.Context, job *models.ScrapingJob, asset *models.Asset) error {
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
	for attempt := range 3 {
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
	return nil
}
