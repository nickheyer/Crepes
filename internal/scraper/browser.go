package scraper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/utils"
)

// BROWSERMANAGER MANAGES A POOL OF BROWSER INSTANCES
type BrowserManager struct {
	// CONFIGURATION
	maxBrowsers     int
	maxTabs         int
	browserLifetime time.Duration
	defaultTimeout  time.Duration
	headless        bool

	// STATE
	browsers       []*ManagedBrowser
	mu             sync.Mutex
	logger         *utils.Logger
	browserOptions []chromedp.ExecAllocatorOption
}

// MANAGEDBROWSER REPRESENTS A MANAGED BROWSER INSTANCE
type ManagedBrowser struct {
	ID           string
	Context      context.Context
	Cancel       context.CancelFunc
	AllocCancel  context.CancelFunc
	Tabs         []*ManagedTab
	TabCount     int
	LastUsed     time.Time
	Headless     bool
	IsTerminated bool
	InUse        bool
	mu           sync.Mutex
}

// MANAGEDTAB REPRESENTS A BROWSER TAB
type ManagedTab struct {
	ID          string
	Context     context.Context
	Cancel      context.CancelFunc
	InUse       bool
	LastUsed    time.Time
	CurrentURL  string
	ParentID    string
	NetworkLogs []NetworkLog
}

// NETWORKLOG REPRESENTS A NETWORK REQUEST
type NetworkLog struct {
	Method      string
	URL         string
	RequestID   string
	RequestType string
	Status      int
	MimeType    string
	Headers     map[string]any
	Body        []byte
	Timestamp   time.Time
}

// SINGLETON BROWSER MANAGER INSTANCE
var (
	defaultBrowserManager *BrowserManager
	browserManagerOnce    sync.Once
)

// GETBROWSERMANAGER RETURNS THE SINGLETON BROWSER MANAGER INSTANCE
func GetBrowserManager() *BrowserManager {
	browserManagerOnce.Do(func() {
		// CREATE DEFAULT BROWSER MANAGER
		defaultBrowserManager = NewBrowserManager(
			config.AppConfig.MaxBrowsers,
			config.AppConfig.MaxBrowserTabs,
			config.AppConfig.BrowserLifetime,
			config.AppConfig.DefaultTimeout,
			true, // DEFAULT TO HEADLESS
		)
	})

	return defaultBrowserManager
}

// NEWBROWSERMANAGER CREATES A NEW BROWSER MANAGER
func NewBrowserManager(maxBrowsers, maxTabs int, lifetime, timeout time.Duration, headless bool) *BrowserManager {
	// SET SENSIBLE DEFAULTS
	if maxBrowsers <= 0 {
		maxBrowsers = 3
	}
	if maxTabs <= 0 {
		maxTabs = 5
	}
	if lifetime <= 0 {
		lifetime = 30 * time.Minute
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// DEFAULT BROWSER OPTIONS
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
		chromedp.Flag("disable-site-isolation-trials", true),
		chromedp.Flag("autoplay-policy", "no-user-gesture-required"),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("allow-running-insecure-content", true),
		chromedp.UserAgent(config.GetRandomUserAgent()),
		chromedp.WindowSize(1920, 1080),
	)

	manager := &BrowserManager{
		maxBrowsers:     maxBrowsers,
		maxTabs:         maxTabs,
		browserLifetime: lifetime,
		defaultTimeout:  timeout,
		headless:        headless,
		browsers:        make([]*ManagedBrowser, 0, maxBrowsers),
		browserOptions:  options,
		logger:          utils.GetLogger(),
	}

	// START BACKGROUND CLEANUP ROUTINE
	go manager.cleanup()

	return manager
}

