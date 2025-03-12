package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	stealth "github.com/jonfriesen/playwright-go-stealth"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"
	"github.com/playwright-community/playwright-go"
	"gorm.io/gorm"
)

// ENGINE ERRORS
var (
	ErrPlaywrightNotInitialized = errors.New("PLAYWRIGHT NOT INITIALIZED")
	ErrJobAlreadyRunning        = errors.New("JOB IS ALREADY RUNNING")
	ErrJobNotFound              = errors.New("JOB NOT FOUND")
	ErrBrowserCreation          = errors.New("FAILED TO CREATE BROWSER")
	ErrPageCreation             = errors.New("FAILED TO CREATE PAGE")
)

// ENGINE CORE STRUCT
type Engine struct {
	db            *gorm.DB
	cfg           *config.Config
	runningJobs   map[string]context.CancelFunc
	jobProgress   map[string]int
	jobStartTimes map[string]time.Time
	jobDurations  map[string]time.Duration
	mu            sync.Mutex
	playwright    *playwright.Playwright
	browserPool   chan browserInstance
	initialized   bool
	initMu        sync.Mutex
}

// BROWSER INSTANCE
type browserInstance struct {
	browser *playwright.Browser
}

// JOB INFO STRUCT
type jobInfo struct {
	ctx             context.Context
	cancel          context.CancelFunc
	browser         *playwright.Browser
	page            *playwright.Page
	job             *models.Job
	startTime       time.Time
	visitedURLs     map[string]bool
	foundAssets     map[string]bool
	paginationLinks []string
	progress        int
	wg              *sync.WaitGroup
	urlsMu          sync.Mutex
	assetsMu        sync.Mutex
	paginationMu    sync.Mutex
}

// SCRAPE TASK
type scrapeTask struct {
	url   string
	depth int
}

// NEW ENGINE FACTORY
func NewEngine(db *gorm.DB, cfg *config.Config) *Engine {
	log.Printf("CREATING NEW SCRAPER ENGINE")
	engine := &Engine{
		db:            db,
		cfg:           cfg,
		runningJobs:   make(map[string]context.CancelFunc),
		jobProgress:   make(map[string]int),
		jobStartTimes: make(map[string]time.Time),
		jobDurations:  make(map[string]time.Duration),
		mu:            sync.Mutex{},
		browserPool:   make(chan browserInstance, cfg.MaxConcurrent),
		initialized:   false,
		initMu:        sync.Mutex{},
	}

	log.Printf("INITIALIZING PLAYWRIGHT FOR ENGINE")
	err := engine.initPlaywright()
	if err != nil {
		log.Printf("ERROR INITIALIZING PLAYWRIGHT: %v", err)
		engine.initialized = false
	}

	return engine
}

// INIT PLAYWRIGHT
func (e *Engine) initPlaywright() error {
	e.initMu.Lock()
	defer e.initMu.Unlock()

	log.Printf("PLAYWRIGHT INITIALIZING WITH %d BROWSERS IN POOL", len(e.browserPool))

	// AVOID DOUBLE INITIALIZATION
	if e.initialized {
		log.Printf("PLAYWRIGHT WAS ALREADY INITIALIZED WITH %d BROWSERS IN POOL", len(e.browserPool))
		return nil
	}

	// INSTALL PLAYWRIGHT IF NEEDED
	if err := playwright.Install(); err != nil {
		log.Printf("COULD NOT INSTALL PLAYWRIGHT: %v", err)
		return err
	}

	// START PLAYWRIGHT
	log.Printf("STARTING PLAYWRIGHT")
	pw, err := playwright.Run()
	if err != nil {
		log.Printf("COULD NOT START PLAYWRIGHT: %v", err)
		return err
	}

	e.playwright = pw
	e.initialized = true

	log.Printf("PLAYWRIGHT INITIALIZED WITH %d BROWSERS IN POOL", len(e.browserPool))
	return nil
}

// ENSURE PLAYWRIGHT IS INITIALIZED
func (e *Engine) ensureInitialized() error {
	log.Printf("PLAYWRIGHT INIT CHECK STARTED")
	if !e.initialized {
		e.initMu.Lock()
		defer e.initMu.Unlock()

		if !e.initialized {
			log.Printf("INITIALIZING PLAYWRIGHT")
			return e.initPlaywright()
		}
	}
	log.Printf("PLAYWRIGHT ALREADY INITIALIZED")
	return nil
}

// LAUNCH BROWSER WITH STEALTH MODE
func (e *Engine) launchBrowser(headless bool) (*playwright.Browser, error) {
	log.Printf("LAUNCHING BROWSER (HEADLESS: %v)", headless)
	if err := e.ensureInitialized(); err != nil {
		log.Printf("PLAYWRIGHT INIT CHECK FAILED: %v", err)
		return nil, err
	}

	if e.playwright == nil {
		log.Printf("PLAYWRIGHT WAS NOT INITIALIZED!")
		return nil, ErrPlaywrightNotInitialized
	}

	// LAUNCH BROWSER WITH STEALTH OPTIONS
	log.Printf("OPENING BROWSER")
	browser, err := e.playwright.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--disable-gpu",
			"--disable-dev-shm-usage",
			"--disable-setuid-sandbox",
			"--no-sandbox",
			"--disable-blink-features=AutomationControlled",
			"--disable-features=IsolateOrigins,site-per-process",
			"--disable-site-isolation-trials",
			"--ignore-certificate-errors",
			"--disable-web-security",
			"--allow-running-insecure-content",
		},
	})
	if err != nil {
		log.Printf("BROWSER LAUNCH FAILED: %v", err)
		return nil, fmt.Errorf("COULD NOT LAUNCH BROWSER: %v", err)
	}
	log.Printf("BROWSER LAUNCHED SUCCESSFULLY")

	return &browser, nil
}

