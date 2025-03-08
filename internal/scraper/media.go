package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/config"
)

// MEDIA EXTRACTOR HANDLES EXTRACTING MEDIA FROM PAGES

// BROWSER POOL FOR MEDIA EXTRACTION
var (
	browserPool      []*ManagedBrowser
	browserPoolMutex sync.Mutex
	browserPoolSize  = 3 // DEFAULT POOL SIZE
)

// MEDIASOURCE REPRESENTS A DETECTED MEDIA SOURCE
type MediaSource struct {
	URL         string   `json:"url"`
	Type        string   `json:"type"`
	MimeType    string   `json:"mimeType,omitempty"`
	Quality     string   `json:"quality,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`
	Size        int64    `json:"size,omitempty"`
	Sources     []string `json:"sources,omitempty"`
	Method      string   `json:"method,omitempty"`
	Referer     string   `json:"referer,omitempty"`
	Confidence  float64  `json:"confidence"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
}

// INIT INITIALIZES THE BROWSER POOL
func InitBrowserPool(size int) {
	browserPoolMutex.Lock()
	defer browserPoolMutex.Unlock()

	if size > 0 {
		browserPoolSize = size
	}

	// CLEANUP OLD BROWSERS
	for _, browser := range browserPool {
		if browser.Cancel != nil {
			browser.Cancel()
		}
		if browser.AllocCancel != nil {
			browser.AllocCancel()
		}
	}

	browserPool = make([]*ManagedBrowser, 0, browserPoolSize)
}

// GETBROWSER GETS AN AVAILABLE BROWSER FROM THE POOL OR CREATES A NEW ONE
func GetBrowser(ctx context.Context, headless bool) (*ManagedBrowser, error) {
	browserPoolMutex.Lock()
	defer browserPoolMutex.Unlock()

	// LOOK FOR AN AVAILABLE BROWSER WITH MATCHING HEADLESS MODE
	for _, browser := range browserPool {
		if !browser.InUse && browser.Headless == headless {
			browser.InUse = true
			browser.LastUsed = time.Now()
			return browser, nil
		}
	}

	// IF POOL IS NOT FULL, CREATE A NEW BROWSER
	if len(browserPool) < browserPoolSize {
		browser, err := createBrowser(ctx, headless)
		if err != nil {
			return nil, err
		}

		browserPool = append(browserPool, browser)
		return browser, nil
	}

	// POOL IS FULL, LOOK FOR THE OLDEST BROWSER TO REPLACE
	oldestIdx := 0
	oldestTime := time.Now()

	for i, browser := range browserPool {
		if browser.LastUsed.Before(oldestTime) {
			oldestIdx = i
			oldestTime = browser.LastUsed
		}
	}

	// CLOSE THE OLDEST BROWSER
	if browserPool[oldestIdx].Cancel != nil {
		browserPool[oldestIdx].Cancel()
	}
	if browserPool[oldestIdx].AllocCancel != nil {
		browserPool[oldestIdx].AllocCancel()
	}

	// CREATE A NEW BROWSER
	browser, err := createBrowser(ctx, headless)
	if err != nil {
		return nil, err
	}

	browserPool[oldestIdx] = browser
	return browser, nil
}

// CREATEBROWSER CREATES A NEW BROWSER INSTANCE
func createBrowser(ctx context.Context, headless bool) (*ManagedBrowser, error) {
	log.Printf("Creating new browser (headless: %v)", headless)

	// DEFINE BROWSER OPTIONS
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("autoplay-policy", "no-user-gesture-required"),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("allow-running-insecure-content", true),
		chromedp.Flag("user-agent", config.GetRandomUserAgent()),
		chromedp.WindowSize(1920, 1080),
	)

	// CREATE ALLOCATOR CONTEXT
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)

	// CREATE BROWSER CONTEXT
	browserCtx, browserCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
		chromedp.WithDebugf(log.Printf),
		chromedp.WithErrorf(log.Printf),
	)

	// CREATE TIMEOUT CONTEXT FOR INITIALIZATION
	timeoutCtx, cancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancel()

	// INITIALIZE BROWSER (NAVIGATE TO BLANK PAGE)
	if err := chromedp.Run(timeoutCtx, chromedp.Navigate("about:blank")); err != nil {
		allocCancel()
		browserCancel()
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	// return &ManagedBrowser{
	// 	ctx:         browserCtx,
	// 	cancel:      browserCancel,
	// 	allocCtx:    allocCtx,
	// 	allocCancel: allocCancel,
	// 	inUse:       true,
	// 	headless:    headless,
	// 	id:          uuid.New().String(),
	// 	lastUsed:    time.Now(),
	// }, nil

	return &ManagedBrowser{
		ID:          uuid.New().String(),
		Context:     browserCtx,
		Cancel:      browserCancel,
		AllocCancel: allocCancel,
		Tabs:        make([]*ManagedTab, 0),
		LastUsed:    time.Now(),
		Headless:    headless,
		InUse:       true,
	}, nil
}

// RELEASEBROWSER RETURNS A BROWSER TO THE POOL
func ReleaseBrowser(browser *ManagedBrowser) {
	if browser == nil {
		return
	}

	browserPoolMutex.Lock()
	defer browserPoolMutex.Unlock()

	// FIND THE BROWSER IN THE POOL
	for i, b := range browserPool {
		if b.ID == browser.ID {
			browserPool[i].InUse = false
			browserPool[i].LastUsed = time.Now()
			break
		}
	}
}

// CLOSEBROWSERPOOL CLOSES ALL BROWSERS IN THE POOL
func CloseBrowserPool() {
	browserPoolMutex.Lock()
	defer browserPoolMutex.Unlock()

	for _, browser := range browserPool {
		if browser.Cancel != nil {
			browser.Cancel()
		}
		if browser.AllocCancel != nil {
			browser.AllocCancel()
		}
	}

	browserPool = make([]*ManagedBrowser, 0)
}

// EXTRACTMEDIASTREAMS EXTRACTS MEDIA STREAMS FROM A PAGE
func ExtractMediaStreams(ctx context.Context, url string, headless bool) ([]MediaSource, error) {
	log.Printf("Extracting media from %s (headless: %v)", url, headless)

	// CREATE A MULTI-STRATEGY RESULT SET
	var results []MediaSource

	// STRATEGY 1: DIRECT NETWORK TRAFFIC ANALYSIS WITH BROWSER
	browser, err := GetBrowser(ctx, headless)
	if err != nil {
		log.Printf("Failed to get browser: %v, trying fallback methods", err)
	} else {
		defer ReleaseBrowser(browser)

		// COLLECT MEDIA SOURCES USING NETWORK INTERCEPTION
		sources, err := extractMediaWithNetworkAnalysis(browser.Context, url)
		if err != nil {
			log.Printf("Network analysis extraction failed: %v", err)
		} else if len(sources) > 0 {
			log.Printf("Found %d media sources via network analysis", len(sources))
			results = append(results, sources...)
		}
	}

	// STRATEGY 2: DOM ANALYSIS FOR COMMON VIDEO PATTERNS
	if browser != nil {
		sources, err := extractMediaFromDOM(browser.Context, url)
		if err != nil {
			log.Printf("DOM extraction failed: %v", err)
		} else if len(sources) > 0 {
			log.Printf("Found %d media sources via DOM analysis", len(sources))
			results = append(results, sources...)
		}
	}

	// STRATEGY 3: JAVASCRIPT EXECUTION FOR PLAYER EXTRACTION
	if browser != nil {
		sources, err := extractMediaWithJavaScript(browser.Context, url)
		if err != nil {
			log.Printf("JavaScript extraction failed: %v", err)
		} else if len(sources) > 0 {
			log.Printf("Found %d media sources via JavaScript extraction", len(sources))
			results = append(results, sources...)
		}
	}

	// STRATEGY 4: DIRECT HLS/DASH MANIFEST ANALYSIS
	// THIS CAN WORK EVEN WITHOUT A BROWSER
	hlsSources, _ := extractHLSManifests(ctx, url, results)
	if len(hlsSources) > 0 {
		log.Printf("Found %d HLS/DASH sources", len(hlsSources))
		results = append(results, hlsSources...)
	}

	// DE-DUPLICATE RESULTS
	results = deduplicateMediaSources(results)

	// RANK RESULTS BY CONFIDENCE AND QUALITY
	rankMediaSources(results)

	log.Printf("Extraction complete, found %d unique media sources", len(results))

	return results, nil
}

// EXTRACTMEDIAWITHNETWORKANALYSIS EXTRACTS MEDIA BY ANALYZING NETWORK TRAFFIC
func extractMediaWithNetworkAnalysis(ctx context.Context, pageURL string) ([]MediaSource, error) {
	// INITIALIZE MEDIA SOURCES SLICE
	var mediaSources []MediaSource

	// INITIALIZE NETWORK TRACKING
	requestURLs := make(map[string]bool)
	mediaURLs := make(map[string]string) // URL -> MIME TYPE

	// START NETWORK MONITORING
	chromedp.ListenTarget(ctx, func(ev any) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			requestURLs[e.Request.URL] = true

			// CHECK URL FOR MEDIA PATTERNS
			if isMediaURL(e.Request.URL) {
				log.Printf("Detected potential media request: %s", e.Request.URL)
				mediaURLs[e.Request.URL] = fmt.Sprintf("%s", e.Request.Headers["Content-Type"])
			}

		case *network.EventResponseReceived:
			// CHECK CONTENT TYPE FOR MEDIA
			if isMediaContentType(e.Response.MimeType) {
				log.Printf("Detected media response: %s (%s)", e.Response.URL, e.Response.MimeType)
				mediaURLs[e.Response.URL] = e.Response.MimeType
			}
		}
	})

	// NAVIGATION TIMEOUT
	navCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	// NAVIGATE TO THE PAGE
	err := chromedp.Run(navCtx,
		network.Enable(),
		chromedp.Navigate(pageURL),
		chromedp.Sleep(5*time.Second), // ALLOW TIME FOR MEDIA TO LOAD
	)

	if err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// SIMULATE USER INTERACTION TO TRIGGER MEDIA PLAYBACK
	simulateUserInteraction(ctx)

	// WAIT A BIT LONGER FOR MEDIA TO LOAD AFTER INTERACTION
	time.Sleep(3 * time.Second)

	// CONVERT DETECTED MEDIA URLS TO MEDIA SOURCES
	for mediaURL, mimeType := range mediaURLs {
		confidence := calculateMediaConfidence(mediaURL, mimeType)

		// CREATE MEDIA SOURCE
		mediaSource := MediaSource{
			URL:        mediaURL,
			Type:       getMediaTypeFromURL(mediaURL, mimeType),
			MimeType:   mimeType,
			Method:     "network",
			Referer:    pageURL,
			Confidence: confidence,
		}

		mediaSources = append(mediaSources, mediaSource)
	}

	return mediaSources, nil
}

// SIMULATEUSERINTERACTION SIMULATES USER INTERACTION WITH THE PAGE
func simulateUserInteraction(ctx context.Context) {
	// CREATE A BRIEF TIMEOUT FOR INTERACTIONS
	interactCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// TRY TO FIND AND CLICK PLAY BUTTONS AND VIDEO ELEMENTS
	_ = chromedp.Run(interactCtx, chromedp.Tasks{
		// CLICK COMMON PLAY BUTTONS
		chromedp.Evaluate(`
			(function() {
				// TRY TO FIND AND CLICK PLAY BUTTONS
				const playButtons = [
					document.querySelector('.play-button'),
					document.querySelector('[class*="play"]'),
					document.querySelector('[id*="play"]'),
					document.querySelector('button[aria-label*="Play"]'),
					document.querySelector('button[title*="Play"]'),
					document.querySelector('.ytp-play-button'), // YOUTUBE
					document.querySelector('.vjs-play-control')  // VIDEO.JS
				];
				
				// CLICK FIRST AVAILABLE PLAY BUTTON
				for (const button of playButtons) {
					if (button) {
						console.log('Clicking play button:', button);
						button.click();
						break;
					}
				}
				
				// FIND ALL VIDEO ELEMENTS
				const videos = document.querySelectorAll('video');
				console.log('Found', videos.length, 'video elements');
				
				// TRY TO PLAY ALL VIDEOS
				videos.forEach(video => {
					try {
						if (video.paused) {
							console.log('Playing video:', video);
							video.play().catch(e => console.log('Video play error:', e));
						}
						
						// ADD CLICK HANDLER
						video.click();
					} catch (e) {
						console.log('Error interacting with video:', e);
					}
				});
				
				return videos.length;
			})()
		`, nil),

		// WAIT A MOMENT
		chromedp.Sleep(2 * time.Second),

		// TRY CLICKING IN THE CENTER OF THE PAGE (COMMON FOR OVERLAY PLAYERS)
		chromedp.Evaluate(`
			(function() {
				// GET PAGE DIMENSIONS
				const width = window.innerWidth;
				const height = window.innerHeight;
				
				// CREATE AND DISPATCH CLICK EVENT IN CENTER OF PAGE
				const clickEvent = new MouseEvent('click', {
					bubbles: true,
					cancelable: true,
					view: window,
					clientX: width / 2,
					clientY: height / 2
				});
				
				document.elementFromPoint(width / 2, height / 2)?.dispatchEvent(clickEvent);
				
				return {width, height};
			})()
		`, nil),
	})
}

// EXTRACTMEDIAFROMDOM EXTRACTS MEDIA SOURCES FROM THE DOM
func extractMediaFromDOM(ctx context.Context, pageURL string) ([]MediaSource, error) {
	var mediaSources []MediaSource

	// EXECUTE JAVASCRIPT TO EXTRACT MEDIA ELEMENTS
	var mediaElements []struct {
		Type    string `json:"type"`
		Src     string `json:"src"`
		Sources []struct {
			Src  string `json:"src"`
			Type string `json:"type"`
		} `json:"sources"`
		Poster   string  `json:"poster"`
		Width    int     `json:"width"`
		Height   int     `json:"height"`
		Duration float64 `json:"duration"`
	}

	// RUN THE EXTRACTION
	err := chromedp.Run(ctx, chromedp.Evaluate(`
		(function() {
			const results = [];
			
			// EXTRACT VIDEO ELEMENTS
			document.querySelectorAll('video').forEach(video => {
				const sources = [];
				video.querySelectorAll('source').forEach(source => {
					sources.push({
						src: source.src,
						type: source.type
					});
				});
				
				results.push({
					type: 'video',
					src: video.src,
					sources: sources,
					poster: video.poster,
					width: video.videoWidth,
					height: video.videoHeight,
					duration: video.duration
				});
			});
			
			// EXTRACT AUDIO ELEMENTS
			document.querySelectorAll('audio').forEach(audio => {
				const sources = [];
				audio.querySelectorAll('source').forEach(source => {
					sources.push({
						src: source.src,
						type: source.type
					});
				});
				
				results.push({
					type: 'audio',
					src: audio.src,
					sources: sources,
					duration: audio.duration
				});
			});
			
			// EXTRACT IFRAME ELEMENTS THAT MIGHT CONTAIN MEDIA
			document.querySelectorAll('iframe').forEach(iframe => {
				if (iframe.src.includes('youtube.com') || 
					iframe.src.includes('vimeo.com') || 
					iframe.src.includes('dailymotion.com') ||
					iframe.src.includes('player')) {
					
					results.push({
						type: 'iframe',
						src: iframe.src,
						width: iframe.width,
						height: iframe.height
					});
				}
			});
			
			// RETURN RESULTS
			return results;
		})()
	`, &mediaElements))

	if err != nil {
		return nil, fmt.Errorf("DOM extraction failed: %w", err)
	}

	// CONVERT TO MEDIA SOURCES
	for _, element := range mediaElements {
		// ADD MAIN SOURCE IF PRESENT
		if element.Src != "" {
			mediaSources = append(mediaSources, MediaSource{
				URL:        element.Src,
				Type:       element.Type,
				Resolution: fmt.Sprintf("%dx%d", element.Width, element.Height),
				Method:     "dom",
				Referer:    pageURL,
				Confidence: 0.7, // DEFAULT CONFIDENCE FOR DOM SOURCES
			})
		}

		// ADD SOURCES FROM SOURCE ELEMENTS
		for _, source := range element.Sources {
			if source.Src != "" {
				mediaSources = append(mediaSources, MediaSource{
					URL:        source.Src,
					Type:       element.Type,
					MimeType:   source.Type,
					Resolution: fmt.Sprintf("%dx%d", element.Width, element.Height),
					Method:     "dom-source",
					Referer:    pageURL,
					Confidence: 0.8, // HIGHER CONFIDENCE FOR EXPLICIT SOURCES
				})
			}
		}
	}

	return mediaSources, nil
}

// EXTRACTMEDIAWITHJAVASCRIPT USES JAVASCRIPT INJECTION TO EXTRACT MEDIA
func extractMediaWithJavaScript(ctx context.Context, pageURL string) ([]MediaSource, error) {
	var mediaSources []MediaSource

	// INJECT ADVANCED MEDIA EXTRACTION SCRIPT
	var extractedSources []struct {
		URL         string   `json:"url"`
		Type        string   `json:"type"`
		Quality     string   `json:"quality"`
		Player      string   `json:"player"`
		IsEncrypted bool     `json:"isEncrypted"`
		Sources     []string `json:"sources"`
	}

	// RUN THE EXTRACTION SCRIPT
	err := chromedp.Run(ctx, chromedp.Evaluate(`
		(function() {
			const results = [];
			
			// ATTEMPT TO DETECT KNOWN PLAYER LIBRARIES
			function detectPlayers() {
				const players = [];
				
				// JW PLAYER
				if (typeof jwplayer !== 'undefined') {
					players.push('jwplayer');
					
					// GET SOURCES FROM ALL JW PLAYER INSTANCES
					try {
						const instances = jwplayer();
						const config = instances.getConfig();
						if (config && config.sources) {
							config.sources.forEach(source => {
								results.push({
									url: source.file,
									type: 'video',
									quality: source.label,
									player: 'jwplayer',
									isEncrypted: source.drm ? true : false
								});
							});
						}
					} catch(e) {
						console.log('JW Player extraction error', e);
					}
				}
				
				// VIDEO.JS
				if (typeof videojs !== 'undefined') {
					players.push('videojs');
					
					// TRY TO GET PLAYERS
					try {
						const players = document.querySelectorAll('.video-js');
						players.forEach(playerEl => {
							const player = videojs.getPlayer(playerEl);
							if (player) {
								const src = player.src();
								if (src) {
									results.push({
										url: typeof src === 'string' ? src : src[0].src,
										type: 'video',
										player: 'videojs'
									});
								}
							}
						});
					} catch(e) {
						console.log('VideoJS extraction error', e);
					}
				}
				
				// FLOWPLAYER
				if (typeof flowplayer !== 'undefined' || document.querySelector('.flowplayer')) {
					players.push('flowplayer');
					
					// TRY TO EXTRACT SOURCES
					try {
						document.querySelectorAll('.flowplayer').forEach(player => {
							const data = player.dataset;
							if (data && data.src) {
								results.push({
									url: data.src,
									type: 'video',
									player: 'flowplayer'
								});
							}
						});
					} catch(e) {
						console.log('Flowplayer extraction error', e);
					}
				}
				
				// HULU PLAYER
				if (typeof VilosPlayer !== 'undefined') {
					players.push('hulu');
				}
				
				// HTML5 MEDIA ELEMENT API
				if (typeof HTMLMediaElement !== 'undefined') {
					players.push('html5');
					
					// TRY TO GET SOURCES FROM MEDIA ELEMENTS
					try {
						const mediaElements = document.querySelectorAll('video, audio');
						mediaElements.forEach(media => {
							if (media.src) {
								results.push({
									url: media.src,
									type: media.tagName.toLowerCase(),
									player: 'html5'
								});
							}
							
							// CHECK MEDIASOURCE API
							if (media.srcObject) {
								results.push({
									url: 'mediastream:' + (media.id || 'unknown'),
									type: media.tagName.toLowerCase(),
									player: 'html5-mediastream'
								});
							}
						});
					} catch(e) {
						console.log('HTML5 media extraction error', e);
					}
				}
				
				// EXTRACT URLS FROM JSON IN SCRIPT TAGS
				try {
					document.querySelectorAll('script').forEach(script => {
						if (script.innerText) {
							// LOOK FOR COMMON PATTERNS
							extractURLsFromText(script.innerText);
						}
					});
				} catch(e) {
					console.log('Script extraction error', e);
				}
				
				return players;
			}
			
			// EXTRACT URLS FROM TEXT USING REGEX
			function extractURLsFromText(text) {
				// MATCH MEDIA URLS IN JSON OR JS
				const mediaPatterns = [
					/"(?:file|src|source|url)"\s*:\s*"(https?:\/\/[^"]+\.(mp4|webm|m3u8|mpd))/gi,
					/'(?:file|src|source|url)'\s*:\s*'(https?:\/\/[^']+\.(mp4|webm|m3u8|mpd))/gi,
					/(?:file|src|source|url):\s*["'](https?:\/\/[^"']+\.(mp4|webm|m3u8|mpd))["']/gi,
					/source\s*=\s*["'](https?:\/\/[^"']+\.(mp4|webm|m3u8|mpd))["']/gi,
					/(https?:\/\/[^"'\s]+\.(mp4|webm|m3u8|mpd))/gi
				];
				
				for (const pattern of mediaPatterns) {
					let match;
					while ((match = pattern.exec(text)) !== null) {
						const url = match[1];
						if (url && isValidURL(url) && !results.some(r => r.url === url)) {
							results.push({
								url: url,
								type: getTypeFromExtension(url),
								player: 'script'
							});
						}
					}
				}
			}
			
			// GET TYPE FROM FILE EXTENSION
			function getTypeFromExtension(url) {
				const ext = url.split('.').pop().toLowerCase();
				if (['mp4', 'webm', 'mov', 'avi', 'm3u8', 'mpd'].includes(ext)) {
					return 'video';
				}
				if (['mp3', 'wav', 'ogg', 'aac'].includes(ext)) {
					return 'audio';
				}
				return 'unknown';
			}
			
			// CHECK IF URL IS VALID
			function isValidURL(url) {
				try {
					const parsedURL = new URL(url);
					return ['http:', 'https:'].includes(parsedURL.protocol);
				} catch(e) {
					return false;
				}
			}
			
			// DETECT ANY WINDOW VARIABLES CONTAINING MEDIA URLS
			function scanWindowVariables() {
				const urlProps = ['src', 'url', 'file', 'source', 'streamUrl', 'hlsUrl'];
				const results = [];
				
				// RECURSIVELY SCAN OBJECT PROPERTIES UP TO DEPTH 3
				function scanObject(obj, path, depth) {
					if (depth > 3) return; // LIMIT RECURSION
					if (!obj || typeof obj !== 'object') return;
					
					Object.keys(obj).forEach(key => {
						const val = obj[key];
						const newPath = path ? path + '.' + key : key;
						
						// IF IT'S A MEDIA URL
						if (typeof val === 'string' && val.match(/^https?:\/\/.+\.(mp4|webm|m3u8|mpd)/i)) {
							results.push({
								url: val,
								type: getTypeFromExtension(val),
								player: 'window.' + newPath
							});
						}
						
						// IF PROPERTY NAME SUGGESTS MEDIA URL
						if (urlProps.includes(key.toLowerCase()) && typeof val === 'string' &&
							val.match(/^https?:\/\//)) {
							results.push({
								url: val,
								type: 'unknown',
								player: 'window.' + newPath
							});
						}
						
						// CHECK FOR OBJECTS WITH SOURCES ARRAY
						if (key === 'sources' && Array.isArray(val)) {
							val.forEach(source => {
								if (source && typeof source === 'object' && 
									typeof source.file === 'string' && source.file.match(/^https?:\/\//)) {
									results.push({
										url: source.file,
										type: source.type || 'unknown',
										quality: source.label || source.quality,
										player: 'window.' + path
									});
								}
							});
						}
						
						// RECURSE INTO OBJECT PROPERTIES
						if (val && typeof val === 'object') {
							scanObject(val, newPath, depth + 1);
						}
					});
				}
				
				// START SCAN
				try {
					scanObject(window, '', 0);
				} catch(e) {
					console.log('Window scan error', e);
				}
				
				return results;
			}
			
			// RUN ALL DETECTION METHODS
			const players = detectPlayers();
			console.log('Detected players:', players);
			
			// SCAN WINDOW VARIABLES
			const windowVars = scanWindowVariables();
			windowVars.forEach(item => {
				if (!results.some(r => r.url === item.url)) {
					results.push(item);
				}
			});
			
			return results;
		})()
	`, &extractedSources))

	if err != nil {
		return nil, fmt.Errorf("JavaScript extraction failed: %w", err)
	}

	// CONVERT TO MEDIA SOURCES
	for _, source := range extractedSources {
		if source.URL != "" {
			confidence := 0.6 // BASE CONFIDENCE

			// ADJUST CONFIDENCE BASED ON SOURCE
			if source.Player == "jwplayer" || source.Player == "videojs" {
				confidence = 0.9 // KNOWN PLAYERS ARE MORE RELIABLE
			}

			mediaSources = append(mediaSources, MediaSource{
				URL:        source.URL,
				Type:       source.Type,
				Quality:    source.Quality,
				Method:     fmt.Sprintf("js-%s", source.Player),
				Referer:    pageURL,
				Confidence: confidence,
				Sources:    source.Sources,
			})
		}
	}

	return mediaSources, nil
}

// EXTRACTHLSMANIFESTS ANALYZES HLS AND DASH MANIFESTS
func extractHLSManifests(ctx context.Context, pageURL string, knownSources []MediaSource) ([]MediaSource, error) {
	var mediaSources []MediaSource

	// CREATE HTTP CLIENT
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// PROCESS KNOWN SOURCES FOR HLS/DASH MANIFESTS
	for _, source := range knownSources {
		// ONLY CHECK STREAMING MANIFEST FORMATS
		if !strings.HasSuffix(source.URL, ".m3u8") && !strings.HasSuffix(source.URL, ".mpd") {
			continue
		}

		// FETCH MANIFEST
		req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
		if err != nil {
			continue
		}

		// SET REFERER
		req.Header.Set("Referer", pageURL)
		req.Header.Set("User-Agent", config.GetRandomUserAgent())

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		// READ MANIFEST CONTENT
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// PROCESS HLS MANIFEST
		if strings.HasSuffix(source.URL, ".m3u8") {
			// PARSE MANIFEST
			variants := parseHLSManifest(string(body))

			// ADD VARIANTS AS SOURCES
			for _, variant := range variants {
				absoluteURL := variant.URL
				if !strings.HasPrefix(absoluteURL, "http") {
					// RESOLVE RELATIVE URL
					relURL, err := url.Parse(variant.URL)
					if err != nil {
						continue
					}

					manifestURL, err := url.Parse(source.URL)
					if err != nil {
						continue
					}

					absoluteURL = manifestURL.ResolveReference(relURL).String()
				}

				mediaSources = append(mediaSources, MediaSource{
					URL:        absoluteURL,
					Type:       "video",
					MimeType:   "application/x-mpegURL",
					Quality:    variant.Quality,
					Resolution: variant.Resolution,
					Method:     "hls-variant",
					Referer:    source.URL,
					Confidence: 0.85,
				})
			}
		}

		// PROCESS DASH MANIFEST (MPD)
		if strings.HasSuffix(source.URL, ".mpd") {
			// BASIC MPD PARSING
			adaptationSets := parseDASHManifest(string(body))

			// ADD ADAPTATION SETS AS SOURCES
			for _, adaptSet := range adaptationSets {
				absoluteURL := adaptSet.URL
				if !strings.HasPrefix(absoluteURL, "http") {
					// RESOLVE RELATIVE URL
					relURL, err := url.Parse(adaptSet.URL)
					if err != nil {
						continue
					}

					manifestURL, err := url.Parse(source.URL)
					if err != nil {
						continue
					}

					absoluteURL = manifestURL.ResolveReference(relURL).String()
				}

				mediaSources = append(mediaSources, MediaSource{
					URL:        absoluteURL,
					Type:       adaptSet.Type,
					MimeType:   adaptSet.MimeType,
					Quality:    adaptSet.Quality,
					Resolution: adaptSet.Resolution,
					Method:     "dash-adaptation",
					Referer:    source.URL,
					Confidence: 0.85,
				})
			}
		}
	}

	return mediaSources, nil
}

// HLSVARIANT REPRESENTS AN HLS VARIANT STREAM
type HLSVariant struct {
	URL        string
	Bandwidth  int
	Resolution string
	Quality    string
}

// PARSEHLSMANIFEST PARSES AN HLS MANIFEST TO EXTRACT VARIANT STREAMS
func parseHLSManifest(content string) []HLSVariant {
	var variants []HLSVariant

	// SPLIT INTO LINES
	lines := strings.Split(content, "\n")

	var currentVariant HLSVariant
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// CHECK FOR EXT-X-STREAM-INF
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			// PARSE ATTRIBUTES
			attributes := line[len("#EXT-X-STREAM-INF:"):]

			// EXTRACT BANDWIDTH
			bandwidthMatch := regexp.MustCompile(`BANDWIDTH=(\d+)`).FindStringSubmatch(attributes)
			if len(bandwidthMatch) > 1 {
				bandwidth, _ := strconv.Atoi(bandwidthMatch[1])
				currentVariant.Bandwidth = bandwidth

				// DETERMINE QUALITY LABEL BASED ON BANDWIDTH
				if bandwidth > 5000000 {
					currentVariant.Quality = "High"
				} else if bandwidth > 2000000 {
					currentVariant.Quality = "Medium"
				} else {
					currentVariant.Quality = "Low"
				}
			}

			// EXTRACT RESOLUTION
			resolutionMatch := regexp.MustCompile(`RESOLUTION=(\d+x\d+)`).FindStringSubmatch(attributes)
			if len(resolutionMatch) > 1 {
				currentVariant.Resolution = resolutionMatch[1]
			}

			// GET THE NEXT LINE FOR URL
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if !strings.HasPrefix(nextLine, "#") {
					currentVariant.URL = nextLine
					variants = append(variants, currentVariant)
					currentVariant = HLSVariant{} // RESET
				}
			}
		}
	}

	return variants
}

// DASHADAPTATION REPRESENTS A DASH ADAPTATION SET
type DASHAdaptation struct {
	URL        string
	Type       string
	MimeType   string
	Bandwidth  int
	Resolution string
	Quality    string
}

// PARSEDASGMANIFEST VERY BASIC DASH MANIFEST PARSER
func parseDASHManifest(content string) []DASHAdaptation {
	var adaptations []DASHAdaptation

	// THIS IS A VERY BASIC PARSER - FOR PRODUCTION USE, CONSIDER A PROPER XML PARSER

	// LOOK FOR ADAPTATION SETS
	adaptationSets := regexp.MustCompile(`<AdaptationSet[^>]*>(.*?)</AdaptationSet>`).FindAllStringSubmatch(content, -1)

	for _, adaptSet := range adaptationSets {
		if len(adaptSet) < 2 {
			continue
		}

		// DETERMINE TYPE
		mimeType := ""
		contentType := "unknown"

		mimeTypeMatch := regexp.MustCompile(`mimeType=["']([^"']+)["']`).FindStringSubmatch(adaptSet[0])
		if len(mimeTypeMatch) > 1 {
			mimeType = mimeTypeMatch[1]
			if strings.HasPrefix(mimeType, "video/") {
				contentType = "video"
			} else if strings.HasPrefix(mimeType, "audio/") {
				contentType = "audio"
			}
		}

		// FIND REPRESENTATIONS
		representations := regexp.MustCompile(`<Representation[^>]*>(.*?)</Representation>`).FindAllStringSubmatch(adaptSet[1], -1)

		for _, repr := range representations {
			var adaptation DASHAdaptation
			adaptation.Type = contentType
			adaptation.MimeType = mimeType

			// GET BANDWIDTH
			bandwidthMatch := regexp.MustCompile(`bandwidth=["'](\d+)["']`).FindStringSubmatch(repr[0])
			if len(bandwidthMatch) > 1 {
				bandwidth, _ := strconv.Atoi(bandwidthMatch[1])
				adaptation.Bandwidth = bandwidth

				// DETERMINE QUALITY LABEL
				if bandwidth > 5000000 {
					adaptation.Quality = "High"
				} else if bandwidth > 2000000 {
					adaptation.Quality = "Medium"
				} else {
					adaptation.Quality = "Low"
				}
			}

			// GET RESOLUTION
			widthMatch := regexp.MustCompile(`width=["'](\d+)["']`).FindStringSubmatch(repr[0])
			heightMatch := regexp.MustCompile(`height=["'](\d+)["']`).FindStringSubmatch(repr[0])

			if len(widthMatch) > 1 && len(heightMatch) > 1 {
				adaptation.Resolution = fmt.Sprintf("%sx%s", widthMatch[1], heightMatch[1])
			}

			// GET SEGMENT URL
			segmentMatch := regexp.MustCompile(`<BaseURL[^>]*>(.*?)</BaseURL>`).FindStringSubmatch(repr[0])
			if len(segmentMatch) > 1 {
				adaptation.URL = segmentMatch[1]
				adaptations = append(adaptations, adaptation)
			}
		}
	}

	return adaptations
}

