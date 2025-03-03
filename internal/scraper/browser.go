package scraper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/nickheyer/Crepes/internal/models"
	utils "github.com/nickheyer/Crepes/internal/utils"
)

// CREATEBROWSERCONTEXT CREATES A BROWSER CONTEXT FOR SCRAPING
func CreateBrowserContext(ctx context.Context, job *models.ScrapingJob) (context.Context, context.CancelFunc) {
	// CHECK CHROME PERMISSIONS AND ENVIRONMENT
	CheckChromeEnvironment()

	// TRY WITH HEADLESS MODE FIRST
	log.Println("Attempting to create browser with headless mode")
	browserCtx, browserCancel, success := AttemptBrowserCreation(ctx, job, true)

	// IF HEADLESS FAILS, TRY NON-HEADLESS AS FALLBACK
	if !success {
		log.Println("Headless mode failed, attempting with non-headless mode")
		browserCancel() // Cancel the failed browser
		browserCtx, browserCancel, success = AttemptBrowserCreation(ctx, job, false)

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

// CHECKCHROMEENVIRONMENT CHECKS THE ENVIRONMENT FOR CHROME COMPATIBILITY
func CheckChromeEnvironment() {
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
	chromePath := FindChromePath()
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

// ATTEMPTBROWSERCREATION ATTEMPTS TO CREATE A CHROME BROWSER INSTANCE
func AttemptBrowserCreation(ctx context.Context, job *models.ScrapingJob, headless bool) (context.Context, context.CancelFunc, bool) {
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

// FINDCHROMEPATH FINDS THE CHROME EXECUTABLE PATH
func FindChromePath() string {
	// TRY COMMON CHROME/CHROMIUM LOCATIONS BASED ON OS
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

	// CHECK IF ANY OF THE PATHS EXIST
	for _, path := range paths {
		if utils.FileExists(path) {
			return path
		}
	}

	// TRY TO FIND USING EXEC.LOOKPATH
	for _, browser := range []string{"chrome", "google-chrome", "chromium", "chromium-browser", "msedge"} {
		if path, err := exec.LookPath(browser); err == nil {
			return path
		}
	}

	return ""
}

// FETCHWITHCHROMEDP FETCHES PAGE CONTENT USING CHROMEDP
func FetchWithChromedp(ctx context.Context, url string, timeout time.Duration) (string, error) {
	// CREATE SEPARATE TIMEOUT CONTEXT JUST FOR THIS NAVIGATION
	navCtx, navCancel := context.WithTimeout(ctx, timeout)
	defer navCancel()

	var htmlContent string
	err := chromedp.Run(navCtx,
		chromedp.Navigate(url),
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

// CLICKPAGINATIONLINK CLICKS A PAGINATION LINK AND RETURNS THE NEXT URL
func ClickPaginationLink(ctx context.Context, url string, paginationSelector string) (string, error) {
	// CREATE SEPARATE TIMEOUT CONTEXT JUST FOR THIS NAVIGATION
	navCtx, navCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer navCancel()

	var nextURL string
	err := chromedp.Run(navCtx,
		chromedp.Navigate(url),
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
		// TRY TO FIND AND CLICK THE NEXT PAGE BUTTON
		chromedp.ActionFunc(func(ctx context.Context) error {
			// CHECK IF THE SELECTOR EXISTS
			var exists bool
			if err := chromedp.Evaluate(`!!document.querySelector(`+strconv.Quote(paginationSelector)+`)`, &exists).Do(ctx); err != nil {
				return err
			}

			if !exists {
				return fmt.Errorf("pagination selector not found: %s", paginationSelector)
			}

			// GET HREF
			var href string
			if err := chromedp.AttributeValue(paginationSelector, "href", &href, &exists).Do(ctx); err == nil && exists {
				// STORE NEXT URL FOR LATER
				nextURL = MakeAbsoluteURL(url, href)
			}

			// CLICK THE ELEMENT
			return chromedp.Click(paginationSelector).Do(ctx)
		}),

		// WAIT FOR NAVIGATION TO COMPLETE
		chromedp.ActionFunc(func(ctx context.Context) error {
			// WAIT FOR THE PAGE TO LOAD AFTER CLICKING
			var readyState string
			if err := chromedp.Evaluate(`document.readyState`, &readyState).Do(ctx); err != nil {
				return err
			}
			if readyState != "complete" {
				// WAIT A BIT MORE
				return chromedp.Sleep(3 * time.Second).Do(ctx)
			}

			// IF WE DIDN'T GET THE HREF EARLIER, GET THE CURRENT URL
			if nextURL == "" {
				return chromedp.Evaluate(`window.location.href`, &nextURL).Do(ctx)
			}
			return nil
		}),
	)

	if err != nil {
		return "", fmt.Errorf("pagination click failed: %w", err)
	}

	return nextURL, nil
}
