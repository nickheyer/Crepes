package scraper

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

func FetchWithHTTP(ctx context.Context, url, userAgent string) (string, error) {
	// CREATE TRANSPORT WITH RELAXED SECURITY AND TIMEOUTS
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // IGNORE CERT ERRORS
		},
		DisableCompression:    false,
		MaxIdleConns:          100,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
	}

	// CREATE COOKIE JAR
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// CREATE CLIENT - NO TIMEOUT HERE, WE USE CONTEXT
	client := &http.Client{
		Transport: transport,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// ALLOW UP TO 10 REDIRECTS
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			// COPY HEADERS ON REDIRECT
			for key, val := range via[0].Header {
				if _, ok := req.Header[key]; !ok {
					req.Header[key] = val
				}
			}
			return nil
		},
	}

	// CREATE REQUEST
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
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

	// SEND REQUEST WITH RETRIES
	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = client.Do(req)

		// SUCCESS CASE
		if err == nil && resp.StatusCode < 500 {
			break
		}

		// HANDLE ERROR
		if resp != nil {
			resp.Body.Close()
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("server returned status: %d", resp.StatusCode)
		}

		// CHECK CONTEXT BEFORE RETRY
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 2 * time.Second):
			// CONTINUE AFTER BACKOFF
		}

		log.Printf("Retrying HTTP fetch for %s (attempt %d/%d): %v", url, attempt+1, maxRetries, lastErr)
	}

	if resp == nil {
		return "", fmt.Errorf("HTTP fetch failed after %d attempts: %v", maxRetries, lastErr)
	}
	defer resp.Body.Close()

	// CHECK RESPONSE STATUS
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("server returned status code %d", resp.StatusCode)
	}

	// CHECK CONTENT TYPE FOR HTML
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "html") &&
		!strings.Contains(strings.ToLower(contentType), "text") &&
		contentType != "" {
		log.Printf("Warning: URL %s returned non-HTML content type: %s", url, contentType)
	}

	// READ RESPONSE BODY WITH GZIP HANDLING
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error creating gzip reader: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// READ WITH SIZE LIMIT
	body, err := io.ReadAll(io.LimitReader(reader, 10*1024*1024)) // 10MB LIMIT
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	return string(body), nil
}

func TestSiteAccessibility(ctx context.Context, url string) error {
	// FIRST TEST WITH A SIMPLE HTTP REQUEST
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	// CREATE A REQUEST WITH COMMON BROWSER HEADERS
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

	// CHECK RESPONSE STATUS
	if resp.StatusCode >= 400 {
		return fmt.Errorf("site returned error status: %d %s", resp.StatusCode, resp.Status)
	}

	// READ A BIT OF THE BODY TO VERIFY CONTENT
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	// CHECK IF IT'S LIKELY A BOT PROTECTION PAGE
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