// HELPER FUNCTIONS

// ISMEDIAURL CHECKS IF A URL LIKELY POINTS TO MEDIA
func isMediaURL(url string) bool {
	// CHECK COMMON MEDIA EXTENSIONS
	mediaExtensions := []string{".mp4", ".webm", ".m3u8", ".mpd", ".mp3", ".wav", ".ogg", ".mov", ".avi"}
	for _, ext := range mediaExtensions {
		if strings.HasSuffix(strings.ToLower(url), ext) {
			return true
		}
	}

	// CHECK FOR COMMON MEDIA PATTERNS
	mediaPatterns := []string{
		"/media/", "/video/", "/audio/", "/stream/",
		"videoplayback", "get_video", "manifest", "playlist",
		"stream_url", "play_url", "source",
	}

	urlLower := strings.ToLower(url)
	for _, pattern := range mediaPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}

	return false
}

// ISMEDIACONTENTTYPE CHECKS IF A CONTENT TYPE IS FOR MEDIA
func isMediaContentType(contentType string) bool {
	mediaTypes := []string{
		"video/", "audio/",
		"application/x-mpegURL", "application/vnd.apple.mpegURL",
		"application/dash+xml",
	}

	for _, mediaType := range mediaTypes {
		if strings.HasPrefix(contentType, mediaType) {
			return true
		}
	}

	return false
}