// GET BROWSER FROM POOL OR CREATE NEW
func (e *Engine) getBrowser(job *models.Job) (*playwright.Browser, error) {
	log.Printf("GETTING BROWSER FOR JOB %s", job.ID)
	// TRY TO GET FROM POOL FIRST
	select {
	case instance := <-e.browserPool:
		log.Printf("REUSING BROWSER FROM POOL")
		return instance.browser, nil
	default:
		// POOL EMPTY, CREATE NEW BROWSER
		log.Printf("BROWSER POOL EMPTY, CREATING NEW BROWSER")
		headless, _ := job.Processing["headless"].(bool)
		return e.launchBrowser(headless)
	}
}

// RETURN BROWSER TO POOL
func (e *Engine) returnBrowser(browser *playwright.Browser) {
	if browser == nil {
		log.Printf("ATTEMPTED TO RETURN NIL BROWSER")
		return
	}

	log.Printf("RETURNING BROWSER TO POOL")
	// ONLY RETURN TO POOL IF NOT FULL
	select {
	case e.browserPool <- browserInstance{browser: browser}:
		log.Printf("BROWSER RETURNED TO POOL")
	default:
		// POOL FULL, CLOSE BROWSER
		log.Printf("BROWSER POOL FULL, CLOSING BROWSER")
		(*browser).Close()
	}
}

// RUN JOB
func (e *Engine) RunJob(jobID string) error {
	log.Printf("STARTING JOB %s", jobID)
	if err := e.ensureInitialized(); err != nil {
		log.Printf("PLAYWRIGHT NOT INITIALIZED FOR JOB %s: %v", jobID, err)
		return err
	}

	e.mu.Lock()
	// CHECK IF JOB IS ALREADY RUNNING
	if _, running := e.runningJobs[jobID]; running {
		log.Printf("JOB %s IS ALREADY RUNNING", jobID)
		e.mu.Unlock()
		return ErrJobAlreadyRunning
	}
	e.mu.Unlock()

	// GET JOB FROM DATABASE
	var job models.Job
	if err := e.db.First(&job, "id = ?", jobID).Error; err != nil {
		log.Printf("JOB %s NOT FOUND: %v", jobID, err)
		return fmt.Errorf("FAILED TO FIND JOB: %v", err)
	}

	// UPDATE JOB STATUS
	log.Printf("UPDATING JOB %s STATUS TO RUNNING", jobID)
	e.db.Model(&job).Updates(map[string]any{
		"status":   "running",
		"last_run": time.Now(),
	})

	// CREATE CONTEXT WITH TIMEOUT
	timeout := time.Duration(e.cfg.DefaultTimeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// RECORD JOB START
	e.mu.Lock()
	e.runningJobs[jobID] = cancel
	e.jobStartTimes[jobID] = time.Now()
	e.jobProgress[jobID] = 0
	e.mu.Unlock()
	log.Printf("JOB %s REGISTERED AND STARTING", jobID)

	// RUN JOB IN GOROUTINE WITH IMPROVED ERROR HANDLING
	go e.executeJob(ctx, cancel, jobID, &job)

	return nil
}

func (e *Engine) executeJob(ctx context.Context, cancel context.CancelFunc, jobID string, job *models.Job) {
	defer cancel()
	defer e.finishJob(jobID)
	log.Printf("JOB %s GOROUTINE STARTED", jobID)

	// GET BROWSER
	browser, err := e.getBrowser(job)
	if err != nil {
		log.Printf("ERROR GETTING BROWSER FOR JOB %s: %v", jobID, err)
		e.updateJobStatus(jobID, "error")
		return
	}
	defer e.returnBrowser(browser)

	// CREATE JOB INFO WITH WAITGROUP FOR TASKS
	var wg sync.WaitGroup
	info := &jobInfo{
		ctx:             ctx,
		cancel:          cancel,
		browser:         browser,
		job:             job,
		startTime:       time.Now(),
		visitedURLs:     make(map[string]bool),
		foundAssets:     make(map[string]bool),
		paginationLinks: []string{},
		progress:        0,
		wg:              &wg,
	}

	// TRY TO CREATE PAGE
	log.Printf("CREATING PAGE FOR JOB %s", jobID)
	page, err := (*browser).NewPage(playwright.BrowserNewPageOptions{
		RecordVideo: &playwright.RecordVideo{
			Dir: e.cfg.StoragePath,
		},
	})
	if err != nil {
		log.Printf("ERROR CREATING PAGE FOR JOB %s: %v", jobID, err)
		e.updateJobStatus(jobID, "error")
		return
	}
	info.page = &page
	defer page.Close()
	log.Printf("PAGE CREATED FOR JOB %s", jobID)

	// CONFIGURE PAGE
	log.Printf("CONFIGURING PAGE FOR JOB %s", jobID)
	e.configurePage(info)

	// CREATE TASK QUEUE
	taskQueue := make(chan scrapeTask, 100)
	log.Printf("TASK QUEUE CREATED FOR JOB %s", jobID)

	// CREATE DONE CHANNEL FOR CLEAN SHUTDOWN
	done := make(chan struct{})

	// START WORKER GOROUTINES
	maxWorkers := 5 // LIMIT CONCURRENT SCRAPES
	log.Printf("STARTING %d WORKERS FOR JOB %s", maxWorkers, jobID)
	for range maxWorkers {
		wg.Add(1)
		go e.scrapeWorker(info, taskQueue, &wg)
	}

	// ADD INITIAL URL TO QUEUE
	log.Printf("ADDING INITIAL URL %s TO QUEUE FOR JOB %s", job.BaseURL, jobID)
	taskQueue <- scrapeTask{url: job.BaseURL, depth: 0}

	// MONITOR GOROUTINE FOR COMPLETION
	go func() {
		wg.Wait()
		close(done)
	}()

	// WAIT FOR COMPLETION OR TIMEOUT
	select {
	case <-ctx.Done():
		// TIMEOUT OR CANCELLATION
		log.Printf("JOB %s CONTEXT DONE: %v", jobID, ctx.Err())
		close(taskQueue) // SAFELY CLOSE THE QUEUE
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("JOB %s TIMED OUT", jobID)
			e.updateJobStatus(jobID, "timeout")
		} else {
			log.Printf("JOB %s STOPPED", jobID)
			e.updateJobStatus(jobID, "stopped")
		}
	case <-done:
		// ALL TASKS COMPLETED, SAFELY CLOSE QUEUE
		log.Printf("JOB %s TASKS COMPLETED, CLOSING QUEUE", jobID)
		close(taskQueue)

		// PROCESS PAGINATION IF ANY
		info.paginationMu.Lock()
		paginationLinks := info.paginationLinks
		info.paginationMu.Unlock()

		if len(paginationLinks) > 0 {
			log.Printf("JOB %s HAS %d PAGINATION LINKS TO PROCESS", jobID, len(paginationLinks))
			// HANDLE PAGINATION LOGIC HERE - THIS WOULD BE A SEPARATE PHASE
		}

		log.Printf("JOB %s COMPLETED SUCCESSFULLY", jobID)
		e.updateJobStatus(jobID, "completed")
	}
}

