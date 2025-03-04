package scraper

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/nickheyer/Crepes/internal/assets"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/storage"
)

func RunJob(job *models.ScrapingJob) {
	job.Mutex.Lock()
	if job.Status == "running" {
		job.Mutex.Unlock()
		return
	}

	// CREATE CANCELABLE CONTEXT WITHOUT TIMEOUT
	ctx, cancel := context.WithCancel(context.Background())
	job.CancelFunc = cancel
	job.Status = "running"
	job.LastRun = time.Now()

	job.Mutex.Unlock()

	storage.SaveJobs()
	log.Printf("Started job %s: %s", job.ID, job.BaseURL)

	// TEST SITE ACCESSIBILITY WITH SEPARATE TIMEOUT
	accessCtx, accessCancel := context.WithTimeout(ctx, 20*time.Second)
	defer accessCancel()
	if err := TestSiteAccessibility(accessCtx, job.BaseURL); err != nil {
		log.Printf("WARNING: Site accessibility check failed: %v", err)
		// CONTINUE ANYWAY, BUT LOG THE WARNING
	}

	// CREATE HEADLESS BROWSER CONTEXT - NO GLOBAL TIMEOUT
	browserCtx, browserCancel := CreateBrowserContext(ctx, job)
	defer browserCancel()

	// START SCRAPING WITHOUT GLOBAL TIMEOUT
	err := ScrapeURL(browserCtx, job, job.BaseURL, 0)

	// UPDATE JOB STATUS
	job.Mutex.Lock()
	if err != nil && !IsContextCanceled(err) {
		job.Status = "failed"
		log.Printf("Job %s failed: %v", job.ID, err)
	} else if IsContextCanceled(err) {
		job.Status = "stopped"
		log.Printf("Job %s stopped", job.ID)
	} else {
		job.Status = "completed"
		log.Printf("Job %s completed", job.ID)
	}

	job.CancelFunc = nil
	job.Mutex.Unlock()

	storage.SaveJobs()
}

// SCRAPEURL SCRAPES A SINGLE URL WITH THE GIVEN JOB PARAMETERS
func ScrapeURL(ctx context.Context, job *models.ScrapingJob, url string, depth int) error {
	// CHECK JOB CONSTRAINTS
	job.Mutex.Lock()
	if depth > job.Rules.MaxDepth && job.Rules.MaxDepth > 0 {
		job.Mutex.Unlock()
		return nil
	}

	if job.Rules.MaxAssets > 0 && len(job.Assets) >= job.Rules.MaxAssets {
		job.Mutex.Unlock()
		return nil
	}
	job.Mutex.Unlock()

	// ADD DELAY TO AVOID DETECTION
	if job.Rules.RequestDelay > 0 {
		delay := job.Rules.RequestDelay
		if job.Rules.RandomizeDelay {
			delay = time.Duration(float64(delay) * (0.5 + rand.Float64()))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// CONTINUE AFTER DELAY
		}
	}

	log.Printf("Scraping URL: %s (depth: %d)", url, depth)

	// CHECK CONTEXT BEFORE PROCEEDING
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// CONTINUE PROCESSING
	}

	// TRY WITH CHROMEDP FIRST
	htmlContent, err := FetchWithChromedp(ctx, url, 5*time.Minute)

	// FALLBACK TO HTTP CLIENT IF CHROMEDP FAILS
	if err != nil {
		log.Printf("ChromeDP failed for %s: %v, falling back to HTTP client", url, err)
		httpCtx, httpCancel := context.WithTimeout(ctx, 2*time.Minute)
		defer httpCancel()
		htmlContent, err = FetchWithHTTP(httpCtx, url, job.Rules.UserAgent)
		if err != nil {
			log.Printf("HTTP client also failed for %s: %v", url, err)

			// DON'T FAIL THE ENTIRE JOB FOR A SINGLE URL TIMEOUT
			if depth == 0 {
				// REPORT FAILURE IF ITS BASE URL
				return err
			} else {
				// LOG AND CONTINUE ON CHILD URL FAILURE
				log.Printf("Skipping URL %s due to error: %v", url, err)
				return nil
			}
		}
	}

	// CHECK CONTENT LENGTH
	if len(htmlContent) < 100 {
		log.Printf("Warning: Content from %s seems too short (%d chars)", url, len(htmlContent))
	} else {
		log.Printf("Successfully scraped URL: %s (content length: %d)", url, len(htmlContent))
	}

	// PARSE HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML from %s: %v", url, err)
		if depth == 0 {
			return err
		}
		return nil
	}

	// PROCESS LINKS
	var links []string
	for _, selector := range job.Selectors {
		if selector.For == "links" {
			if selector.Type == "css" {
				doc.Find(selector.Value).Each(func(_ int, s *goquery.Selection) {
					if href, exists := s.Attr("href"); exists {
						absURL := MakeAbsoluteURL(url, href)
						if IsValidURL(absURL, job.Rules.IncludeURLPattern, job.Rules.ExcludeURLPattern) {
							links = append(links, absURL)
						}
					}
				})
			} else if selector.Type == "xpath" {
				// XPATH PROCESSING WOULD GO HERE
				// REQUIRES ADDITIONAL LIBRARY SUPPORT
			}
		}
	}

	// EXTRACT PAGE METADATA FOR USE WITH ASSETS
	metadata := ExtractPageMetadata(doc, job.Selectors, url)

	// FIND ASSETS
	for _, selector := range job.Selectors {
		if selector.For == "assets" {
			if selector.Type == "css" {
				doc.Find(selector.Value).Each(func(_ int, s *goquery.Selection) {
					// USE A GOROUTINE TO PROCESS ASSETS CONCURRENTLY
					go func(selection *goquery.Selection) {
						assets.ProcessAsset(job, selection, url, metadata)
					}(s)
				})
			}
		}
	}

	// RECURSIVELY SCRAPE LINKS
	for _, link := range links {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// CHECK IF WE ALREADY SCRAPED THIS URL
			job.Mutex.Lock()
			if job.CompletedAssets[link] {
				job.Mutex.Unlock()
				continue
			}
			job.CompletedAssets[link] = true
			job.Mutex.Unlock()

			// DON'T LET A FAILURE IN ONE LINK STOP THE WHOLE JOB
			if err := ScrapeURL(ctx, job, link, depth+1); err != nil {
				if IsContextCanceled(err) {
					return err
				}
				// LOG ERROR BUT CONTINUE WITH OTHER LINKS
				log.Printf("Error scraping link %s: %v", link, err)
			}
		}
	}

	if depth == 0 {
		var paginationSelector string
		// FIND PAGINATION SELECTOR IF ANY
		for _, selector := range job.Selectors {
			if selector.For == "pagination" {
				paginationSelector = selector.Value
				break
			}
		}

		if paginationSelector != "" {
			nextPageURL, err := ClickPaginationLink(ctx, url, paginationSelector)
			if err != nil {
				log.Printf("Pagination failed: %v", err)
			} else if nextPageURL != "" && nextPageURL != url {
				log.Printf("Found next page URL: %s", nextPageURL)

				// CLEAR THE COMPLETED ASSETS MAP TO ALLOW RESCANNING LINKS ON THE NEW PAGE
				job.Mutex.Lock()
				job.CompletedAssets = make(map[string]bool)
				job.CurrentPage++
				job.Mutex.Unlock()

				// RECURSIVELY SCRAPE THE NEXT PAGE
				return ScrapeURL(ctx, job, nextPageURL, 0)
			}
		}
	}

	return nil
}

