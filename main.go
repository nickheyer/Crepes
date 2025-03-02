package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"
)

// MAIN TYPES
type ScrapingJob struct {
	ID              string             `json:"id"`
	BaseURL         string             `json:"baseUrl"`
	Selectors       []Selector         `json:"selectors"`
	Rules           ScrapingRules      `json:"rules"`
	Schedule        string             `json:"schedule"`
	Status          string             `json:"status"`
	LastRun         time.Time          `json:"lastRun"`
	NextRun         time.Time          `json:"nextRun"`
	Assets          []Asset            `json:"assets"`
	CompletedAssets map[string]bool    `json:"-"`
	Mutex           *sync.Mutex        `json:"-"`
	CancelFunc      context.CancelFunc `json:"-"`
}

type Selector struct {
	Type  string `json:"type"` // CSS OR XPATH
	Value string `json:"value"`
	For   string `json:"for"` // LINKS, ASSETS, METADATA
}

type ScrapingRules struct {
	MaxDepth          int           `json:"maxDepth"`
	MaxAssets         int           `json:"maxAssets"`
	IncludeURLPattern string        `json:"includeUrlPattern"`
	ExcludeURLPattern string        `json:"excludeUrlPattern"`
	Timeout           time.Duration `json:"timeout"`
	UserAgent         string        `json:"userAgent"`
	RequestDelay      time.Duration `json:"requestDelay"`
	RandomizeDelay    bool          `json:"randomizeDelay"`
}