// WORKER THAT PROCESSES URLS
func (e *Engine) scrapeWorker(info *jobInfo, taskQueue chan scrapeTask, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range taskQueue {
		// CHECK CONTEXT BEFORE PROCESSING
		if info.ctx.Err() != nil {
			log.Printf("WORKER EXITING DUE TO CANCELLED CONTEXT: %v", info.ctx.Err())
			return
		}

		log.Printf("WORKER PROCESSING URL: %s (DEPTH: %d)", task.url, task.depth)
		if err := e.processURL(info, task.url, task.depth, taskQueue); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				log.Printf("WORKER EXITING DUE TO CONTEXT: %v", err)
				return // EXIT ON CONTEXT DONE
			}
			log.Printf("ERROR PROCESSING URL %s: %v", task.url, err)
		}
	}
	log.Printf("WORKER EXITING (QUEUE CLOSED)")
}

// CONFIGURE PAGE WITH DEFAULT SETTINGS
func (e *Engine) configurePage(info *jobInfo) {
	log.Printf("CONFIGURING PAGE WITH DEFAULT SETTINGS")
	// SET USER AGENT AND HEADERS
	(*info.page).SetExtraHTTPHeaders(map[string]string{
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Accept-Encoding":           "gzip, deflate, br",
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
	})

	// SET TIMEOUT
	(*info.page).SetDefaultTimeout(float64(e.cfg.DefaultTimeout))
	log.Printf("PAGE TIMEOUT SET TO %d MS", e.cfg.DefaultTimeout)

	// APPLY STEALTH MODE
	log.Printf("APPLYING STEALTH MODE TO PAGE")
	err := stealth.Inject(*info.page)
	if err != nil {
		log.Printf("STEALTH INJECTION FAILED: %v", err)
	} else {
		log.Printf("STEALTH MODE APPLIED SUCCESSFULLY")
	}
}

func safeSendToQueue(taskQueue chan scrapeTask, task scrapeTask) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RECOVERED FROM PANIC WHEN SENDING TO QUEUE: %v", r)
		}
	}()

	select {
	case taskQueue <- task:
		return true
	default:
		// TRY NON-BLOCKING SEND FIRST
		select {
		case taskQueue <- task:
			return true
		case <-time.After(100 * time.Millisecond):
			// IF THE CHANNEL IS CLOSED, THIS WILL PANIC AND BE CAUGHT BY RECOVER
			// ATTEMPT A SEND BUT WITH TIMEOUT TO AVOID BLOCKING FOREVER
			return false
		}
	}
}