// GETBROWSER GETS OR CREATES A BROWSER
func (bm *BrowserManager) GetBrowser(ctx context.Context) (*ManagedBrowser, error) {
	if bm == nil {
		return nil, fmt.Errorf("browser manager is nil")
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	// LOOK FOR AVAILABLE BROWSER
	for _, browser := range bm.browsers {
		browser.mu.Lock()
		if !browser.IsTerminated && browser.TabCount < bm.maxTabs {
			browser.LastUsed = time.Now()
			browser.mu.Unlock()
			return browser, nil
		}
		browser.mu.Unlock()
	}

	// CREATE A NEW BROWSER IF POSSIBLE
	if len(bm.browsers) < bm.maxBrowsers {
		browser, err := bm.createBrowser(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create browser: %w", err)
		}
		bm.browsers = append(bm.browsers, browser)
		return browser, nil
	}

	// FIND THE LEAST RECENTLY USED BROWSER
	oldestIdx := 0
	oldestTime := time.Now()

	for i, browser := range bm.browsers {
		browser.mu.Lock()
		if !browser.IsTerminated && browser.LastUsed.Before(oldestTime) {
			oldestIdx = i
			oldestTime = browser.LastUsed
		}
		browser.mu.Unlock()
	}

	// REUSE THE OLDEST BROWSER
	browser := bm.browsers[oldestIdx]

	// RESET BROWSER TABS IF ALL TABS ARE IN USE
	browser.mu.Lock()
	if browser.TabCount >= bm.maxTabs {
		// CLOSE ALL TABS
		for _, tab := range browser.Tabs {
			if tab.Cancel != nil {
				tab.Cancel()
			}
		}

		browser.Tabs = make([]*ManagedTab, 0, bm.maxTabs)
		browser.TabCount = 0
	}

	browser.LastUsed = time.Now()
	browser.mu.Unlock()

	return browser, nil
}

// CREATEBROWSER CREATES A NEW BROWSER INSTANCE
func (bm *BrowserManager) createBrowser(ctx context.Context) (*ManagedBrowser, error) {
	// CREATE ALLOCATOR CONTEXT
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, bm.browserOptions...)

	// CREATE BROWSER CONTEXT
	browserCtx, browserCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
		chromedp.WithDebugf(log.Printf),
		chromedp.WithErrorf(log.Printf),
	)

	// CREATE BROWSER
	browser := &ManagedBrowser{
		ID:          uuid.New().String(),
		Context:     browserCtx,
		Cancel:      browserCancel,
		AllocCancel: allocCancel,
		Tabs:        make([]*ManagedTab, 0, bm.maxTabs),
		LastUsed:    time.Now(),
		Headless:    bm.headless,
	}

	// INITIALIZE BROWSER
	initCtx, cancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancel()

	err := chromedp.Run(initCtx, chromedp.Navigate("about:blank"))
	if err != nil {
		// CLEANUP ON ERROR
		browserCancel()
		allocCancel()
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	bm.logger.Info("Created new browser", map[string]any{
		"browserId": browser.ID,
		"headless":  browser.Headless,
	})

	return browser, nil
}

// GETTAB GETS OR CREATES A TAB IN THE BROWSER
func (b *ManagedBrowser) GetTab(ctx context.Context) (*ManagedTab, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// CHECK IF BROWSER IS TERMINATED
	if b.IsTerminated {
		return nil, fmt.Errorf("browser is terminated")
	}

	// LOOK FOR AVAILABLE TAB
	for _, tab := range b.Tabs {
		if !tab.InUse {
			tab.InUse = true
			tab.LastUsed = time.Now()
			return tab, nil
		}
	}

	// CREATE A NEW TAB
	tabCtx, tabCancel := chromedp.NewContext(b.Context)

	tab := &ManagedTab{
		ID:          uuid.New().String(),
		Context:     tabCtx,
		Cancel:      tabCancel,
		InUse:       true,
		LastUsed:    time.Now(),
		CurrentURL:  "about:blank",
		ParentID:    b.ID,
		NetworkLogs: make([]NetworkLog, 0),
	}

	// INITIALIZE TAB
	initCtx, cancel := context.WithTimeout(tabCtx, 10*time.Second)
	defer cancel()

	// INITIALIZE NETWORK MONITORING
	if err := chromedp.Run(initCtx,
		network.Enable(),
		page.Enable(),
		chromedp.Navigate("about:blank"),
	); err != nil {
		tabCancel()
		return nil, fmt.Errorf("failed to initialize tab: %w", err)
	}

	b.Tabs = append(b.Tabs, tab)
	b.TabCount++

	return tab, nil
}

