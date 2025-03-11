package scraper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"
	"github.com/playwright-community/playwright-go"
	"gorm.io/gorm"
)

// ENGINE HOLDS THE CORE SCRAPING FUNCTIONALITY
type Engine struct {
	db            *gorm.DB
	cfg           *config.Config
	runningJobs   map[string]context.CancelFunc
	jobProgress   map[string]int
	jobStartTimes map[string]time.Time
	jobDurations  map[string]time.Duration
	mu            sync.Mutex
	playwright    *playwright.Playwright
	browserPool   chan *playwright.Browser
}

// RUNNING JOB INFO STRUCT
type jobInfo struct {
	ctx         context.Context
	cancel      context.CancelFunc
	browser     *playwright.Browser
	page        *playwright.Page
	job         *models.Job
	startTime   time.Time
	visitedURLs map[string]bool
	foundAssets map[string]bool
	progress    int
}

// NEWENGINE CREATES A NEW SCRAPER ENGINE
func NewEngine(db *gorm.DB, cfg *config.Config) *Engine {
	// CREATE NEW ENGINE
	engine := &Engine{
		db:            db,
		cfg:           cfg,
		runningJobs:   make(map[string]context.CancelFunc),
		jobProgress:   make(map[string]int),
		jobStartTimes: make(map[string]time.Time),
		jobDurations:  make(map[string]time.Duration),
		mu:            sync.Mutex{},
		browserPool:   make(chan *playwright.Browser, cfg.MaxConcurrent),
	}

	// INITIALIZE PLAYWRIGHT
	go func() {
		err := engine.initPlaywright()
		if err != nil {
			log.Printf("ERROR INITIALIZING PLAYWRIGHT: %v", err)
		}
	}()

	return engine
}

// INITIALIZE PLAYWRIGHT
func (e *Engine) initPlaywright() error {
	// INSTALL PLAYWRIGHT IF NEEDED
	if err := playwright.Install(); err != nil {
		return fmt.Errorf("COULD NOT INSTALL PLAYWRIGHT: %v", err)
	}

	// START PLAYWRIGHT
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("COULD NOT START PLAYWRIGHT: %v", err)
	}

	e.playwright = pw

	// PRE-LAUNCH BROWSERS FOR POOL
	// for i := range e.cfg.MaxConcurrent {
	// 	browser, err := e.launchBrowser()
	// 	if err != nil {
	// 		log.Printf("WARNING: FAILED TO PRE-LAUNCH BROWSER %d: %v", i, err)
	// 		continue
	// 	}
	// 	e.browserPool <- browser
	// }

	return nil
}

// LAUNCH A NEW BROWSER
func (e *Engine) launchBrowser(headless bool) (*playwright.Browser, error) {
	if e.playwright == nil {
		return nil, errors.New("PLAYWRIGHT NOT INITIALIZED")
	}

	// LAUNCH BROWSER WITH HEADLESS MODE
	browser, err := e.playwright.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--disable-gpu",
			"--disable-dev-shm-usage",
			"--disable-setuid-sandbox",
			"--no-sandbox",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("COULD NOT LAUNCH BROWSER: %v", err)
	}

	return &browser, nil
}

// GET BROWSER FROM POOL OR CREATE NEW ONE
func (e *Engine) getBrowser(job *models.Job) (*playwright.Browser, error) {
	select {
	case browser := <-e.browserPool:
		// GOT BROWSER FROM POOL
		return browser, nil
	default:
		// POOL EMPTY, CREATE NEW BROWSER
		headless, _ := job.Processing["headless"].(bool)
		return e.launchBrowser(headless)
	}
}

// RETURN BROWSER TO POOL
func (e *Engine) returnBrowser(browser *playwright.Browser) {
	// ONLY RETURN TO POOL IF NOT FULL
	select {
	case e.browserPool <- browser:
		// BROWSER RETURNED TO POOL
	default:
		// POOL FULL, CLOSE BROWSER
		if browser != nil {
			(*browser).Close()
		}
	}
}