// PROCESS URL WITH RETRY LOGIC
func (e *Engine) processURL(info *jobInfo, urlStr string, depth int, taskQueue chan scrapeTask) error {
	log.Printf("PROCESSING URL: %s (DEPTH: %d)", urlStr, depth)
	// CHECK CONTEXT
	if info.ctx.Err() != nil {
		log.Printf("CONTEXT ERROR WHILE PROCESSING %s: %v", urlStr, info.ctx.Err())
		return info.ctx.Err()
	}

	// CHECK IF URL ALREADY VISITED
	info.urlsMu.Lock()
	if info.visitedURLs[urlStr] {
		info.urlsMu.Unlock()
		log.Printf("URL ALREADY VISITED: %s", urlStr)
		return nil
	}
	info.visitedURLs[urlStr] = true
	info.urlsMu.Unlock()
	log.Printf("URL MARKED AS VISITED: %s", urlStr)

	// GET MAX DEPTH FROM RULES
	maxDepth := 3 // DEFAULT
	if val, ok := info.job.Rules["maxDepth"].(float64); ok {
		maxDepth = int(val)
	}
	log.Printf("MAX DEPTH: %d, CURRENT: %d", maxDepth, depth)

	// CHECK DEPTH LIMIT
	if depth > maxDepth {
		log.Printf("MAX DEPTH REACHED FOR URL: %s", urlStr)
		return nil
	}

	// CHECK DOMAIN RESTRICTION
	if sameDomainOnly, _ := info.job.Rules["sameDomainOnly"].(bool); sameDomainOnly {
		if !isSameDomain(info.job.BaseURL, urlStr) {
			log.Printf("URL %s SKIPPED (DIFFERENT DOMAIN)", urlStr)
			return nil
		}
	}

	// RANDOM DELAY BEFORE NAVIGATION
	randDelay := time.Duration(100+rand.Intn(500)) * time.Millisecond
	log.Printf("WAITING %d MS BEFORE NAVIGATION TO %s", randDelay.Milliseconds(), urlStr)
	time.Sleep(randDelay)

	// ATTEMPT NAVIGATION WITH RETRIES
	maxRetries := 2
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("RETRY %d FOR URL %s", attempt, urlStr)
			time.Sleep(time.Duration(attempt) * time.Second) // BACKOFF
		}

		// NAVIGATION ATTEMPT
		log.Printf("NAVIGATING TO URL: %s (ATTEMPT: %d)", urlStr, attempt)
		response, err := (*info.page).Goto(urlStr, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(float64(e.cfg.DefaultTimeout)),
		})

		// CHECK FOR SUCCESS
		if err == nil && response != nil && response.Status() < 400 {
			log.Printf("NAVIGATION SUCCESSFUL: %s (STATUS: %d)", urlStr, response.Status())

			// UPDATE PROGRESS
			e.updateJobProgress(info.job.ID, len(info.visitedURLs))
			log.Printf("PROGRESS UPDATED: %d URLS VISITED", len(info.visitedURLs))

			// EXTRACT CONTENT
			log.Printf("EXTRACTING CONTENT FROM %s", urlStr)
			if err := e.extractContent(info, urlStr, depth, taskQueue); err != nil {
				log.Printf("EXTRACTION ERROR FOR %s: %v", urlStr, err)
			} else {
				log.Printf("CONTENT EXTRACTED SUCCESSFULLY FROM %s", urlStr)
			}

			// SUCCESS - ADD RANDOM DELAY
			baseDelay := 1000 // 1 SECOND
			if delay, ok := info.job.Rules["requestDelay"].(float64); ok && delay > 0 {
				baseDelay = int(delay)
			}
			jitter := float64(baseDelay) * (0.7 + 0.6*rand.Float64())
			delay := time.Duration(jitter) * time.Millisecond
			log.Printf("WAITING %d MS AFTER PROCESSING %s", delay.Milliseconds(), urlStr)
			time.Sleep(delay)

			return nil
		}

		// LOG ERROR
		log.Printf("NAVIGATION ERROR FOR %s: %v", urlStr, err)

		// CHECK FOR PERMANENT ERRORS
		if err != nil && (strings.Contains(err.Error(), "ERR_CERT_") ||
			strings.Contains(err.Error(), "ERR_NAME_NOT_RESOLVED") ||
			strings.Contains(err.Error(), "ERR_CONNECTION_REFUSED")) {
			log.Printf("PERMANENT ERROR FOR %s: %v", urlStr, err)
			return nil // SKIP THIS URL
		}
	}

	log.Printf("MAX RETRIES REACHED FOR %s: %v", urlStr, err)
	return nil
}

// EXTRACT CONTENT FROM PAGE
func (e *Engine) extractContent(info *jobInfo, urlStr string, depth int, taskQueue chan scrapeTask) error {
	log.Printf("EXTRACTING CONTENT FROM %s", urlStr)
	// CHECK CONTEXT
	if info.ctx.Err() != nil {
		log.Printf("CONTEXT ERROR DURING EXTRACTION: %v", info.ctx.Err())
		return info.ctx.Err()
	}

	// PROCESS SELECTORS
	for _, selectorItem := range info.job.Selectors {
		selector, ok := selectorItem.(map[string]any)
		if !ok {
			log.Printf("INVALID SELECTOR FORMAT")
			continue
		}

		selectorValue, _ := selector["value"].(string)
		purpose, _ := selector["purpose"].(string)
		attribute, _ := selector["attribute"].(string)

		// SKIP EMPTY SELECTORS
		if selectorValue == "" || attribute == "" {
			log.Printf("EMPTY SELECTOR OR ATTRIBUTE")
			continue
		}

		log.Printf("PROCESSING SELECTOR: %s (PURPOSE: %s, ATTRIBUTE: %s)", selectorValue, purpose, attribute)

		// QUERY FOR ELEMENTS
		elements, err := (*info.page).Locator(selectorValue).All()
		if err != nil {
			log.Printf("SELECTOR ERROR %s: %v", selectorValue, err)
			continue
		}
		log.Printf("FOUND %d ELEMENTS WITH SELECTOR %s", len(elements), selectorValue)

		// PROCESS ELEMENTS
		for i, element := range elements {
			// CHECK CONTEXT
			if info.ctx.Err() != nil {
				log.Printf("CONTEXT ERROR DURING ELEMENT PROCESSING: %v", info.ctx.Err())
				return info.ctx.Err()
			}

			// GET ATTRIBUTE
			attrValue, err := element.GetAttribute(attribute)
			if err != nil || attrValue == "" {
				log.Printf("ELEMENT %d: NO ATTRIBUTE %s FOUND", i, attribute)
				continue
			}
			log.Printf("ELEMENT %d: ATTRIBUTE %s = %s", i, attribute, attrValue)

			// RESOLVE URL
			absURL := utils.ResolveURL(urlStr, attrValue)
			if absURL == "" {
				log.Printf("COULD NOT RESOLVE URL FROM %s", attrValue)
				continue
			}
			log.Printf("RESOLVED URL: %s", absURL)

			// HANDLE BASED ON PURPOSE
			switch purpose {
			case "assets", "asset":
				// DETERMINE ASSET TYPE
				assetType := ""
				if tagName, err := element.Evaluate("el => el.tagName.toLowerCase()", nil); err == nil {
					switch tagName {
					case "img":
						assetType = "image"
					case "video", "source":
						assetType = "video"
					case "audio":
						assetType = "audio"
					}
					log.Printf("ASSET TYPE FROM TAG: %s", assetType)
				}

				// FALLBACK TO URL DETECTION
				if assetType == "" {
					assetType = detectAssetType(absURL)
					log.Printf("ASSET TYPE FROM URL: %s", assetType)
				}

				// PROCESS ASSET
				log.Printf("PROCESSING ASSET: %s (TYPE: %s)", absURL, assetType)
				e.processAssetURL(info, absURL, assetType, urlStr)

			case "links", "link":
				// SAFELY ADD TO QUEUE
				log.Printf("ADDING LINK TO QUEUE: %s (DEPTH: %d)", absURL, depth+1)
				if !safeSendToQueue(taskQueue, scrapeTask{url: absURL, depth: depth + 1}) {
					log.Printf("FAILED TO ADD LINK TO QUEUE (LIKELY CLOSED): %s", absURL)
				}

			case "pagination":
				if depth == 0 {
					// STORE FOR PROCESSING AFTER REGULAR LINKS
					log.Printf("FOUND PAGINATION LINK: %s", absURL)
					info.paginationMu.Lock()
					info.paginationLinks = append(info.paginationLinks, absURL)
					info.paginationMu.Unlock()
				}
			}
		}
	}

	return nil
}