// ISCONTEXTCANCELED CHECKS IF THE ERROR IS DUE TO CONTEXT CANCELLATION
func IsContextCanceled(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "context canceled") ||
		strings.Contains(err.Error(), "deadline exceeded")
}

// EXTRACTPAGEMETADATA EXTRACTS METADATA FROM THE PAGE USING SELECTORS
func ExtractPageMetadata(doc *goquery.Document, selectors []models.Selector, pageURL string) map[string]string {
	metadata := make(map[string]string)

	// DEFAULT VALUES
	metadata["title"] = doc.Find("title").Text()
	metadata["description"] = ""
	metadata["author"] = ""
	metadata["date"] = ""

	// CHECK META TAGS FOR COMMON METADATA
	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		nameLower := strings.ToLower(name)
		propertyLower := strings.ToLower(property)

		if nameLower == "description" || propertyLower == "og:description" {
			metadata["description"] = content
		} else if nameLower == "author" {
			metadata["author"] = content
		} else if nameLower == "date" || propertyLower == "article:published_time" {
			metadata["date"] = content
		}
	})

	// USE CUSTOM SELECTORS OVERRIDE DEFAULTS
	for _, selector := range selectors {
		if selector.Type == "css" {
			switch selector.For {
			case "title":
				if text := doc.Find(selector.Value).First().Text(); text != "" {
					metadata["title"] = strings.TrimSpace(text)
				}
			case "description":
				if text := doc.Find(selector.Value).First().Text(); text != "" {
					metadata["description"] = strings.TrimSpace(text)
				}
			case "author":
				if text := doc.Find(selector.Value).First().Text(); text != "" {
					metadata["author"] = strings.TrimSpace(text)
				}
			case "date":
				if text := doc.Find(selector.Value).First().Text(); text != "" {
					metadata["date"] = strings.TrimSpace(text)
				}
			}
		}
	}

	// ADD PAGE URL
	metadata["sourceUrl"] = pageURL

	return metadata
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

// ISVALIDURL CHECKS IF A URL MATCHES THE INCLUDE/EXCLUDE PATTERNS
func IsValidURL(url, includePattern, excludePattern string) bool {
	if includePattern != "" {
		matched, _ := regexp.MatchString(includePattern, url)
		if !matched {
			return false
		}
	}

	if excludePattern != "" {
		matched, _ := regexp.MatchString(excludePattern, url)
		if matched {
			return false
		}
	}

	return true
}