// RUNJOB STARTS A SCRAPING JOB
func (e *Engine) RunJob(jobID string) error {
	e.mu.Lock()
	// CHECK IF JOB IS ALREADY RUNNING
	if _, running := e.runningJobs[jobID]; running {
		e.mu.Unlock()
		return errors.New("JOB IS ALREADY RUNNING")
	}
	e.mu.Unlock()

	// GET JOB FROM DATABASE
	var job models.Job
	if err := e.db.First(&job, "id = ?", jobID).Error; err != nil {
		return fmt.Errorf("FAILED TO FIND JOB: %v", err)
	}

	// UPDATE JOB STATUS
	e.db.Model(&job).Updates(map[string]interface{}{
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

	// RUN JOB IN GOROUTINE
	go func() {
		defer cancel()
		defer e.finishJob(jobID)

		// GET BROWSER
		browser, err := e.getBrowser(&job)
		if err != nil {
			log.Printf("ERROR GETTING BROWSER FOR JOB %s: %v", jobID, err)
			e.updateJobStatus(jobID, "error")
			return
		}
		defer e.returnBrowser(browser)

		// CREATE JOB INFO
		info := &jobInfo{
			ctx:         ctx,
			cancel:      cancel,
			browser:     browser,
			job:         &job,
			startTime:   time.Now(),
			visitedURLs: make(map[string]bool),
			foundAssets: make(map[string]bool),
			progress:    0,
		}

		// TRY TO CREATE PAGE
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

		// CONFIGURE PAGE
		e.configurePage(info)

		// START SCRAPING FROM BASE URL
		err = e.scrape(info, job.BaseURL, 0)
		if err != nil {
			log.Printf("ERROR SCRAPING JOB %s: %v", jobID, err)
			if ctx.Err() == context.DeadlineExceeded {
				e.updateJobStatus(jobID, "timeout")
			} else {
				e.updateJobStatus(jobID, "error")
			}
			return
		}

		// JOB COMPLETED SUCCESSFULLY
		e.updateJobStatus(jobID, "completed")
	}()

	return nil
}

// CONFIGURE PAGE WITH DEFAULT SETTINGS
func (e *Engine) configurePage(info *jobInfo) {
	// SET USER AGENT
	(*info.page).SetExtraHTTPHeaders(map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36",
	})

	// SET TIMEOUT
	(*info.page).SetDefaultTimeout(float64(e.cfg.DefaultTimeout))

	// SETUP EVENT HANDLERS
	(*info.page).On("request", func(request playwright.Request) {
		// PLACEHOLDER FOR REQUEST HANDLING
		// fmt.Printf("INCOMING REQUEST HOOK: %+v\n", request)
	})

	(*info.page).On("response", func(response playwright.Response) {
		// PLACEHOLDER FOR RESPONSE HANDLING
		//fmt.Printf("OUTGOING RESPONSE HOOK: %+v\n", response)
	})
}

// MAIN SCRAPING FUNCTION - RECURSIVE
func (e *Engine) scrape(info *jobInfo, url string, depth int) error {
	// CHECK CONTEXT
	if info.ctx.Err() != nil {
		return info.ctx.Err()
	}

	// CHECK IF URL ALREADY VISITED
	if info.visitedURLs[url] {
		return nil
	}
	info.visitedURLs[url] = true

	// GET MAX DEPTH FROM RULES
	maxDepth := 3 // DEFAULT
	if val, ok := info.job.Rules["maxDepth"].(float64); ok {
		maxDepth = int(val)
	}

	// CHECK DEPTH LIMIT
	if depth > maxDepth {
		return nil
	}

	// CHECK DOMAIN RESTRICTION
	if sameDomainOnly, _ := info.job.Rules["sameDomainOnly"].(bool); sameDomainOnly {
		if !isSameDomain(info.job.BaseURL, url) {
			return nil
		}
	}

	// NAVIGATE TO URL
	if _, err := (*info.page).Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		log.Printf("ERROR NAVIGATING TO %s: %v", url, err)
		return nil // CONTINUE WITH OTHER URLS
	}

	// UPDATE PROGRESS
	e.updateJobProgress(info.job.ID, len(info.visitedURLs))

	// PROCESS PAGE
	if err := e.processPage(info, url, depth); err != nil {
		log.Printf("ERROR PROCESSING PAGE %s: %v", url, err)
	}

	// APPLY DELAY IF SPECIFIED
	if delay, ok := info.job.Rules["requestDelay"].(float64); ok && delay > 0 {
		select {
		case <-info.ctx.Done():
			return info.ctx.Err()
		case <-time.After(time.Duration(delay) * time.Millisecond):
			// DELAY COMPLETE
		}
	}

	return nil
}

// PROCESS PAGE CONTENT
func (e *Engine) processPage(info *jobInfo, url string, depth int) error {
	// EXTRACT ASSETS AND LINKS BASED ON SELECTORS
	for _, selectorItem := range info.job.Selectors {
		selector, ok := selectorItem.(map[string]any)
		if !ok {
			continue
		}

		selectorValue, _ := selector["value"].(string)
		//selectorType, _ := selector["type"].(string)
		purpose, _ := selector["purpose"].(string)
		attributeSource, _ := selector["attributeSource"].(string)

		// QUERY FOR ELEMENTS
		elements, err := (*info.page).Locator(selectorValue).All()
		if err != nil {
			log.Printf("ERROR SELECTING %s: %v", selectorValue, err)
			continue
		}

		// PROCESS EACH ELEMENT
		for _, element := range elements {
			// GET ATTRIBUTE VALUE
			attrValue, err := element.GetAttribute(attributeSource)
			if err != nil || attrValue == "" {
				continue
			}

			// RESOLVE RELATIVE URL
			absURL := utils.ResolveURL(url, attrValue)

			// HANDLE ELEMENT BASED ON TYPE/PURPOSE
			switch {
			case purpose == "assets":
				// FOUND IMAGE ASSET
				if !info.foundAssets[absURL] {
					fmt.Printf("FOUND ASSET: %s\n", absURL)
					info.foundAssets[absURL] = true
					e.processAsset(info, absURL, "image")
				}

			case purpose == "links":
				// FOUND LINK - QUEUE FOR SCRAPING
				go func(link string, currentDepth int) {
					fmt.Printf("FOUND SCRAPE LINK -> DEPTH %v: %s\n", currentDepth+1, link)
					e.scrape(info, link, currentDepth+1)
				}(absURL, depth)

			case purpose == "pagination":
				// FOUND PAGINATION LINK
				go func(link string, currentDepth int) {
					fmt.Printf("FOUND PAGINATION LINK AT DEPTH %v: %s\n", currentDepth, link)
					e.scrape(info, link, currentDepth)
				}(absURL, depth)

				// TODO: ADD ADDITIONAL TYPES
			}
		}
	}

	return nil
}