// DETECT ASSET TYPE FROM URL
func detectAssetType(urlStr string) string {
	ext := strings.ToLower(filepath.Ext(urlStr))
	log.Printf("DETECTING ASSET TYPE FOR %s (EXT: %s)", urlStr, ext)

	switch {
	case strings.Contains(ext, ".jpg"), strings.Contains(ext, ".jpeg"),
		strings.Contains(ext, ".png"), strings.Contains(ext, ".gif"),
		strings.Contains(ext, ".bmp"), strings.Contains(ext, ".webp"):
		return "image"
	case strings.Contains(ext, ".mp4"), strings.Contains(ext, ".webm"),
		strings.Contains(ext, ".mov"), strings.Contains(ext, ".avi"),
		strings.Contains(ext, ".mkv"):
		return "video"
	case strings.Contains(ext, ".mp3"), strings.Contains(ext, ".wav"),
		strings.Contains(ext, ".ogg"), strings.Contains(ext, ".flac"):
		return "audio"
	case strings.Contains(ext, ".pdf"), strings.Contains(ext, ".doc"),
		strings.Contains(ext, ".docx"), strings.Contains(ext, ".txt"):
		return "document"
	default:
		// CHECK URL PATTERNS
		if strings.Contains(urlStr, "video") || strings.Contains(urlStr, "stream") {
			return "video"
		}
		return "unknown"
	}
}

// PROCESS ASSET URL
func (e *Engine) processAssetURL(info *jobInfo, url string, assetType string, sourceURL string) {
	log.Printf("PROCESSING ASSET URL: %s (TYPE: %s)", url, assetType)
	// CHECK MAX ASSETS LIMIT
	maxAssets := 0
	if val, ok := info.job.Rules["maxAssets"].(float64); ok {
		maxAssets = int(val)
	}
	log.Printf("MAX ASSETS: %d, CURRENT: %d", maxAssets, len(info.foundAssets))

	// CHECK IF ALREADY PROCESSED
	info.assetsMu.Lock()
	defer info.assetsMu.Unlock()

	if info.foundAssets[url] {
		log.Printf("ASSET ALREADY PROCESSED: %s", url)
		return
	}

	if maxAssets > 0 && len(info.foundAssets) >= maxAssets {
		log.Printf("MAX ASSETS LIMIT REACHED, SKIPPING: %s", url)
		return
	}

	// MARK AS FOUND
	info.foundAssets[url] = true
	log.Printf("ASSET MARKED AS FOUND: %s", url)

	// QUEUE PROCESSING
	info.wg.Add(1)
	log.Printf("QUEUING ASSET DOWNLOAD: %s", url)
	go func(assetURL, assetType, sourcePage string) {
		defer info.wg.Done()
		e.processAsset(info, assetURL, assetType)
	}(url, assetType, sourceURL)
}