type Asset struct {
	ID            string            `json:"id"`
	URL           string            `json:"url"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Type          string            `json:"type"`
	Size          int64             `json:"size"`
	LocalPath     string            `json:"localPath"`
	ThumbnailPath string            `json:"thumbnailPath"`
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

// GLOBAL VARIABLES
var (
	jobs       = make(map[string]*ScrapingJob)
	jobsMutex  sync.Mutex
	scheduler  *gocron.Scheduler
	appConfig  AppConfig
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:90.0) Gecko/20100101 Firefox/90.0",
	}
)

func init() {
	// LOAD CONFIG
	appConfig = AppConfig{
		Port:           8080,
		StoragePath:    "./storage",
		ThumbnailsPath: "./thumbnails",
		MaxConcurrent:  5,
		LogFile:        "scraper.log",
		DefaultTimeout: 60 * time.Second,
	}

	// ENSURE DIRECTORIES EXIST
	os.MkdirAll(appConfig.StoragePath, 0755)
	os.MkdirAll(appConfig.ThumbnailsPath, 0755)

	// SETUP SCHEDULER
	scheduler = gocron.NewScheduler(time.UTC)
	scheduler.StartAsync()

	// SETUP LOG FILE
	logFile, err := os.OpenFile(appConfig.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// CREATE NECESSARY FILES
	if err := createTemplates(); err != nil {
		log.Printf("Error creating templates: %v", err)
	}

	if err := createStaticFiles(); err != nil {
		log.Printf("Error creating static files: %v", err)
	}

	// TRY TO CONVERT SVG TO JPG
	convertSvgToJpg()

	// LOAD EXISTING JOBS
	loadJobs()
}

func main() {
	// SETUP ROUTER
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Static("/assets", appConfig.StoragePath)
	r.Static("/thumbnails", appConfig.ThumbnailsPath)
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// API ROUTES
	api := r.Group("/api")
	{
		api.POST("/jobs", createJob)
		api.GET("/jobs", listJobs)
		api.GET("/jobs/:id", getJob)
		api.DELETE("/jobs/:id", deleteJob)
		api.POST("/jobs/:id/start", startJob)
		api.POST("/jobs/:id/stop", stopJob)
		api.GET("/jobs/:id/assets", getJobAssets)
		api.GET("/assets/:id", getAsset)
	}

	// WEB INTERFACE ROUTES
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Web Scraper",
		})
	})

	r.GET("/jobs/:id", func(c *gin.Context) {
		jobID := c.Param("id")
		c.HTML(http.StatusOK, "job.html", gin.H{
			"jobID": jobID,
		})
	})

	// START SERVER
	log.Printf("Server starting on port %d", appConfig.Port)
	r.Run(fmt.Sprintf(":%d", appConfig.Port))
}

// HANDLERS
func createJob(c *gin.Context) {
	var job ScrapingJob
	if err := c.ShouldBindJSON(&job); err != nil {
		log.Printf("JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// VALIDATE JOB
	if job.BaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "baseUrl is required"})
		return
	}

	if len(job.Selectors) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one selector is required"})
		return
	}

	// SET DEFAULTS
	job.ID = uuid.New().String()
	job.Status = "idle"
	job.CompletedAssets = make(map[string]bool)
	job.Mutex = &sync.Mutex{}
	job.Assets = []Asset{}

	if job.Rules.UserAgent == "" {
		job.Rules.UserAgent = userAgents[0]
	}

	// HANDLE TIMEOUT AS INT64 OR FLOAT64 FROM JSON
	if job.Rules.Timeout == 0 {
		// Convert timeout from seconds to nanoseconds if it's a number value in JSON
		timeoutValue := c.Request.URL.Query().Get("timeout")
		if timeoutValue != "" {
			if seconds, err := strconv.ParseFloat(timeoutValue, 64); err == nil {
				job.Rules.Timeout = time.Duration(seconds * float64(time.Second))
			}
		}

		// If still zero, use default
		if job.Rules.Timeout == 0 {
			job.Rules.Timeout = appConfig.DefaultTimeout
		}
	}

	// Log the job being created for debugging
	jobBytes, _ := json.Marshal(job)
	log.Printf("Creating job: %s", string(jobBytes))

	// STORE JOB
	jobsMutex.Lock()
	jobs[job.ID] = &job
	jobsMutex.Unlock()

	// SCHEDULE JOB IF NEEDED
	if job.Schedule != "" {
		scheduleJob(jobs[job.ID])
	}

	saveJobs()
	c.JSON(http.StatusCreated, job)
}

func listJobs(c *gin.Context) {
	jobsMutex.Lock()
	defer jobsMutex.Unlock()

	jobsList := make([]*ScrapingJob, 0, len(jobs))
	for _, job := range jobs {
		jobsList = append(jobsList, job)
	}

	c.JSON(http.StatusOK, jobsList)
}

func getJob(c *gin.Context) {
	jobID := c.Param("id")

	jobsMutex.Lock()
	job, exists := jobs[jobID]
	jobsMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func deleteJob(c *gin.Context) {
	jobID := c.Param("id")

	jobsMutex.Lock()
	job, exists := jobs[jobID]
	if exists {
		// STOP RUNNING JOB
		if job.Status == "running" && job.CancelFunc != nil {
			job.CancelFunc()
		}
		delete(jobs, jobID)
	}
	jobsMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	saveJobs()
	c.JSON(http.StatusOK, gin.H{"message": "job deleted"})
}

func startJob(c *gin.Context) {
	jobID := c.Param("id")

	jobsMutex.Lock()
	job, exists := jobs[jobID]
	jobsMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// START JOB ONLY IF NOT RUNNING
	if job.Status == "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job already running"})
		return
	}

	go runJob(job)

	c.JSON(http.StatusOK, gin.H{"message": "job started"})
}

func stopJob(c *gin.Context) {
	jobID := c.Param("id")

	jobsMutex.Lock()
	job, exists := jobs[jobID]
	jobsMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// STOP JOB ONLY IF RUNNING
	if job.Status != "running" || job.CancelFunc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job not running"})
		return
	}

	job.CancelFunc()
	job.Status = "stopped"
	saveJobs()

	c.JSON(http.StatusOK, gin.H{"message": "job stopped"})
}

func getJobAssets(c *gin.Context) {
	jobID := c.Param("id")

	jobsMutex.Lock()
	job, exists := jobs[jobID]
	jobsMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job.Assets)
}

func getAsset(c *gin.Context) {
	assetID := c.Param("id")
	var foundAsset *Asset

	jobsMutex.Lock()
	for _, job := range jobs {
		for _, asset := range job.Assets {
			if asset.ID == assetID {
				foundAsset = &asset
				break
			}
		}
		if foundAsset != nil {
			break
		}
	}
	jobsMutex.Unlock()

	if foundAsset == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}

	c.JSON(http.StatusOK, foundAsset)
}

// JOB EXECUTION
func runJob(job *ScrapingJob) {
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

	saveJobs()
	log.Printf("Started job %s: %s", job.ID, job.BaseURL)

	// TEST SITE ACCESSIBILITY WITH SEPARATE TIMEOUT
	accessCtx, accessCancel := context.WithTimeout(ctx, 20*time.Second)
	defer accessCancel()
	if err := testSiteAccessibility(accessCtx, job.BaseURL); err != nil {
		log.Printf("WARNING: Site accessibility check failed: %v", err)
		// Continue anyway, but log the warning
	}

	// CREATE HEADLESS BROWSER CONTEXT - NO GLOBAL TIMEOUT
	browserCtx, browserCancel := createBrowserContext(ctx, job)
	defer browserCancel()

	// START SCRAPING WITHOUT GLOBAL TIMEOUT
	err := scrapeURL(browserCtx, job, job.BaseURL, 0)

	// UPDATE JOB STATUS
	job.Mutex.Lock()
	if err != nil && !isContextCanceled(err) {
		job.Status = "failed"
		log.Printf("Job %s failed: %v", job.ID, err)
	} else if isContextCanceled(err) {
		job.Status = "stopped"
		log.Printf("Job %s stopped", job.ID)
	} else {
		job.Status = "completed"
		log.Printf("Job %s completed", job.ID)
	}
	job.CancelFunc = nil
	job.Mutex.Unlock()

	saveJobs()
}

func isContextCanceled(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "context canceled") ||
		strings.Contains(err.Error(), "deadline exceeded")
}

func createBrowserContext(ctx context.Context, job *ScrapingJob) (context.Context, context.CancelFunc) {
	// CHECK CHROME PERMISSIONS AND ENVIRONMENT
	checkChromeEnvironment()

	// TRY WITH HEADLESS MODE FIRST
	log.Println("Attempting to create browser with headless mode")
	browserCtx, browserCancel, success := attemptBrowserCreation(ctx, job, true)

	// IF HEADLESS FAILS, TRY NON-HEADLESS AS FALLBACK
	if !success {
		log.Println("Headless mode failed, attempting with non-headless mode")
		browserCancel() // Cancel the failed browser
		browserCtx, browserCancel, success = attemptBrowserCreation(ctx, job, false)

		if !success {
			log.Println("WARNING: Both headless and non-headless modes failed, falling back to HTTP client")
			// RETURN A VALID CONTEXT EVEN IF BROWSER FAILED - WE'LL USE HTTP CLIENT INSTEAD
			return ctx, func() {
				browserCancel()
				log.Println("Cancelled fallback context")
			}
		}
	}

	return browserCtx, browserCancel
}

func checkChromeEnvironment() {
	// CHECK USER PERMISSIONS
	currentUser, err := user.Current()
	if err == nil {
		log.Printf("Running as user: %s (uid=%s, gid=%s)", currentUser.Username, currentUser.Uid, currentUser.Gid)
	}

	// CHECK IF RUNNING IN DOCKER/CONTAINER
	if _, err := os.Stat("/.dockerenv"); err == nil {
		log.Println("Running inside Docker container")
	}

	// CHECK CHROME VERSION (SILENTLY CONTINUE IF FAILS)
	chromePath := findChromePath()
	if chromePath != "" {
		cmd := exec.Command(chromePath, "--version")
		output, err := cmd.CombinedOutput()
		if err == nil {
			log.Printf("Chrome version: %s", strings.TrimSpace(string(output)))
		}
	} else {
		log.Println("Chrome not found in common locations")
	}

	// CHECK MEMORY AVAILABILITY
	var memInfo runtime.MemStats
	runtime.ReadMemStats(&memInfo)
	log.Printf("Available memory: %d MB", memInfo.Sys/1024/1024)
}

func attemptBrowserCreation(ctx context.Context, job *ScrapingJob, headless bool) (context.Context, context.CancelFunc, bool) {
	// BASE OPTIONS
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.WindowSize(1920, 1080),
		chromedp.Flag("no-sandbox", true), // Important for Docker/root

		// USER AGENT
		chromedp.UserAgent(job.Rules.UserAgent),
	}

	// CONDITIONALLY ADD HEADLESS MODE
	if headless {
		opts = append(opts,
			chromedp.Headless,
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
		)
	} else {
		// FOR NON-HEADLESS, USE MINIMAL WINDOW
		opts = append(opts,
			chromedp.Flag("window-position", "0,0"),
			chromedp.Flag("window-size", "1,1"),
		)
	}

	// CREATE DEBUG OUTPUT
	debugOutput := &bytes.Buffer{}
	opts = append(opts, chromedp.CombinedOutput(debugOutput))

	// CREATE ALLOCATOR WITH NO TIMEOUT
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)

	// CREATE BROWSER WITH LOGGING
	browserCtx, browserCancel := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(log.Printf),
		chromedp.WithErrorf(log.Printf),
	)

	// TEST THE BROWSER CONNECTION
	var version string
	testErr := chromedp.Run(browserCtx,
		chromedp.Evaluate(`navigator.userAgent`, &version),
	)

	if testErr != nil {
		log.Printf("Browser initialization test failed: %v", testErr)
		log.Printf("Chrome debug output: %s", debugOutput.String())

		// COMBINED CANCEL FOR FAILURE CASE
		combinedCancel := func() {
			browserCancel()
			allocCancel()
			log.Printf("Cancelled failed browser context")
		}

		return browserCtx, combinedCancel, false
	}

	log.Printf("Browser successfully initialized with user agent: %s", version)

	// COMBINED CANCEL FOR SUCCESS CASE
	combinedCancel := func() {
		log.Printf("Closing browser context")
		browserCancel()
		allocCancel()
	}

	return browserCtx, combinedCancel, true
}

func findChromePath() string {
	// Try common Chrome/Chromium locations based on OS
	var paths []string

	switch runtime.GOOS {
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Google\Chrome Dev\Application\chrome.exe`,
			`C:\Program Files\Google\Chrome Beta\Application\chrome.exe`,
			`C:\Program Files\Chromium\Application\chrome.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`, // Edge is Chromium-based
		}
	case "darwin": // macOS
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default: // Linux and others
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome-beta",
			"/usr/bin/google-chrome-unstable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
			"/usr/bin/microsoft-edge",
		}
	}

	// Check if any of the paths exist
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}

	// Try to find using exec.LookPath
	for _, browser := range []string{"chrome", "google-chrome", "chromium", "chromium-browser", "msedge"} {
		if path, err := exec.LookPath(browser); err == nil {
			return path
		}
	}

	return ""
}