// RELEASETAB RETURNS A TAB TO THE POOL
func (b *ManagedBrowser) ReleaseTab(tab *ManagedTab) {
	if tab == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// FIND THE TAB
	for i, t := range b.Tabs {
		if t.ID == tab.ID {
			b.Tabs[i].InUse = false
			b.Tabs[i].LastUsed = time.Now()

			// NAVIGATE TO BLANK PAGE TO RESET STATE AND STOP MEDIA
			go func(ctx context.Context) {
				// CREATE TIMEOUT CONTEXT
				timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				// NAVIGATE TO BLANK PAGE
				_ = chromedp.Run(timeoutCtx,
					page.StopLoading(),
					chromedp.Navigate("about:blank"),
				)
			}(t.Context)

			break
		}
	}
}

// CLEANUP PERIODICALLY CLEANS UP UNUSED BROWSERS AND TABS
func (bm *BrowserManager) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		bm.mu.Lock()

		for i := 0; i < len(bm.browsers); {
			browser := bm.browsers[i]
			browser.mu.Lock()

			if browser.IsTerminated {
				// REMOVE TERMINATED BROWSER
				browser.mu.Unlock()
				bm.browsers = append(bm.browsers[:i], bm.browsers[i+1:]...)
				continue
			}

			// CHECK IF BROWSER IS TOO OLD
			if time.Since(browser.LastUsed) > bm.browserLifetime {
				bm.logger.Info("Terminating old browser", map[string]any{
					"browserId": browser.ID,
					"age":       time.Since(browser.LastUsed).String(),
				})

				// TERMINATE BROWSER
				if browser.Cancel != nil {
					browser.Cancel()
				}
				if browser.AllocCancel != nil {
					browser.AllocCancel()
				}

				browser.IsTerminated = true
				browser.mu.Unlock()

				// REMOVE FROM LIST
				bm.browsers = append(bm.browsers[:i], bm.browsers[i+1:]...)
				continue
			}

			// CLEANUP UNUSED TABS
			for j := 0; j < len(browser.Tabs); {
				tab := browser.Tabs[j]
				if !tab.InUse && time.Since(tab.LastUsed) > 10*time.Minute {
					// CLOSE TAB
					if tab.Cancel != nil {
						tab.Cancel()
					}

					// REMOVE TAB
					browser.Tabs = append(browser.Tabs[:j], browser.Tabs[j+1:]...)
					browser.TabCount--
					continue
				}
				j++
			}

			browser.mu.Unlock()
			i++
		}

		bm.mu.Unlock()
	}
}

// NAVIGATE NAVIGATES TO A URL WITH THE GIVEN TAB
func (t *ManagedTab) Navigate(ctx context.Context, url string, timeout time.Duration) error {
	// CREATE TIMEOUT CONTEXT
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// SETUP NETWORK LOGGING
	var networkLogs []NetworkLog
	var networkMu sync.Mutex

	// LISTEN FOR NETWORK EVENTS
	chromedp.ListenTarget(t.Context, func(ev any) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			log := NetworkLog{
				Method:      e.Request.Method,
				URL:         e.Request.URL,
				RequestID:   string(e.RequestID),
				RequestType: string(e.Type),
				Headers:     make(map[string]any),
				Timestamp:   time.Now(),
			}

			// CONVERT HEADERS
			for k, v := range e.Request.Headers {
				log.Headers[k] = v
			}

			networkMu.Lock()
			networkLogs = append(networkLogs, log)
			networkMu.Unlock()

		case *network.EventResponseReceived:
			networkMu.Lock()
			for i, log := range networkLogs {
				if log.RequestID == string(e.RequestID) {
					networkLogs[i].Status = int(e.Response.Status)
					networkLogs[i].MimeType = e.Response.MimeType
					break
				}
			}
			networkMu.Unlock()
		}
	})

	// NAVIGATE TO URL
	err := chromedp.Run(timeoutCtx,
		network.Enable(),
		page.Enable(),
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// WAIT FOR PAGE TO LOAD
			var readyState string
			err := chromedp.Evaluate(`document.readyState`, &readyState).Do(ctx)
			if err != nil {
				return err
			}

			if readyState != "complete" {
				// WAIT FOR LOAD EVENT
				loadEventFired := make(chan struct{})
				chromedp.ListenTarget(ctx, func(ev any) {
					if _, ok := ev.(*page.EventLoadEventFired); ok {
						close(loadEventFired)
					}
				})

				select {
				case <-loadEventFired:
					// PAGE LOADED
				case <-timeoutCtx.Done():
					return timeoutCtx.Err()
				}
			}

			return nil
		}),
	)

	// UPDATE TAB STATE
	if err == nil {
		t.CurrentURL = url
		t.NetworkLogs = networkLogs
	}

	return err
}