// GETMEDIATYPEFROMURL DETERMINES MEDIA TYPE FROM URL AND MIME TYPE
// GETMEDIATYPEFROMURL DETERMINES MEDIA TYPE FROM URL AND MIME TYPE
func getMediaTypeFromURL(url, mimeType string) string {
	// CHECK MIME TYPE FIRST
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	} else if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	} else if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}

	// CHECK URL EXTENSIONS
	urlLower := strings.ToLower(url)

	// VIDEO FORMATS
	if strings.HasSuffix(urlLower, ".mp4") ||
		strings.HasSuffix(urlLower, ".webm") ||
		strings.HasSuffix(urlLower, ".mov") ||
		strings.HasSuffix(urlLower, ".avi") ||
		strings.HasSuffix(urlLower, ".m3u8") ||
		strings.HasSuffix(urlLower, ".mpd") ||
		strings.HasSuffix(urlLower, ".ts") {
		return "video"
	}

	// AUDIO FORMATS
	if strings.HasSuffix(urlLower, ".mp3") ||
		strings.HasSuffix(urlLower, ".wav") ||
		strings.HasSuffix(urlLower, ".ogg") ||
		strings.HasSuffix(urlLower, ".aac") {
		return "audio"
	}

	// CHECK URL PATTERNS
	if strings.Contains(urlLower, "video") ||
		strings.Contains(urlLower, "stream") ||
		strings.Contains(urlLower, "player") ||
		strings.Contains(urlLower, "watch") {
		return "video"
	}

	if strings.Contains(urlLower, "audio") || strings.Contains(urlLower, "sound") {
		return "audio"
	}

	// DEFAULT TO UNKNOWN
	return "unknown"
}