func scrapeURL(ctx context.Context, job *ScrapingJob, url string, depth int) error {
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
			// Continue after delay
		}
	}

	log.Printf("Scraping URL: %s (depth: %d)", url, depth)

	// CHECK CONTEXT BEFORE PROCEEDING
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// TRY WITH CHROMEDP FIRST
	htmlContent, err := fetchWithChromedp(ctx, url, 5*time.Minute)

	// FALLBACK TO HTTP CLIENT IF CHROMEDP FAILS
	if err != nil {
		log.Printf("ChromeDP failed for %s: %v, falling back to HTTP client", url, err)
		httpCtx, httpCancel := context.WithTimeout(ctx, 2*time.Minute)
		defer httpCancel()
		htmlContent, err = fetchWithHTTP(httpCtx, url, job.Rules.UserAgent)
		if err != nil {
			log.Printf("HTTP client also failed for %s: %v", url, err)

			// KEY FIX: DON'T FAIL THE ENTIRE JOB FOR A SINGLE URL TIMEOUT
			// Check if this is a parent URL (depth 0) or a child URL
			if depth == 0 {
				// If it's the main URL and it failed, then we should report the failure
				return err
			} else {
				// If it's a child URL, just log and continue with other URLs
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
						absURL := makeAbsoluteURL(url, href)
						if isValidURL(absURL, job.Rules.IncludeURLPattern, job.Rules.ExcludeURLPattern) {
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

	// FIND ASSETS
	for _, selector := range job.Selectors {
		if selector.For == "assets" {
			if selector.Type == "css" {
				doc.Find(selector.Value).Each(func(_ int, s *goquery.Selection) {
					// Use a goroutine to process assets concurrently
					go func(selection *goquery.Selection) {
						processAsset(ctx, job, selection, url)
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
			if err := scrapeURL(ctx, job, link, depth+1); err != nil {
				if isContextCanceled(err) {
					return err
				}
				// Log error but continue with other links
				log.Printf("Error scraping link %s: %v", link, err)
			}
		}
	}

	return nil
}

func downloadAsset(ctx context.Context, job *ScrapingJob, asset *Asset) error {
	// CREATE DIRECTORY IF NOT EXISTS
	assetDir := filepath.Join(appConfig.StoragePath, job.ID)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return err
	}

	// DETERMINE FILE EXTENSION
	ext := filepath.Ext(asset.URL)
	if ext == "" {
		ext = getExtensionByType(asset.Type)
	}

	// CREATE FILE PATH
	fileName := fmt.Sprintf("%s%s", asset.ID, ext)
	filePath := filepath.Join(assetDir, fileName)
	asset.LocalPath = filepath.Join(job.ID, fileName)

	// CREATE SEPARATE BACKGROUND CONTEXT FOR DOWNLOAD
	baseCtx := context.Background()
	dlCtx, dlCancel := context.WithTimeout(baseCtx, 60*time.Minute) // 1 HOUR TIMEOUT FOR LARGE DOWNLOADS

	// Monitor the parent context for cancellation but not timeout
	go func() {
		select {
		case <-ctx.Done():
			// Only cancel if it's a manual cancellation, not a timeout
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				dlCancel()
			}
		case <-dlCtx.Done():
			// This will happen when dlCtx times out or is canceled
		}
	}()

	defer dlCancel()

	req, err := http.NewRequestWithContext(dlCtx, "GET", asset.URL, nil)
	if err != nil {
		return err
	}

	// SET HEADERS TO MIMIC BROWSER
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", job.BaseURL)

	// SEND REQUEST WITH RETRY LOGIC
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
			// Continue after waiting
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

func processAsset(ctx context.Context, job *ScrapingJob, selection *goquery.Selection, pageURL string) {
	// EXTRACT ASSET URL
	assetURL := ""

	// TRY COMMON ATTRIBUTES
	for _, attr := range []string{"src", "href", "data-src", "data-video", "data-media"} {
		if url, exists := selection.Attr(attr); exists && url != "" {
			assetURL = makeAbsoluteURL(pageURL, url)
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
	asset := Asset{
		ID:         uuid.New().String(),
		URL:        assetURL,
		Type:       getAssetType(assetURL),
		Metadata:   make(map[string]string),
		Downloaded: false,
	}

	// EXTRACT METADATA USING THE PROVIDED SELECTOR
	for _, selector := range job.Selectors {
		if selector.For == "metadata" {
			if selector.Type == "css" {
				// USE THE ACTUAL METADATA SELECTOR THE USER PROVIDED
				extractMetadata(selection, selector.Value, &asset)
			}
		}
	}

	// IF TITLE IS STILL EMPTY, USE FILENAME
	if asset.Title == "" {
		parts := strings.Split(asset.URL, "/")
		if len(parts) > 0 {
			fileName := parts[len(parts)-1]
			asset.Title = strings.TrimSpace(fileName)
		}
	}

	// DOWNLOAD ASSET
	err := downloadAsset(ctx, job, &asset)
	if err != nil {
		log.Printf("Error downloading asset %s: %v", assetURL, err)
		asset.Error = err.Error()
	} else {
		asset.Downloaded = true

		// GENERATE THUMBNAIL
		thumbnailPath, err := generateThumbnail(&asset)
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
	if len(job.Assets)%5 == 0 {
		saveJobs()
	}
}

func extractMetadata(selection *goquery.Selection, metadataSelector string, asset *Asset) {
	// USE ACTUAL USER-SPECIFIED SELECTOR FOR TITLE
	titleElement := selection.FindSelection(selection.Find(metadataSelector))
	if titleElement.Length() > 0 {
		title := titleElement.Text()
		if title != "" {
			asset.Title = strings.TrimSpace(title)
		}
	}

	// IF TITLE IS STILL EMPTY, EXTRACT FROM URL
	if asset.Title == "" {
		parts := strings.Split(asset.URL, "/")
		if len(parts) > 0 {
			fileName := parts[len(parts)-1]
			asset.Title = strings.TrimSpace(fileName)
		}
	}
}

func generateThumbnail(asset *Asset) (string, error) {
	// CREATE THUMBNAIL DIRECTORY
	thumbnailDir := filepath.Join(appConfig.ThumbnailsPath, filepath.Dir(asset.LocalPath))
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", err
	}

	thumbName := fmt.Sprintf("%s.jpg", asset.ID)
	thumbPath := filepath.Join(thumbnailDir, thumbName)
	relThumbPath := filepath.Join(filepath.Dir(asset.LocalPath), thumbName)

	// CHECK ASSET TYPE
	switch asset.Type {
	case "video":
		// EXTRACT FRAME USING FFMPEG
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-i", filepath.Join(appConfig.StoragePath, asset.LocalPath),
			"-ss", "00:00:05", // TAKE FRAME AT 5 SECONDS
			"-vframes", "1",
			"-vf", "scale=320:-1",
			"-y",
			thumbPath,
		)

		if err := cmd.Run(); err != nil {
			log.Printf("FFMPEG failed for video thumbnail: %v", err)
			return generateGenericThumbnail(asset)
		}

	case "image":
		// RESIZE IMAGE USING FFMPEG
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-i", filepath.Join(appConfig.StoragePath, asset.LocalPath),
			"-vf", "scale=320:-1",
			"-y",
			thumbPath,
		)

		if err := cmd.Run(); err != nil {
			log.Printf("FFMPEG failed for image thumbnail: %v", err)
			return generateGenericThumbnail(asset)
		}

	default:
		return generateGenericThumbnail(asset)
	}

	return relThumbPath, nil
}

func generateGenericThumbnail(asset *Asset) (string, error) {
	// CREATE GENERIC THUMBNAIL BASED ON FILE TYPE
	thumbnailDir := filepath.Join(appConfig.ThumbnailsPath, filepath.Dir(asset.LocalPath))
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", err
	}

	thumbName := fmt.Sprintf("%s.jpg", asset.ID)
	thumbPath := filepath.Join(thumbnailDir, thumbName)
	relThumbPath := filepath.Join(filepath.Dir(asset.LocalPath), thumbName)

	// COPY GENERIC ICON BASED ON TYPE
	genericPath := fmt.Sprintf("./static/icons/%s.jpg", asset.Type)
	if !fileExists(genericPath) {
		genericPath = "./static/icons/generic.jpg"
	}

	input, err := os.Open(genericPath)
	if err != nil {
		return "", err
	}
	defer input.Close()

	output, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	if err != nil {
		return "", err
	}

	return relThumbPath, nil
}

func fetchWithChromedp(ctx context.Context, url string, timeout time.Duration) (string, error) {
	// CREATE SEPARATE TIMEOUT CONTEXT JUST FOR THIS NAVIGATION
	navCtx, navCancel := context.WithTimeout(ctx, timeout)
	defer navCancel()

	var htmlContent string
	err := chromedp.Run(navCtx,
		chromedp.Navigate(url),
		//chromedp.Sleep(2*time.Second), // Give page time to load
		chromedp.ActionFunc(func(ctx context.Context) error {
			// CHECK IF PAGE IS LOADED
			var readyState string
			if err := chromedp.Evaluate(`document.readyState`, &readyState).Do(ctx); err != nil {
				return err
			}
			if readyState != "complete" {
				log.Printf("Page not fully loaded, state: %s", readyState)
				// WAIT A BIT MORE
				return chromedp.Sleep(3 * time.Second).Do(ctx)
			}
			return nil
		}),
		chromedp.OuterHTML("html", &htmlContent),
	)

	// IF CONTEXT DEADLINE EXCEEDED, THE JOB ITSELF SHOULDN'T STOP, JUST THIS NAVIGATION
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		return "", fmt.Errorf("navigation timeout after %v: %w", timeout, err)
	}

	return htmlContent, err
}

func fetchWithHTTP(ctx context.Context, url, userAgent string) (string, error) {
	// IMPORTANT: CREATE A SEPARATE CONTEXT THAT WON'T AFFECT THE PARENT CONTEXT
	baseCtx := context.Background()
	httpCtx, httpCancel := context.WithTimeout(baseCtx, 2*time.Minute)

	// Also create a way to cancel when the original context is canceled
	go func() {
		select {
		case <-ctx.Done():
			// Only cancel if it's a manual cancellation, not a timeout
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				httpCancel()
			}
		case <-httpCtx.Done():
			// This will happen when httpCtx times out or is canceled
		}
	}()

	defer httpCancel()

	// CREATE HTTP CLIENT WITHOUT TIMEOUT (WE USE CONTEXT TIMEOUT)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableCompression: false,
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
		},
		// NO CLIENT TIMEOUT - WE USE CONTEXT TIMEOUT
	}

	// CREATE REQUEST WITH CONTEXT
	req, err := http.NewRequestWithContext(httpCtx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	// ADD BROWSER-LIKE HEADERS
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	// SEND REQUEST
	resp, err := client.Do(req)
	if err != nil {
		// CONVERT CONTEXT DEADLINE TO CUSTOM ERROR TO AVOID PROPAGATION
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("HTTP request timeout after 2 minutes: operation took too long")
		}
		return "", err
	}
	defer resp.Body.Close()

	// CHECK RESPONSE STATUS
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status: %d %s", resp.StatusCode, resp.Status)
	}

	// READ BODY WITH SIZE LIMIT TO PREVENT MEMORY ISSUES
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB LIMIT
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

// UTILS
func makeAbsoluteURL(base, ref string) string {
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

func isValidURL(url, includePattern, excludePattern string) bool {
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

func getAssetType(url string) string {
	ext := strings.ToLower(filepath.Ext(url))

	videoExts := []string{".mp4", ".webm", ".mkv", ".avi", ".mov", ".flv", ".m4v", ".mpg", ".mpeg", ".ts"}
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".ico"}
	audioExts := []string{".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma"}
	docExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".rtf", ".csv"}

	for _, vExt := range videoExts {
		if ext == vExt {
			return "video"
		}
	}

	for _, iExt := range imageExts {
		if ext == iExt {
			return "image"
		}
	}

	for _, aExt := range audioExts {
		if ext == aExt {
			return "audio"
		}
	}

	for _, dExt := range docExts {
		if ext == dExt {
			return "document"
		}
	}

	// TRY TO GUESS FROM URL PATTERNS
	urlLower := strings.ToLower(url)
	if strings.Contains(urlLower, "video") ||
		strings.Contains(urlLower, "movie") ||
		strings.Contains(urlLower, "watch") {
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

	return "unknown"
}

func getExtensionByType(assetType string) string {
	switch assetType {
	case "video":
		return ".mp4"
	case "image":
		return ".jpg"
	case "audio":
		return ".mp3"
	case "document":
		return ".pdf"
	default:
		return ".bin"
	}
}

func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

func scheduleJob(job *ScrapingJob) {
	_, err := scheduler.Cron(job.Schedule).Do(func() {
		// CHECK IF JOB IS ALREADY RUNNING
		job.Mutex.Lock()
		if job.Status == "running" {
			job.Mutex.Unlock()
			return
		}
		job.Mutex.Unlock()

		go runJob(job)
	})

	if err != nil {
		log.Printf("Error scheduling job %s: %v", job.ID, err)
	}
}

func saveJobs() {
	// SERIALIZE JOBS WITHOUT MUTEX AND CANCELFUNC
	jobsMutex.Lock()
	defer jobsMutex.Unlock()

	serializedJobs := make(map[string]ScrapingJob)
	for id, job := range jobs {
		// CREATE A COPY WITHOUT MUTEX AND CANCELFUNC
		serializedJob := *job
		serializedJob.Mutex = nil
		serializedJob.CancelFunc = nil
		serializedJob.CompletedAssets = nil
		serializedJobs[id] = serializedJob
	}

	// SAVE TO FILE
	data, err := json.MarshalIndent(serializedJobs, "", "  ")
	if err != nil {
		log.Printf("Error serializing jobs: %v", err)
		return
	}

	// WRITE TO TEMP FILE FIRST, THEN RENAME FOR ATOMIC OPERATION
	tempFile := "jobs.json.tmp"
	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		log.Printf("Error saving jobs to temp file: %v", err)
		return
	}

	err = os.Rename(tempFile, "jobs.json")
	if err != nil {
		log.Printf("Error renaming temp file to jobs.json: %v", err)
	}
}

func loadJobs() {
	data, err := os.ReadFile("jobs.json")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading jobs: %v", err)
		}
		return
	}

	var serializedJobs map[string]ScrapingJob
	err = json.Unmarshal(data, &serializedJobs)
	if err != nil {
		log.Printf("Error parsing jobs.json: %v", err)
		return
	}

	// RESTORE JOBS WITH PROPER MUTEX AND MAPS
	jobsMutex.Lock()
	defer jobsMutex.Unlock()

	for id, serializedJob := range serializedJobs {
		// CREATE A NEW JOB WITH PROPER MUTEX
		loadedJob := serializedJob
		loadedJob.Mutex = &sync.Mutex{} // CREATE NEW MUTEX
		loadedJob.CancelFunc = nil
		loadedJob.CompletedAssets = make(map[string]bool)

		// STORE AS POINTER
		jobs[id] = &loadedJob

		// RESCHEDULE JOB IF NEEDED
		if loadedJob.Schedule != "" && loadedJob.Status != "running" {
			scheduleJob(jobs[id]) // PASS POINTER
		}
	}

	log.Printf("Loaded %d jobs", len(jobs))
}