// PROCESS AND SAVE ASSET
func (e *Engine) processAsset(info *jobInfo, url string, assetType string) {
	// CHECK MAX ASSETS LIMIT
	maxAssets := 0
	if val, ok := info.job.Rules["maxAssets"].(float64); ok {
		maxAssets = int(val)
	}
	if maxAssets > 0 && len(info.foundAssets) > maxAssets {
		return
	}

	// CREATE ASSET RECORD
	asset := models.Asset{
		ID:        utils.GenerateID("asset"),
		JobID:     info.job.ID,
		URL:       url,
		Type:      assetType,
		Date:      time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	fmt.Printf("PROCESSING ASSET: %+v\n", asset)

	// EXTRACT ADDITIONAL METADATA
	// THIS IS JUST A STUB - THE ACTUAL IMPLEMENTATION WOULD DOWNLOAD THE ASSET
	// AND EXTRACT MORE METADATA

	// SAVE ASSET TO DATABASE
	if err := e.db.Create(&asset).Error; err != nil {
		log.Printf("ERROR SAVING ASSET %s: %v", url, err)
	}
}

// STOP A RUNNING JOB
func (e *Engine) StopJob(jobID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// CHECK IF JOB IS RUNNING
	cancel, running := e.runningJobs[jobID]
	if running {
		// CANCEL JOB CONTEXT
		cancel()
		// UPDATE JOB STATUS
		e.updateJobStatus(jobID, "stopped")
	}
}

// GET JOB PROGRESS
func (e *Engine) GetJobProgress(jobID string) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.jobProgress[jobID]
}

// UPDATE JOB PROGRESS
func (e *Engine) updateJobProgress(jobID string, progress int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.jobProgress[jobID] = progress
}

// GET JOB DURATION
func (e *Engine) GetJobDuration(jobID string) time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()

	// CHECK IF JOB IS RUNNING
	if startTime, running := e.jobStartTimes[jobID]; running {
		// CALCULATE CURRENT DURATION
		return time.Since(startTime)
	}

	// RETURN STORED DURATION FOR COMPLETED JOBS
	return e.jobDurations[jobID]
}

// FINISH JOB AND CLEANUP
func (e *Engine) finishJob(jobID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// CALCULATE FINAL DURATION
	if startTime, ok := e.jobStartTimes[jobID]; ok {
		e.jobDurations[jobID] = time.Since(startTime)
	}

	// CLEAN UP
	delete(e.runningJobs, jobID)
	delete(e.jobStartTimes, jobID)
}

// UPDATE JOB STATUS IN DATABASE
func (e *Engine) updateJobStatus(jobID string, status string) {
	e.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", status)
}

// CLEAN UP RESOURCES
func (e *Engine) Close() {
	// CLOSE ALL RUNNING JOBS
	e.mu.Lock()
	for jobID, cancel := range e.runningJobs {
		cancel()
		e.updateJobStatus(jobID, "stopped")
	}
	e.mu.Unlock()

	// DRAIN BROWSER POOL
	close(e.browserPool)
	for browser := range e.browserPool {
		(*browser).Close()
	}

	// STOP PLAYWRIGHT
	if e.playwright != nil {
		e.playwright.Stop()
	}
}

// HELPER FUNCTION TO CHECK IF URLS HAVE SAME DOMAIN
func isSameDomain(baseURL, targetURL string) bool {
	// SIMPLE STRING-BASED DOMAIN EXTRACTION
	// A MORE ROBUST IMPLEMENTATION WOULD USE URL PARSING
	baseDomain := extractDomain(baseURL)
	targetDomain := extractDomain(targetURL)
	return baseDomain == targetDomain
}

// EXTRACT DOMAIN FROM URL
func extractDomain(urlStr string) string {
	// STRIP PROTOCOL
	domainStart := 0
	if idx := strings.Index(urlStr, "://"); idx != -1 {
		domainStart = idx + 3
	}

	// FIND END OF DOMAIN
	domainEnd := len(urlStr)
	if idx := strings.Index(urlStr[domainStart:], "/"); idx != -1 {
		domainEnd = domainStart + idx
	}

	return urlStr[domainStart:domainEnd]
}