// GETHTML GETS THE HTML CONTENT OF THE CURRENT PAGE
func (t *ManagedTab) GetHTML(ctx context.Context) (string, error) {
	var html string

	// CREATE TIMEOUT CONTEXT
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := chromedp.Run(timeoutCtx,
		chromedp.OuterHTML("html", &html),
	)

	return html, err
}

// EXTRACTCONTENT EXTRACTS CONTENT USING A SELECTOR
func (t *ManagedTab) ExtractContent(ctx context.Context, selector, attribute string, selectorType string) ([]string, error) {
	var results []string

	// CREATE TIMEOUT CONTEXT
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if selectorType == "xpath" {
		// EXTRACT USING XPATH
		var nodes []*cdp.Node
		err := chromedp.Run(timeoutCtx,
			chromedp.Nodes(selector, &nodes, chromedp.BySearch),
		)
		if err != nil {
			return nil, err
		}

		for range nodes {
			if attribute == "text" {
				// GET TEXT CONTENT
				var text string
				err := chromedp.Run(timeoutCtx,
					chromedp.Text(selector, &text, chromedp.BySearch),
				)
				if err != nil {
					continue
				}
				results = append(results, text)
			} else if attribute == "html" {
				// GET HTML CONTENT
				var html string
				err := chromedp.Run(timeoutCtx,
					chromedp.OuterHTML(selector, &html, chromedp.BySearch),
				)
				if err != nil {
					continue
				}
				results = append(results, html)
			} else {
				// GET ATTRIBUTE
				var attrValue string
				err := chromedp.Run(timeoutCtx,
					chromedp.AttributeValue(selector, attribute, &attrValue, nil, chromedp.BySearch),
				)
				if err != nil {
					continue
				}
				results = append(results, attrValue)
			}
		}
	} else {
		// DEFAULT TO CSS SELECTOR
		if attribute == "text" {
			// GET TEXT CONTENT
			var texts []string
			err := chromedp.Run(timeoutCtx,
				chromedp.Evaluate(fmt.Sprintf(`
					Array.from(document.querySelectorAll("%s")).map(el => el.textContent)
				`, selector), &texts),
			)
			if err != nil {
				return nil, err
			}
			results = texts
		} else if attribute == "html" {
			// GET HTML CONTENT
			var htmls []string
			err := chromedp.Run(timeoutCtx,
				chromedp.Evaluate(fmt.Sprintf(`
					Array.from(document.querySelectorAll("%s")).map(el => el.outerHTML)
				`, selector), &htmls),
			)
			if err != nil {
				return nil, err
			}
			results = htmls
		} else {
			// GET ATTRIBUTE
			var attrs []string
			err := chromedp.Run(timeoutCtx,
				chromedp.Evaluate(fmt.Sprintf(`
					Array.from(document.querySelectorAll("%s")).map(el => el.getAttribute("%s"))
				`, selector, attribute), &attrs),
			)
			if err != nil {
				return nil, err
			}
			results = attrs
		}
	}

	// FILTER OUT EMPTY RESULTS
	var filteredResults []string
	for _, result := range results {
		if result != "" {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults, nil
}

// EXECUTEJAVASCRIPT EXECUTES JAVASCRIPT IN THE PAGE
func (t *ManagedTab) ExecuteJavaScript(ctx context.Context, script string) (any, error) {
	var result any

	// CREATE TIMEOUT CONTEXT
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := chromedp.Run(timeoutCtx,
		chromedp.Evaluate(script, &result),
	)

	return result, err
}

// TAKESCREENSHOT TAKES A SCREENSHOT OF THE CURRENT PAGE
func (t *ManagedTab) TakeScreenshot(ctx context.Context) ([]byte, error) {
	var buf []byte

	// CREATE TIMEOUT CONTEXT
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := chromedp.Run(timeoutCtx,
		chromedp.CaptureScreenshot(&buf),
	)

	return buf, err
}

// CLICKELEMENT CLICKS AN ELEMENT ON THE PAGE
func (t *ManagedTab) ClickElement(ctx context.Context, selector string, selectorType string) error {
	// CREATE TIMEOUT CONTEXT
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var err error
	if selectorType == "xpath" {
		err = chromedp.Run(timeoutCtx,
			chromedp.Click(selector, chromedp.BySearch),
		)
	} else {
		err = chromedp.Run(timeoutCtx,
			chromedp.Click(selector),
		)
	}

	return err
}

// WAITFORELEMENT WAITS FOR AN ELEMENT TO APPEAR
func (t *ManagedTab) WaitForElement(ctx context.Context, selector string, selectorType string, timeout time.Duration) error {
	// CREATE TIMEOUT CONTEXT
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var err error
	if selectorType == "xpath" {
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitVisible(selector, chromedp.BySearch),
		)
	} else {
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitVisible(selector),
		)
	}

	return err
}