func testSiteAccessibility(ctx context.Context, url string) error {
	// First test with a simple HTTP request
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	// Create a request with common browser headers
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		return fmt.Errorf("site returned error status: %d %s", resp.StatusCode, resp.Status)
	}

	// Read a bit of the body to verify content
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	// Check if it's likely a bot protection page
	bodyLower := strings.ToLower(string(bodyBytes))
	if strings.Contains(bodyLower, "captcha") ||
		strings.Contains(bodyLower, "cloudflare") && strings.Contains(bodyLower, "security") ||
		strings.Contains(bodyLower, "ddos") ||
		strings.Contains(bodyLower, "checking your browser") {
		return fmt.Errorf("site appears to have bot protection active")
	}

	log.Printf("Site %s is accessible via HTTP with status %d", url, resp.StatusCode)
	return nil
}

// TEMPLATES
func createTemplates() error {
	// ENSURE TEMPLATES DIRECTORY EXISTS
	if err := os.MkdirAll("templates", 0755); err != nil {
		return err
	}

	// CREATE INDEX.HTML
	indexHTML := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Web Scraper - {{.title}}</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.10.2/dist/cdn.min.js" defer></script>
</head>
<body class="bg-gray-900 text-white" x-data="{ 
    jobs: [],
    newJob: {
        baseUrl: '',
        selectors: [
            { type: 'css', value: '', for: 'links' },
            { type: 'css', value: '', for: 'assets' },
            { type: 'css', value: '', for: 'metadata' }
        ],
        rules: {
            maxDepth: 3,
            maxAssets: 100,
            includeUrlPattern: '',
            excludeUrlPattern: '',
            timeout: 60,
            requestDelay: 2000,
            randomizeDelay: true
        },
        schedule: ''
    },
    showNewJobForm: false,
    
    fetchJobs() {
        fetch('/api/jobs')
            .then(response => response.json())
            .then(data => {
                this.jobs = data;
            })
            .catch(error => console.error('Error fetching jobs:', error));
    },
    
	createJob() {
		// CLONE THE DATA TO AVOID MODIFYING THE ORIGINAL
		const jobData = JSON.parse(JSON.stringify(this.newJob));
		
		// CONVERT TIMEOUT FROM SECONDS TO NANOSECONDS FOR BACKEND
		// Handle potential string vs number type issues
		if (jobData.rules.timeout) {
			// Make sure it's a number
			const timeoutSeconds = Number(jobData.rules.timeout);
			if (!isNaN(timeoutSeconds)) {
				jobData.rules.timeout = timeoutSeconds * 1000000000;  // Convert to nanoseconds
			}
		}
		
		// MAKE SURE TO STRINGIFY ANY NUMBERS THAT MIGHT BE IN STRING FORMAT
		if (jobData.rules.requestDelay) {
			jobData.rules.requestDelay = Number(jobData.rules.requestDelay);
		}
		if (jobData.rules.maxDepth) {
			jobData.rules.maxDepth = Number(jobData.rules.maxDepth);
		}
		if (jobData.rules.maxAssets) {
			jobData.rules.maxAssets = Number(jobData.rules.maxAssets);
		}
		
		fetch('/api/jobs', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(jobData)
		})
		.then(response => {
			if (!response.ok) {
				return response.json().then(err => {
					throw new Error('Server error: ' + (err.error || 'Unknown error'));
				});
			}
			return response.json();
		})
		.then(data => {
			this.fetchJobs();
			this.showNewJobForm = false;
			this.resetNewJobForm();
		})
		.catch(error => {
			console.error('Error creating job:', error);
			alert('Error creating job: ' + error.message);
		});
	},
    
    startJob(id) {
        fetch('/api/jobs/' + id + '/start', { method: 'POST' })
            .then(response => response.json())
            .then(data => this.fetchJobs())
            .catch(error => console.error('Error starting job:', error));
    },
    
    stopJob(id) {
        fetch('/api/jobs/' + id + '/stop', { method: 'POST' })
            .then(response => response.json())
            .then(data => this.fetchJobs())
            .catch(error => console.error('Error stopping job:', error));
    },
    
    deleteJob(id) {
        if (confirm('Are you sure you want to delete this job?')) {
            fetch('/api/jobs/' + id, { method: 'DELETE' })
                .then(response => response.json())
                .then(data => this.fetchJobs())
                .catch(error => console.error('Error deleting job:', error));
        }
    },
    
    resetNewJobForm() {
        this.newJob = {
            baseUrl: '',
            selectors: [
                { type: 'css', value: '', for: 'links' },
                { type: 'css', value: '', for: 'assets' },
                { type: 'css', value: '', for: 'metadata' }
            ],
            rules: {
                maxDepth: 3,
                maxAssets: 100,
                includeUrlPattern: '',
                excludeUrlPattern: '',
                timeout: 60,
                requestDelay: 2000,
                randomizeDelay: true
            },
            schedule: ''
        };
    },
    
    addSelector() {
        this.newJob.selectors.push({ type: 'css', value: '', for: 'links' });
    },
    
    removeSelector(index) {
        this.newJob.selectors.splice(index, 1);
    },
    
    formatDate(dateString) {
        if (!dateString) return 'N/A';
        return new Date(dateString).toLocaleString();
    },
    
    formatStatus(status) {
        const colors = {
            'idle': 'bg-gray-500',
            'running': 'bg-green-500',
            'completed': 'bg-blue-500',
            'failed': 'bg-red-500',
            'stopped': 'bg-yellow-500'
        };
        return colors[status] || 'bg-gray-500';
    }
}" 
x-init="fetchJobs(); setInterval(fetchJobs, 5000)">

    <!-- Header -->
    <header class="bg-gray-800 shadow">
        <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8 flex justify-between items-center">
            <h1 class="text-3xl font-bold">Web Scraper</h1>
            <button 
                @click="showNewJobForm = true" 
                class="px-4 py-2 bg-indigo-600 rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
                New Job
            </button>
        </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
        <!-- Jobs list -->
        <div class="bg-gray-800 shadow rounded-lg p-6 mb-6">
            <h2 class="text-xl font-semibold mb-4">Jobs</h2>
            <div class="overflow-x-auto">
                <table class="min-w-full divide-y divide-gray-700">
                    <thead>
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">ID</th>
                            <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Base URL</th>
                            <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Status</th>
                            <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Last Run</th>
                            <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Assets</th>
                            <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Actions</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-700">
                        <template x-for="job in jobs" :key="job.id">
                            <tr>
                                <td class="px-6 py-4 whitespace-nowrap">
                                    <a :href="'/jobs/' + job.id" class="text-indigo-400 hover:text-indigo-300" x-text="job.id.substring(0, 8) + '...'"></a>
                                </td>
                                <td class="px-6 py-4 whitespace-nowrap" x-text="job.baseUrl"></td>
                                <td class="px-6 py-4 whitespace-nowrap">
                                    <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full" 
                                        :class="formatStatus(job.status)" 
                                        x-text="job.status">
                                    </span>
                                </td>
                                <td class="px-6 py-4 whitespace-nowrap" x-text="formatDate(job.lastRun)"></td>
                                <td class="px-6 py-4 whitespace-nowrap" x-text="job.assets ? job.assets.length : 0"></td>
                                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                    <button 
                                        x-show="job.status !== 'running'"
                                        @click="startJob(job.id)" 
                                        class="text-green-400 hover:text-green-300 mr-3">
                                        Start
                                    </button>
                                    <button 
                                        x-show="job.status === 'running'"
                                        @click="stopJob(job.id)" 
                                        class="text-yellow-400 hover:text-yellow-300 mr-3">
                                        Stop
                                    </button>
                                    <button 
                                        @click="deleteJob(job.id)" 
                                        class="text-red-400 hover:text-red-300">
                                        Delete
                                    </button>
                                </td>
                            </tr>
                        </template>