// PROCESS ASSET DOWNLOAD
func (e *Engine) processAsset(info *jobInfo, url string, assetType string) {
	// CHECK CONTEXT
	if info.ctx.Err() != nil {
		log.Printf("CONTEXT ERROR FOR ASSET %s: %v", url, info.ctx.Err())
		return
	}

	log.Printf("DOWNLOADING ASSET: %s (%s)", url, assetType)

	// GENERATE FILENAME
	urlHash := utils.GenerateHash(url)
	ext := filepath.Ext(url)
	if ext == "" {
		// DEFAULT EXTENSIONS
		switch assetType {
		case "image":
			ext = ".jpg"
		case "video":
			ext = ".mp4"
		case "audio":
			ext = ".mp3"
		default:
			ext = ".bin"
		}
		log.Printf("USING DEFAULT EXTENSION %s FOR %s", ext, url)
	}

	// CREATE ASSET RECORD
	assetID := utils.GenerateID("asset")
	log.Printf("CREATED ASSET ID: %s", assetID)
	asset := models.Asset{
		ID:        assetID,
		JobID:     info.job.ID,
		URL:       url,
		Type:      assetType,
		Date:      time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// CREATE STORAGE PATH
	jobFolder := filepath.Join(e.cfg.StoragePath, "jobs", info.job.ID)
	assetFolder := filepath.Join(jobFolder, assetType+"s")
	log.Printf("ASSET FOLDER: %s", assetFolder)

	// ENSURE DIRECTORIES EXIST
	if err := os.MkdirAll(assetFolder, 0755); err != nil {
		log.Printf("DIRECTORY ERROR: %v", err)
		return
	}

	filename := assetID + "_" + urlHash + ext
	localPath := filepath.Join(assetFolder, filename)
	asset.LocalPath = localPath
	log.Printf("ASSET LOCAL PATH: %s", localPath)

	// DOWNLOAD BASED ON TYPE
	var err error
	switch assetType {
	case "video":
		log.Printf("DOWNLOADING VIDEO: %s", url)
		err = e.downloadVideo(info, url, localPath)
	case "image":
		log.Printf("DOWNLOADING IMAGE: %s", url)
		err = e.downloadWithPlaywright(info, url, localPath)
	default:
		log.Printf("DOWNLOADING GENERIC ASSET: %s", url)
		err = e.downloadWithPlaywright(info, url, localPath)
	}

	if err != nil {
		log.Printf("DOWNLOAD ERROR FOR %s: %v", url, err)
	} else {
		log.Printf("ASSET DOWNLOADED SUCCESSFULLY: %s", url)
		// GET FILE SIZE
		if fileInfo, err := os.Stat(localPath); err == nil {
			asset.Size = fileInfo.Size()
			log.Printf("ASSET SIZE: %d BYTES", asset.Size)
		}

		// EXTRACT METADATA
		log.Printf("EXTRACTING METADATA FOR %s", url)
		if err := e.extractAssetMetadata(&asset); err != nil {
			log.Printf("METADATA ERROR: %v", err)
		} else {
			log.Printf("METADATA EXTRACTED SUCCESSFULLY")
		}
	}

	// SAVE TO DATABASE
	log.Printf("SAVING ASSET TO DATABASE: %s", assetID)
	if err := e.db.Create(&asset).Error; err != nil {
		log.Printf("DB ERROR: %v", err)
	} else {
		log.Printf("ASSET SAVED TO DATABASE: %s", assetID)
	}
}

// EXTRACT ASSET METADATA
func (e *Engine) extractAssetMetadata(asset *models.Asset) error {
	log.Printf("EXTRACTING METADATA FOR %s", asset.ID)
	switch asset.Type {
	case "image":
		return e.extractImageMetadata(asset)
	case "video":
		return e.extractVideoMetadata(asset)
	default:
		log.Printf("NO METADATA EXTRACTION FOR TYPE: %s", asset.Type)
		return nil // NO METADATA FOR OTHER TYPES
	}
}

// EXTRACT IMAGE METADATA
func (e *Engine) extractImageMetadata(asset *models.Asset) error {
	log.Printf("EXTRACTING IMAGE METADATA: %s", asset.LocalPath)
	file, err := os.Open(asset.LocalPath)
	if err != nil {
		log.Printf("ERROR OPENING IMAGE FILE: %v", err)
		return err
	}
	defer file.Close()

	// READ IMAGE CONFIG
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Printf("ERROR DECODING IMAGE: %v", err)
		return err
	}

	// STORE DIMENSIONS
	if asset.Metadata == nil {
		asset.Metadata = make(map[string]interface{})
	}
	asset.Metadata["width"] = config.Width
	asset.Metadata["height"] = config.Height
	log.Printf("IMAGE DIMENSIONS: %dx%d", config.Width, config.Height)

	return nil
}

// EXTRACT VIDEO METADATA WITH FFPROBE
func (e *Engine) extractVideoMetadata(asset *models.Asset) error {
	log.Printf("EXTRACTING VIDEO METADATA: %s", asset.LocalPath)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		asset.LocalPath)

	log.Printf("RUNNING FFPROBE: %v", cmd.Args)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("FFPROBE ERROR: %v", err)
		return fmt.Errorf("FFPROBE ERROR: %v", err)
	}

	// PARSE JSON
	var metadata map[string]interface{}
	if err := json.Unmarshal(output, &metadata); err != nil {
		log.Printf("JSON ERROR: %v", err)
		return fmt.Errorf("JSON ERROR: %v", err)
	}

	asset.Metadata = metadata
	log.Printf("VIDEO METADATA EXTRACTED SUCCESSFULLY")
	return nil
}

// DOWNLOAD VIDEO
func (e *Engine) downloadVideo(info *jobInfo, url string, localPath string) error {
	log.Printf("DOWNLOADING VIDEO: %s", url)

	// CHECK CONTEXT FIRST
	if info.ctx.Err() != nil {
		log.Printf("CONTEXT ALREADY CANCELLED, SKIPPING DOWNLOAD: %v", info.ctx.Err())
		return info.ctx.Err()
	}

	// CHECK FOR STREAMING
	if strings.Contains(url, "m3u8") {
		log.Printf("HLS STREAM DETECTED: %s", url)
		return e.downloadHLSVideo(url, localPath)
	} else if strings.Contains(url, ".mpd") {
		log.Printf("DASH STREAM DETECTED: %s", url)
		return e.downloadDASHVideo(url, localPath)
	}

	// TRY HTTP DOWNLOAD FIRST
	log.Printf("ATTEMPTING HTTP VIDEO DOWNLOAD: %s", url)
	err := e.downloadVideoHTTP(info, url, localPath)
	if err == nil {
		return nil
	}

	// FALLBACK TO PLAYWRIGHT IF HTTP FAILS
	if info.ctx.Err() != nil {
		log.Printf("CONTEXT CANCELLED DURING HTTP DOWNLOAD, ABORTING: %v", info.ctx.Err())
		return info.ctx.Err()
	}

	log.Printf("HTTP DOWNLOAD FAILED, TRYING PLAYWRIGHT: %v", err)
	return e.downloadWithPlaywright(info, url, localPath)
}