// TERMINATEBROWSER TERMINATES THE BROWSER
// TERMINATEBROWSER TERMINATES THE BROWSER
func (bm *BrowserManager) TerminateBrowser(browser *ManagedBrowser) {
	if browser == nil {
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	// FIND THE BROWSER
	for i, b := range bm.browsers {
		if b.ID == browser.ID {
			// TERMINATE BROWSER
			browser.mu.Lock()
			if browser.Cancel != nil {
				browser.Cancel()
			}
			if browser.AllocCancel != nil {
				browser.AllocCancel()
			}

			browser.IsTerminated = true
			browser.mu.Unlock()

			// REMOVE FROM LIST
			bm.browsers = append(bm.browsers[:i], bm.browsers[i+1:]...)

			bm.logger.Info("Terminated browser", map[string]any{
				"browserId": browser.ID,
			})

			break
		}
	}
}

// TERMINATEALL TERMINATES ALL BROWSERS
func (bm *BrowserManager) TerminateAll() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, browser := range bm.browsers {
		browser.mu.Lock()
		if !browser.IsTerminated {
			if browser.Cancel != nil {
				browser.Cancel()
			}
			if browser.AllocCancel != nil {
				browser.AllocCancel()
			}
			browser.IsTerminated = true
		}
		browser.mu.Unlock()
	}

	bm.browsers = make([]*ManagedBrowser, 0)

	bm.logger.Info("Terminated all browsers", nil)
}

// FETCHPAGE FETCHES A PAGE WITH AUTOMATIC BROWSER/TAB MANAGEMENT
func (bm *BrowserManager) FetchPage(ctx context.Context, url string) (string, error) {
	// GET BROWSER
	browser, err := bm.GetBrowser(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get browser: %w", err)
	}

	// GET TAB
	tab, err := browser.GetTab(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get tab: %w", err)
	}

	// ENSURE TAB IS RELEASED WHEN DONE
	defer browser.ReleaseTab(tab)

	// NAVIGATE TO URL
	if err := tab.Navigate(ctx, url, bm.defaultTimeout); err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	// GET HTML CONTENT
	html, err := tab.GetHTML(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get HTML: %w", err)
	}

	return html, nil
}

// EXTRACTELEMENTS EXTRACTS ELEMENTS WITH AUTOMATIC BROWSER/TAB MANAGEMENT
func (bm *BrowserManager) ExtractElements(ctx context.Context, url, selector, attribute, selectorType string) ([]string, error) {
	// GET BROWSER
	browser, err := bm.GetBrowser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser: %w", err)
	}

	// GET TAB
	tab, err := browser.GetTab(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab: %w", err)
	}

	// ENSURE TAB IS RELEASED WHEN DONE
	defer browser.ReleaseTab(tab)

	// NAVIGATE TO URL
	if err := tab.Navigate(ctx, url, bm.defaultTimeout); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// EXTRACT CONTENT
	results, err := tab.ExtractContent(ctx, selector, attribute, selectorType)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return results, nil
}