<tr x-show="jobs.length === 0">
                            <td colspan="6" class="px-6 py-4 text-center text-gray-400">No jobs found. Create a new job to get started.</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </main>

    <!-- New Job Modal -->
    <div x-show="showNewJobForm" class="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center p-4 z-50">
        <div class="bg-gray-800 rounded-lg max-w-4xl w-full max-h-screen overflow-y-auto" @click.away="showNewJobForm = false">
            <div class="px-6 py-4 border-b border-gray-700">
                <h3 class="text-lg font-medium">Create New Scraping Job</h3>
            </div>
            <div class="p-6">
                <form @submit.prevent="createJob">
                    <!-- Base URL -->
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Base URL</label>
                        <input type="url" x-model="newJob.baseUrl" required class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                    </div>
                    
                    <!-- Selectors -->
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Selectors</label>
                        <div class="space-y-3">
                            <template x-for="(selector, index) in newJob.selectors" :key="index">
                                <div class="flex space-x-2">
                                    <select x-model="selector.type" class="px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                                        <option value="css">CSS</option>
                                        <option value="xpath">XPath</option>
                                    </select>
                                    <select x-model="selector.for" class="px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                                        <option value="links">Links</option>
                                        <option value="assets">Assets</option>
                                        <option value="metadata">Metadata</option>
                                    </select>
                                    <input type="text" x-model="selector.value" placeholder="Selector value" class="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                                    <button type="button" @click="removeSelector(index)" class="px-3 py-2 bg-red-600 text-white rounded-md hover:bg-red-700">-</button>
                                </div>
                            </template>
                            <button type="button" @click="addSelector" class="px-3 py-2 bg-green-600 text-white rounded-md hover:bg-green-700">+ Add Selector</button>
                        </div>
                    </div>
                    
                    <!-- Rules -->
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-3">Scraping Rules</label>
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div>
                                <label class="block text-xs mb-1">Max Depth</label>
                                <input type="number" x-model="newJob.rules.maxDepth" min="0" class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                            </div>
                            <div>
                                <label class="block text-xs mb-1">Max Assets</label>
                                <input type="number" x-model="newJob.rules.maxAssets" min="0" class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                            </div>
                            <div>
                                <label class="block text-xs mb-1">Include URL Pattern (regex)</label>
                                <input type="text" x-model="newJob.rules.includeUrlPattern" class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                            </div>
                            <div>
                                <label class="block text-xs mb-1">Exclude URL Pattern (regex)</label>
                                <input type="text" x-model="newJob.rules.excludeUrlPattern" class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                            </div>
                            <div>
                                <label class="block text-xs mb-1">Timeout (seconds)</label>
                                <input type="number" x-model="newJob.rules.timeout" min="0" class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                            </div>
                            <div>
                                <label class="block text-xs mb-1">Request Delay (ms)</label>
                                <input type="number" x-model="newJob.rules.requestDelay" min="0" class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                            </div>
                            <div class="flex items-center">
                                <input type="checkbox" x-model="newJob.rules.randomizeDelay" id="randomizeDelay" class="mr-2">
                                <label for="randomizeDelay" class="text-xs">Randomize Delay</label>
                            </div>
                        </div>
                    </div>
                    
                    <!-- Schedule -->
                    <div class="mb-6">
                        <label class="block text-sm font-medium mb-1">Schedule (Cron Expression)</label>
                        <input type="text" x-model="newJob.schedule" placeholder="e.g. */10 * * * *" class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                        <p class="mt-1 text-xs text-gray-400">Leave empty for manual execution only</p>
                    </div>
                    
                    <!-- Buttons -->
                    <div class="flex justify-end space-x-3">
                        <button type="button" @click="showNewJobForm = false" class="px-4 py-2 bg-gray-600 rounded-md hover:bg-gray-700">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-indigo-600 rounded-md hover:bg-indigo-700">Create Job</button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</body>