// CALCULATEMEDIACONFIDENCE CALCULATES THE CONFIDENCE SCORE FOR A MEDIA URL
func calculateMediaConfidence(url, mimeType string) float64 {
	confidence := 0.5 // DEFAULT CONFIDENCE

	// BOOST FOR EXPLICIT MEDIA MIME TYPES
	if strings.HasPrefix(mimeType, "video/") ||
		strings.HasPrefix(mimeType, "audio/") ||
		mimeType == "application/x-mpegURL" ||
		mimeType == "application/dash+xml" {
		confidence += 0.3
	}

	// BOOST FOR EXPLICIT MEDIA FILE EXTENSIONS
	urlLower := strings.ToLower(url)
	if strings.HasSuffix(urlLower, ".mp4") ||
		strings.HasSuffix(urlLower, ".webm") ||
		strings.HasSuffix(urlLower, ".m3u8") ||
		strings.HasSuffix(urlLower, ".mpd") ||
		strings.HasSuffix(urlLower, ".mp3") {
		confidence += 0.2
	}

	// BOOST FOR MEDIA-RELATED TERMS IN URL
	mediaTerms := []string{"video", "media", "stream", "play", "movie", "episode", "watch"}
	for _, term := range mediaTerms {
		if strings.Contains(urlLower, term) {
			confidence += 0.05
		}
	}

	// CAP CONFIDENCE AT 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// DEDUPLICATE MEDIASOURCES REMOVES DUPLICATE MEDIA SOURCES
func deduplicateMediaSources(sources []MediaSource) []MediaSource {
	if len(sources) <= 1 {
		return sources
	}

	// CREATE MAP TO TRACK UNIQUE URLS
	urlMap := make(map[string]bool)
	var uniqueSources []MediaSource

	for _, source := range sources {
		// NORMALIZE URL FOR COMPARISON
		normalizedURL := normalizeMediaURL(source.URL)

		if !urlMap[normalizedURL] {
			urlMap[normalizedURL] = true
			uniqueSources = append(uniqueSources, source)
		}
	}

	return uniqueSources
}

// NORMALIZEMEDIAURL NORMALIZES A MEDIA URL FOR DEDUPLICATION
func normalizeMediaURL(urlStr string) string {
	// PARSE URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	// REMOVE QUERY PARAMETERS THAT DON'T AFFECT THE CONTENT
	q := u.Query()
	for key := range q {
		if strings.Contains(key, "token") ||
			strings.Contains(key, "signature") ||
			strings.Contains(key, "auth") ||
			strings.Contains(key, "key") ||
			strings.Contains(key, "time") {
			q.Del(key)
		}
	}

	u.RawQuery = q.Encode()

	return u.String()
}

// RANKMEDISOURCES SORTS MEDIA SOURCES BY CONFIDENCE AND QUALITY
func rankMediaSources(sources []MediaSource) {
	sort.Slice(sources, func(i, j int) bool {
		// SORT BY CONFIDENCE FIRST
		if sources[i].Confidence != sources[j].Confidence {
			return sources[i].Confidence > sources[j].Confidence
		}

		// THEN BY FILE TYPE PRIORITY
		iScore := getFileTypePriority(sources[i].URL)
		jScore := getFileTypePriority(sources[j].URL)
		if iScore != jScore {
			return iScore > jScore
		}

		// THEN BY RESOLUTION/QUALITY IF AVAILABLE
		if sources[i].Resolution != "" && sources[j].Resolution != "" {
			iRes := parseResolution(sources[i].Resolution)
			jRes := parseResolution(sources[j].Resolution)
			if iRes != jRes {
				return iRes > jRes
			}
		}

		// FALLBACK TO ALPHABETICAL
		return sources[i].URL < sources[j].URL
	})
}

// GETFILETYPEPRIORITY RETURNS A PRIORITY SCORE FOR A MEDIA FILE TYPE
func getFileTypePriority(url string) int {
	urlLower := strings.ToLower(url)

	// PRIORITIZE MEDIA FORMATS
	if strings.HasSuffix(urlLower, ".mp4") {
		return 100
	}
	if strings.HasSuffix(urlLower, ".webm") {
		return 90
	}
	if strings.HasSuffix(urlLower, ".m3u8") {
		return 80
	}
	if strings.HasSuffix(urlLower, ".mpd") {
		return 70
	}
	if strings.HasSuffix(urlLower, ".ts") {
		return 60
	}
	if strings.HasSuffix(urlLower, ".mp3") {
		return 50
	}

	// DEFAULT PRIORITY
	return 0
}

// PARSERESOLUTION CONVERTS A RESOLUTION STRING TO A NUMERIC VALUE
func parseResolution(resolution string) int {
	// TRY TO PARSE WIDTHxHEIGHT FORMAT
	parts := strings.Split(resolution, "x")
	if len(parts) == 2 {
		width, err1 := strconv.Atoi(parts[0])
		height, err2 := strconv.Atoi(parts[1])

		if err1 == nil && err2 == nil {
			return width * height
		}
	}

	// TRY TO PARSE COMMON QUALITY LABELS
	switch strings.ToLower(resolution) {
	case "high":
		return 1000000
	case "medium":
		return 500000
	case "low":
		return 100000
	}

	return 0
}