// EXECUTESCRIPT EXECUTES JAVASCRIPT WITH AUTOMATIC BROWSER/TAB MANAGEMENT
func (bm *BrowserManager) ExecuteScript(ctx context.Context, url, script string) (any, error) {
	// GET BROWSER
	browser, err := bm.GetBrowser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser: %w", err)
	}

	// GET TAB
	tab, err := browser.GetTab(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab: %w", err)
	}

	// ENSURE TAB IS RELEASED WHEN DONE
	defer browser.ReleaseTab(tab)

	// NAVIGATE TO URL
	if err := tab.Navigate(ctx, url, bm.defaultTimeout); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// EXECUTE JAVASCRIPT
	result, err := tab.ExecuteJavaScript(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	return result, nil
}

// FETCHPAGEWITHSCREENSHOT FETCHES A PAGE AND TAKES A SCREENSHOT
func (bm *BrowserManager) FetchPageWithScreenshot(ctx context.Context, url string) (string, []byte, error) {
	// GET BROWSER
	browser, err := bm.GetBrowser(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get browser: %w", err)
	}

	// GET TAB
	tab, err := browser.GetTab(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get tab: %w", err)
	}

	// ENSURE TAB IS RELEASED WHEN DONE
	defer browser.ReleaseTab(tab)

	// NAVIGATE TO URL
	if err := tab.Navigate(ctx, url, bm.defaultTimeout); err != nil {
		return "", nil, fmt.Errorf("navigation failed: %w", err)
	}

	// GET HTML CONTENT
	html, err := tab.GetHTML(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	// TAKE SCREENSHOT
	screenshot, err := tab.TakeScreenshot(ctx)
	if err != nil {
		return html, nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	return html, screenshot, nil
}

// CLICKANDNAVIGATE CLICKS AN ELEMENT AND WAITS FOR NAVIGATION
func (bm *BrowserManager) ClickAndNavigate(ctx context.Context, url, selector, selectorType string) (string, error) {
	// GET BROWSER
	browser, err := bm.GetBrowser(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get browser: %w", err)
	}

	// GET TAB
	tab, err := browser.GetTab(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get tab: %w", err)
	}

	// ENSURE TAB IS RELEASED WHEN DONE
	defer browser.ReleaseTab(tab)

	// NAVIGATE TO URL
	if err := tab.Navigate(ctx, url, bm.defaultTimeout); err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	// WAIT FOR NAVIGATION TO COMPLETE AFTER CLICK
	navigationComplete := make(chan bool, 1)
	chromedp.ListenTarget(tab.Context, func(ev any) {
		if _, ok := ev.(*page.EventLoadEventFired); ok {
			select {
			case navigationComplete <- true:
			default:
			}
		}
	})

	// CLICK ELEMENT
	if err := tab.ClickElement(ctx, selector, selectorType); err != nil {
		return "", fmt.Errorf("failed to click element: %w", err)
	}

	// WAIT FOR NAVIGATION WITH TIMEOUT
	select {
	case <-navigationComplete:
		// NAVIGATION COMPLETED
	case <-time.After(bm.defaultTimeout):
		// TIMEOUT - CONTINUE ANYWAY
	}

	// GET HTML CONTENT AFTER NAVIGATION
	html, err := tab.GetHTML(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get HTML after click: %w", err)
	}

	return html, nil
}

// HANDLENETWORKERROR CREATES A DETAILED ERROR WITH CONTEXT
func (t *ManagedTab) HandleNetworkError(url string, err error, stageID, stageName string) *utils.ScraperError {
	var screenshot []byte
	var html string

	// CREATE BACKGROUND CONTEXT
	ctx := context.Background()

	// TRY TO GET HTML FOR CONTEXT
	htmlCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	html, _ = t.GetHTML(htmlCtx)

	// TRY TO GET SCREENSHOT FOR CONTEXT
	ssCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	screenshot, _ = t.TakeScreenshot(ssCtx)

	// CREATE ERROR WITH CONTEXT
	scraperErr := utils.NewScraperError(
		err.Error(),
		url,
		"", // JOB ID WILL BE ADDED BY CALLER
		stageID,
		stageName,
	)

	// ADD ADDITIONAL CONTEXT
	scraperErr.WithHTML(html)

	// ADD SCREENSHOT IF AVAILABLE
	if len(screenshot) > 0 {
		scraperErr.WithScreenshot(fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(screenshot)))
	}

	// ADD NETWORK LOGS
	networkLogsJSON, _ := json.Marshal(t.NetworkLogs)
	scraperErr.WithMetadata("networkLogs", string(networkLogsJSON))

	return scraperErr
}