</html>
`

	// CREATE JOB.HTML
	jobHTML := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Job Details - Web Scraper</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.10.2/dist/cdn.min.js" defer></script>
</head>
<body class="bg-gray-900 text-white" x-data="{ 
    jobId: '{{.jobID}}',
    job: null,
    assets: [],
    activeTab: 'assets',
    
    init() {
        this.fetchJob();
        setInterval(this.fetchJob, 5000);
    },
    
    fetchJob() {
        if (!this.jobId) return;
        
        fetch('/api/jobs/' + this.jobId)
            .then(response => {
                if (!response.ok) throw new Error('Job not found');
                return response.json();
            })
            .then(data => {
                this.job = data;
                this.fetchAssets();
            })
            .catch(error => {
                console.error('Error fetching job:', error);
                this.job = null;
            });
    },
    
    fetchAssets() {
        if (!this.jobId) return;
        
        fetch('/api/jobs/' + this.jobId + '/assets')
            .then(response => {
                if (!response.ok) throw new Error('Assets not found');
                return response.json();
            })
            .then(data => {
                this.assets = data;
            })
            .catch(error => {
                console.error('Error fetching assets:', error);
                this.assets = [];
            });
    },
    
    startJob() {
        fetch('/api/jobs/' + this.jobId + '/start', { method: 'POST' })
            .then(response => response.json())
            .then(data => this.fetchJob())
            .catch(error => console.error('Error starting job:', error));
    },
    
    stopJob() {
        fetch('/api/jobs/' + this.jobId + '/stop', { method: 'POST' })
            .then(response => response.json())
            .then(data => this.fetchJob())
            .catch(error => console.error('Error stopping job:', error));
    },
    
    deleteJob() {
        if (confirm('Are you sure you want to delete this job?')) {
            fetch('/api/jobs/' + this.jobId, { method: 'DELETE' })
                .then(response => response.json())
                .then(data => window.location.href = '/')
                .catch(error => console.error('Error deleting job:', error));
        }
    },
    
    formatDate(dateString) {
        if (!dateString) return 'N/A';
        return new Date(dateString).toLocaleString();
    },
    
    formatStatus(status) {
        const colors = {
            'idle': 'bg-gray-500',
            'running': 'bg-green-500',
            'completed': 'bg-blue-500',
            'failed': 'bg-red-500',
            'stopped': 'bg-yellow-500'
        };
        return colors[status] || 'bg-gray-500';
    },
    
    formatSize(bytes) {
        if (!bytes) return 'Unknown';
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + sizes[i];
    },
    
    getAssetIcon(type) {
        const icons = {
            'video': '🎬',
            'image': '🖼️',
            'audio': '🔊',
            'document': '📄',
            'unknown': '❓'
        };
        return icons[type] || icons['unknown'];
    },
    
    getAssetUrl(asset) {
        return '/assets/' + asset.localPath;
    },
    
    getThumbnailUrl(asset) {
        return asset.thumbnailPath ? '/thumbnails/' + asset.thumbnailPath : '/static/icons/generic.jpg';
    },
    
    openAsset(asset) {
        if (asset.type === 'video' || asset.type === 'audio' || asset.type === 'image') {
            window.open(this.getAssetUrl(asset), '_blank');
        } else {
            const link = document.createElement('a');
            link.href = this.getAssetUrl(asset);
            link.download = asset.title || 'download';
            link.click();
        }
    }
}">

    <!-- Header -->
    <header class="bg-gray-800 shadow">
        <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8 flex justify-between items-center">
            <div class="flex items-center">
                <a href="/" class="text-indigo-400 hover:text-indigo-300 mr-4">
                    &larr; Back
                </a>
                <h1 class="text-3xl font-bold">Job Details</h1>
            </div>
            <div x-show="job" class="flex space-x-3">
                <button 
                    x-show="job && job.status !== 'running'"
                    @click="startJob()" 
                    class="px-4 py-2 bg-green-600 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500"
                >
                    Start
                </button>
                <button 
                    x-show="job && job.status === 'running'"
                    @click="stopJob()" 
                    class="px-4 py-2 bg-yellow-600 rounded-md hover:bg-yellow-700 focus:outline-none focus:ring-2 focus:ring-yellow-500"
                >
                    Stop
                </button>
                <button 
                    @click="deleteJob()" 
                    class="px-4 py-2 bg-red-600 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
                >
                    Delete
                </button>
            </div>
        </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
        <div x-show="!job" class="bg-gray-800 shadow rounded-lg p-6 mb-6 text-center">
            <p class="text-gray-400">Loading job details or job not found...</p>
        </div>
        
        <!-- Job details -->
        <div class="bg-gray-800 shadow rounded-lg p-6 mb-6" x-show="job">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <h2 class="text-xl font-semibold mb-4">Job Information</h2>
                    <div class="space-y-3">
                        <div>
                            <span class="text-gray-400">ID:</span>
                            <span x-text="job?.id"></span>
                        </div>
                        <div>
                            <span class="text-gray-400">Base URL:</span>
                            <a :href="job?.baseUrl" target="_blank" class="text-indigo-400 hover:text-indigo-300" x-text="job?.baseUrl"></a>
                        </div>
                        <div>
                            <span class="text-gray-400">Status:</span>
                            <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full" 
                                :class="formatStatus(job?.status)" 
                                x-text="job?.status">
                            </span>
                        </div>
                        <div>
                            <span class="text-gray-400">Last Run:</span>
                            <span x-text="formatDate(job?.lastRun)"></span>
                        </div>
                        <div>
                            <span class="text-gray-400">Next Run:</span>
                            <span x-text="job?.schedule ? formatDate(job?.nextRun) : 'Manual execution only'"></span>
                        </div>
                    </div>
                </div>
                <div>
                    <h2 class="text-xl font-semibold mb-4">Job Configuration</h2>
                    <div class="space-y-3">
                        <div>
                            <span class="text-gray-400">Schedule:</span>
                            <span x-text="job?.schedule || 'None'"></span>
                        </div>
                        <div>
                            <span class="text-gray-400">Max Depth:</span>
                            <span x-text="job?.rules?.maxDepth || 'Unlimited'"></span>
                        </div>
                        <div>
                            <span class="text-gray-400">Max Assets:</span>
                            <span x-text="job?.rules?.maxAssets || 'Unlimited'"></span>
                        </div>
                        <div>
                            <span class="text-gray-400">Request Delay:</span>
                            <span x-text="(job?.rules?.requestDelay || 0) + 'ms' + (job?.rules?.randomizeDelay ? ' (randomized)' : '')"></span>
                        </div>
                        <div>
                            <span class="text-gray-400">Timeout:</span>
                            <span x-text="job?.rules?.timeout ? (job?.rules?.timeout / 1000000000) + 's' : 'Default'"></span>
                        </div>
                        <div>
                            <span class="text-gray-400">Total Assets:</span>
                            <span x-text="assets.length || 0"></span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- Tabs -->
        <div class="border-b border-gray-700 mb-6" x-show="job">
            <nav class="-mb-px flex space-x-8">
                <button 
                    @click="activeTab = 'assets'" 
                    :class="activeTab === 'assets' ? 'border-indigo-400 text-indigo-400' : 'border-transparent text-gray-400 hover:text-gray-300 hover:border-gray-300'"
                    class="py-4 px-1 border-b-2 font-medium text-sm"
                >
                    Assets (<span x-text="assets.length"></span>)
                </button>
                <button 
                    @click="activeTab = 'selectors'" 
                    :class="activeTab === 'selectors' ? 'border-indigo-400 text-indigo-400' : 'border-transparent text-gray-400 hover:text-gray-300 hover:border-gray-300'"
                    class="py-4 px-1 border-b-2 font-medium text-sm"
                >
                    Selectors
                </button>
            </nav>
        </div>
        
        <!-- Assets Tab -->
        <div x-show="activeTab === 'assets' && job">
            <!-- Assets Grid -->
            <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
                <template x-for="asset in assets" :key="asset.id">
                    <div class="bg-gray-800 rounded-lg shadow overflow-hidden">
                        <div class="h-48 bg-gray-700 overflow-hidden relative">
                            <img :src="getThumbnailUrl(asset)" class="w-full h-full object-cover">
                            <span class="absolute top-2 right-2 px-2 py-1 rounded-full text-xs font-bold bg-gray-900 text-white" x-text="getAssetIcon(asset.type) + ' ' + asset.type"></span>
                        </div>
                        <div class="p-4">
                            <h3 class="font-medium text-lg truncate" x-text="asset.title || 'Untitled'"></h3>
                            <p class="text-gray-400 text-sm truncate" x-text="asset.description || 'No description'"></p>
                            <div class="mt-3 flex justify-between text-sm">
                                <span class="text-gray-400" x-text="formatSize(asset.size)"></span>
                                <button 
                                    @click="openAsset(asset)" 
                                    class="text-indigo-400 hover:text-indigo-300">
                                    View
                                </button>
                            </div>
                        </div>
                    </div>
                </template>
                <div x-show="assets.length === 0" class="bg-gray-800 rounded-lg shadow p-6 text-center text-gray-400 col-span-full">
                    No assets found yet. Start the job to begin scraping.
                </div>
            </div>
        </div>
        
        <!-- Selectors Tab -->
        <div x-show="activeTab === 'selectors' && job">
            <div class="bg-gray-800 shadow rounded-lg p-6">
                <template x-if="job && job.selectors">
                    <div>
                        <h3 class="text-lg font-medium mb-4">Job Selectors</h3>
                        <div class="overflow-x-auto">
                            <table class="min-w-full divide-y divide-gray-700">
                                <thead>
                                    <tr>
                                        <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Type</th>
                                        <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Purpose</th>
                                        <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Value</th>
                                    </tr>
                                </thead>
                                <tbody class="divide-y divide-gray-700">
                                    <template x-for="(selector, index) in job.selectors" :key="index">
                                        <tr>
                                            <td class="px-6 py-4 whitespace-nowrap" x-text="selector.type"></td>
                                            <td class="px-6 py-4 whitespace-nowrap" x-text="selector.for"></td>
                                            <td class="px-6 py-4" x-text="selector.value"></td>
                                        </tr>
                                    </template>
                                </tbody>
                            </table>
                        </div>
                    </div>
                </template>
                <div x-show="!job || !job.selectors || job.selectors.length === 0" class="text-center text-gray-400">
                    No selectors defined for this job.
                </div>
            </div>
        </div>
    </main>
</body>
</html>
`

	// WRITE INDEX.HTML
	if err := os.WriteFile("templates/index.html", []byte(indexHTML), 0644); err != nil {
		return err
	}

	// WRITE JOB.HTML
	if err := os.WriteFile("templates/job.html", []byte(jobHTML), 0644); err != nil {
		return err
	}

	return nil
}