// DOWNLOAD VIDEO VIA HTTP
func (e *Engine) downloadVideoHTTP(info *jobInfo, url string, localPath string) error {
	log.Printf("DOWNLOADING VIDEO VIA HTTP: %s", url)

	// CREATE CHILD CONTEXT WITH TIMEOUT
	downloadCtx, cancelDownload := context.WithTimeout(info.ctx, 10*time.Minute)
	defer cancelDownload()

	// CREATE NEW GOROUTINE FOR DOWNLOAD TO PREVENT BLOCKING
	downloadDone := make(chan error, 1)
	go func() {
		// CREATE HTTP REQUEST
		req, err := http.NewRequestWithContext(downloadCtx, "GET", url, nil)
		if err != nil {
			log.Printf("HTTP REQUEST ERROR: %v", err)
			downloadDone <- err
			return
		}

		// SET HEADERS
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")

		// SET REFERER IF MAIN PAGE URL IS AVAILABLE
		if info.page != nil {
			req.Header.Set("Referer", (*info.page).URL())
		}

		// CREATE HTTP CLIENT WITH TIMEOUT
		client := &http.Client{
			Timeout: 5 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// ALLOW UP TO 10 REDIRECTS
				if len(via) >= 10 {
					return errors.New("TOO MANY REDIRECTS")
				}
				return nil
			},
		}

		// MAKE HTTP REQUEST
		log.Printf("SENDING HTTP REQUEST FOR VIDEO: %s", url)
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("HTTP REQUEST FAILED: %v", err)
			downloadDone <- err
			return
		}
		defer resp.Body.Close()

		// CHECK STATUS CODE
		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("BAD STATUS CODE: %d", resp.StatusCode)
			log.Printf("HTTP ERROR: %v", err)
			downloadDone <- err
			return
		}

		// CREATE OUTPUT FILE
		log.Printf("CREATING FILE: %s", localPath)
		out, err := os.Create(localPath)
		if err != nil {
			log.Printf("FILE CREATE ERROR: %v", err)
			downloadDone <- err
			return
		}
		defer out.Close()

		// COPY WITH PROGRESS REPORTING AND CONTEXT CHECKING
		written, err := io.Copy(out, resp.Body)
		if err != nil {
			log.Printf("DOWNLOAD WRITE ERROR: %v", err)
			downloadDone <- err
			return
		}

		log.Printf("VIDEO DOWNLOAD COMPLETE: %d BYTES", written)
		downloadDone <- nil
	}()

	// WAIT FOR DOWNLOAD OR CONTEXT CANCELLATION
	select {
	case err := <-downloadDone:
		return err
	case <-downloadCtx.Done():
		log.Printf("DOWNLOAD CANCELLED: %v", downloadCtx.Err())
		return downloadCtx.Err()
	}
}

// DOWNLOAD HLS STREAM
func (e *Engine) downloadHLSVideo(url string, localPath string) error {
	log.Printf("DOWNLOADING HLS STREAM: %s TO %s", url, localPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", url,
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
		"-y", localPath)

	log.Printf("RUNNING FFMPEG: %v", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFMPEG ERROR: %v - OUTPUT: %s", err, string(output))
		return fmt.Errorf("FFMPEG ERROR: %v - OUTPUT: %s", err, string(output))
	}
	log.Printf("HLS DOWNLOAD SUCCESSFUL: %s", url)

	return nil
}

// DOWNLOAD DASH STREAM
func (e *Engine) downloadDASHVideo(url string, localPath string) error {
	log.Printf("DOWNLOADING DASH STREAM: %s TO %s", url, localPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", url,
		"-c", "copy",
		"-y", localPath)

	log.Printf("RUNNING FFMPEG: %v", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFMPEG ERROR: %v - OUTPUT: %s", err, string(output))
		return fmt.Errorf("FFMPEG ERROR: %v - OUTPUT: %s", err, string(output))
	}
	log.Printf("DASH DOWNLOAD SUCCESSFUL: %s", url)

	return nil
}