// STATIC DIRECTORIES AND FILES
func createStaticFiles() error {
	// CREATE STATIC DIRECTORY
	if err := os.MkdirAll("static/icons", 0755); err != nil {
		return err
	}

	// CREATE GENERIC ICON
	genericIcon := `
<svg xmlns="http://www.w3.org/2000/svg" width="320" height="320" viewBox="0 0 320 320">
  <rect width="320" height="320" fill="#2d3748"/>
  <text x="160" y="160" font-family="Arial" font-size="120" text-anchor="middle" dominant-baseline="middle" fill="#a0aec0">?</text>
</svg>
`
	if err := os.WriteFile("static/icons/generic.svg", []byte(genericIcon), 0644); err != nil {
		return err
	}

	// CREATE ICONS FOR EACH TYPE
	icons := map[string]string{
		"video": `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="320" viewBox="0 0 320 320">
  <rect width="320" height="320" fill="#2d3748"/>
  <path d="M160,80 L240,160 L160,240 L80,160 Z" fill="#90cdf4"/>
</svg>`,
		"image": `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="320" viewBox="0 0 320 320">
  <rect width="320" height="320" fill="#2d3748"/>
  <circle cx="120" cy="120" r="20" fill="#f6ad55"/>
  <path d="M80,200 L120,160 L160,200 L220,120 L240,140 L240,220 L80,220 Z" fill="#68d391"/>
</svg>`,
		"audio": `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="320" viewBox="0 0 320 320">
  <rect width="320" height="320" fill="#2d3748"/>
  <path d="M140,80 L140,180 L120,180 L120,140 L100,140 L100,180 L80,180 L80,140 L140,80 Z" fill="#fc8181"/>
  <path d="M160,120 A60,60 0 0 1 160,240 A60,60 0 0 1 160,120" fill="#a0aec0" stroke="#a0aec0" stroke-width="10" fill-opacity="0"/>
  <path d="M190,90 A100,100 0 0 1 190,230 A100,100 0 0 1 190,90" fill="#a0aec0" stroke="#a0aec0" stroke-width="10" fill-opacity="0"/>
</svg>`,
		"document": `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="320" viewBox="0 0 320 320">
  <rect width="320" height="320" fill="#2d3748"/>
  <path d="M100,60 L220,60 L220,260 L100,260 Z" fill="#4a5568"/>
  <path d="M120,100 L200,100 L200,120 L120,120 Z" fill="#a0aec0"/>
  <path d="M120,140 L200,140 L200,160 L120,160 Z" fill="#a0aec0"/>
  <path d="M120,180 L200,180 L200,200 L120,200 Z" fill="#a0aec0"/>
  <path d="M120,220 L160,220 L160,240 L120,240 Z" fill="#a0aec0"/>
</svg>`,
	}

	// WRITE EACH ICON
	for iconType, iconSVG := range icons {
		if err := os.WriteFile(fmt.Sprintf("static/icons/%s.svg", iconType), []byte(iconSVG), 0644); err != nil {
			return err
		}
	}

	return nil
}

// CONVERT SVG TO JPG FOR THUMBNAILS
func convertSvgToJpg() error {
	// FIND ALL SVG FILES
	svgFiles, err := filepath.Glob("static/icons/*.svg")
	if err != nil {
		return err
	}

	for _, svgFile := range svgFiles {
		// GET OUTPUT FILENAME
		jpgFile := strings.TrimSuffix(svgFile, ".svg") + ".jpg"

		// USE IMAGEMAGICK TO CONVERT
		cmd := exec.Command(
			"convert",
			svgFile,
			jpgFile,
		)

		if err := cmd.Run(); err != nil {
			log.Printf("Warning: Could not convert %s to JPG: %v", svgFile, err)
			// CREATE A FALLBACK JPG IF CONVERT FAILS
			createFallbackJpg(jpgFile)
		}
	}

	return nil
}

// FALLBACK JPG IF IMAGEMAGICK IS NOT AVAILABLE
func createFallbackJpg(jpgPath string) {
	// CREATE A SIMPLE 320x320 BLACK IMAGE AS FALLBACK
	img := make([]byte, 320*320*3)
	for i := range img {
		img[i] = 0 // BLACK
	}

	// WRITE AS RAW JPG (NOT IDEAL BUT WORKS AS EMERGENCY FALLBACK)
	os.WriteFile(jpgPath, img, 0644)
}

// INIT
func init() {
	// CREATE NECESSARY FILES
	if err := createTemplates(); err != nil {
		log.Fatalf("Error creating templates: %v", err)
	}

	if err := createStaticFiles(); err != nil {
		log.Fatalf("Error creating static files: %v", err)
	}

	// TRY TO CONVERT SVG TO JPG
	convertSvgToJpg()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