// DOWNLOAD WITH PLAYWRIGHT
func (e *Engine) downloadWithPlaywright(info *jobInfo, url string, localPath string) error {
	log.Printf("DOWNLOADING WITH PLAYWRIGHT: %s", url)
	// CREATE DOWNLOAD CONTEXT
	downloadContext, err := (*info.browser).NewContext()
	if err != nil {
		log.Printf("CONTEXT ERROR: %v", err)
		return fmt.Errorf("CONTEXT ERROR: %v", err)
	}
	defer downloadContext.Close()

	// SET HEADERS
	downloadContext.SetExtraHTTPHeaders(map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":          "*/*",
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         (*info.page).URL(),
	})
	log.Printf("DOWNLOAD HEADERS SET")

	// CREATE PAGE
	downloadPage, err := downloadContext.NewPage()
	if err != nil {
		log.Printf("PAGE ERROR: %v", err)
		return fmt.Errorf("PAGE ERROR: %v", err)
	}
	defer downloadPage.Close()
	log.Printf("DOWNLOAD PAGE CREATED")

	// APPLY STEALTH
	if err := stealth.Inject(downloadPage); err != nil {
		log.Printf("STEALTH ERROR: %v", err)
	} else {
		log.Printf("STEALTH APPLIED TO DOWNLOAD PAGE")
	}

	// HANDLE DOWNLOAD
	log.Printf("SETTING UP DOWNLOAD HANDLER FOR %s", url)
	download, err := downloadPage.ExpectDownload(func() error {
		log.Printf("NAVIGATING TO DOWNLOAD URL: %s", url)
		_, err := downloadPage.Goto(url)
		return err
	})
	if err != nil {
		log.Printf("DOWNLOAD ERROR: %v", err)
		return fmt.Errorf("DOWNLOAD ERROR: %v", err)
	}
	log.Printf("DOWNLOAD STARTED: %s", url)

	// SAVE TO PATH
	log.Printf("SAVING DOWNLOAD TO: %s", localPath)
	if err := download.SaveAs(localPath); err != nil {
		log.Printf("SAVE ERROR: %v", err)
		return fmt.Errorf("SAVE ERROR: %v", err)
	}
	log.Printf("DOWNLOAD SAVED SUCCESSFULLY: %s", url)

	return nil
}

// STOP JOB
func (e *Engine) StopJob(jobID string) error {
	log.Printf("STOPPING JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	cancel, running := e.runningJobs[jobID]
	if !running {
		log.Printf("JOB %s IS NOT RUNNING", jobID)
		return fmt.Errorf("JOB %s NOT RUNNING", jobID)
	}

	log.Printf("CANCELLING JOB: %s", jobID)
	cancel()
	e.updateJobStatus(jobID, "stopped")
	log.Printf("JOB %s STOPPED", jobID)
	return nil
}

// GET JOB PROGRESS
func (e *Engine) GetJobProgress(jobID string) (int, error) {
	log.Printf("GETTING PROGRESS FOR JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	progress, exists := e.jobProgress[jobID]
	if !exists {
		log.Printf("JOB %s NOT FOUND", jobID)
		return 0, ErrJobNotFound
	}

	log.Printf("JOB %s PROGRESS: %d", jobID, progress)
	return progress, nil
}

// UPDATE JOB PROGRESS
func (e *Engine) updateJobProgress(jobID string, progress int) {
	log.Printf("UPDATING PROGRESS FOR JOB %s: %d", jobID, progress)
	e.mu.Lock()
	defer e.mu.Unlock()

	e.jobProgress[jobID] = progress
}

// GET JOB DURATION
func (e *Engine) GetJobDuration(jobID string) (time.Duration, error) {
	log.Printf("GETTING DURATION FOR JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	if startTime, running := e.jobStartTimes[jobID]; running {
		duration := time.Since(startTime)
		log.Printf("JOB %s RUNNING DURATION: %v", jobID, duration)
		return duration, nil
	}

	duration, exists := e.jobDurations[jobID]
	if !exists {
		log.Printf("JOB %s NOT FOUND", jobID)
		return 0, ErrJobNotFound
	}

	log.Printf("JOB %s COMPLETED DURATION: %v", jobID, duration)
	return duration, nil
}

// FINISH JOB AND CLEANUP
func (e *Engine) finishJob(jobID string) {
	log.Printf("FINISHING JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	if startTime, ok := e.jobStartTimes[jobID]; ok {
		duration := time.Since(startTime)
		e.jobDurations[jobID] = duration
		delete(e.jobStartTimes, jobID)
		log.Printf("JOB %s DURATION: %v", jobID, duration)
	}

	delete(e.runningJobs, jobID)
	log.Printf("JOB %s FINISHED AND CLEANED UP", jobID)
}

// UPDATE JOB STATUS
func (e *Engine) updateJobStatus(jobID string, status string) {
	log.Printf("UPDATING JOB %s STATUS: %s", jobID, status)
	if err := e.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", status).Error; err != nil {
		log.Printf("STATUS UPDATE ERROR: %v", err)
	} else {
		log.Printf("JOB %s STATUS UPDATED TO %s", jobID, status)
	}
}

// CLEAN UP RESOURCES
func (e *Engine) Close() {
	log.Printf("ENGINE SHUTDOWN STARTED")

	// STOP ALL JOBS
	e.mu.Lock()
	jobCount := len(e.runningJobs)
	log.Printf("STOPPING %d RUNNING JOBS", jobCount)
	for jobID, cancel := range e.runningJobs {
		log.Printf("CANCELLING JOB: %s", jobID)
		cancel()
		e.updateJobStatus(jobID, "stopped")
	}
	e.runningJobs = make(map[string]context.CancelFunc)
	e.mu.Unlock()
	log.Printf("ALL JOBS STOPPED")

	// DRAIN POOL AND CLOSE BROWSERS
	log.Printf("DRAINING BROWSER POOL")
	close(e.browserPool)
	browserCount := 0
	for browser := range e.browserPool {
		browserCount++
		if browser.browser != nil {
			log.Printf("CLOSING BROWSER %d", browserCount)
			(*browser.browser).Close()
		}
	}
	log.Printf("%d BROWSERS CLOSED", browserCount)

	// STOP PLAYWRIGHT
	e.initMu.Lock()
	if e.playwright != nil {
		log.Printf("STOPPING PLAYWRIGHT")
		e.playwright.Stop()
		e.playwright = nil
		log.Printf("PLAYWRIGHT STOPPED")
	}
	e.initialized = false
	e.initMu.Unlock()

	log.Printf("ENGINE SHUTDOWN COMPLETE")
}

// CHECK IF SAME DOMAIN
func isSameDomain(baseURLStr, targetURLStr string) bool {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		log.Printf("COULD NOT PARSE BASE URL: %v", err)
		return false
	}

	targetURL, err := url.Parse(targetURLStr)
	if err != nil {
		log.Printf("COULD NOT PARSE TARGET URL: %v", err)
		return false
	}

	baseHost := baseURL.Hostname()
	targetHost := targetURL.Hostname()

	log.Printf("COMPARING DOMAINS: %s vs %s", baseHost, targetHost)
	return baseHost == targetHost
}
