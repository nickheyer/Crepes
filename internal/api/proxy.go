package api

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// FORWARD PARSED AND INJECTED WEBPAGE TO BROWSER IFRAME
func ProxyHandler(c *gin.Context) {
	// GET URL FROM QUERY PARAM
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL parameter is required"})
		return
	}

	// CREATE HTTP CLIENT WITH TIMEOUT
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// CREATE REQUEST
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating request: " + err.Error()})
		return
	}

	// SET COMMON HEADERS TO AVOID BEING BLOCKED
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// EXECUTE REQUEST
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching URL: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// CHECK STATUS
	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Remote server returned error: " + resp.Status})
		return
	}

	// COPY RELEVANT HEADERS
	for key, values := range resp.Header {
		// SKIP CONTENT-LENGTH AS WE MIGHT MODIFY THE CONTENT
		if strings.ToLower(key) != "content-length" &&
			strings.ToLower(key) != "content-security-policy" &&
			strings.ToLower(key) != "x-frame-options" {
			for _, value := range values {
				c.Header(key, value)
			}
		}
	}

	// ENSURE CONTENT TYPE IS SET
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/html; charset=utf-8"
	}
	c.Header("Content-Type", contentType)

	// ADD HEADERS TO ALLOW IFRAME
	c.Header("X-Frame-Options", "SAMEORIGIN")
	c.Header("Content-Security-Policy", "frame-ancestors 'self'")

	// READ AND MODIFY CONTENT IF HTML
	if strings.Contains(contentType, "text/html") {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response body: " + err.Error()})
			return
		}

		// MODIFY HTML CONTENT TO MAKE IT WORK IN IFRAME
		htmlContent := string(body)

		// 1. INJECT BASE TAG FOR RELATIVE LINKS
		if !strings.Contains(htmlContent, "<base") {
			htmlContent = strings.Replace(
				htmlContent,
				"<head>",
				"<head>\n<base href=\""+targetURL+"\">",
				1,
			)

			// IF NO HEAD TAG, ADD ONE
			if !strings.Contains(htmlContent, "<head>") {
				htmlContent = strings.Replace(
					htmlContent,
					"<html>",
					"<html>\n<head>\n<base href=\""+targetURL+"\">\n</head>",
					1,
				)
			}
		}

		// 2. REMOVE ALL JAVASCRIPT EVENT HANDLERS
		// This is a simple approach - a more comprehensive solution would use HTML parsing
		htmlContent = strings.Replace(htmlContent, " onclick=", " data-onclick=", -1)
		htmlContent = strings.Replace(htmlContent, " onmouseover=", " data-onmouseover=", -1)
		htmlContent = strings.Replace(htmlContent, " onmouseout=", " data-onmouseout=", -1)
		htmlContent = strings.Replace(htmlContent, " onmouseenter=", " data-onmouseenter=", -1)
		htmlContent = strings.Replace(htmlContent, " onmouseleave=", " data-onmouseleave=", -1)

		// 3. PREVENT LAYOUT SHIFTING CSS
		styleTag := `
		<style>
		html, body {
			height: auto !important;
			position: relative !important;
			overflow: visible !important;
		}
		* {
			transition: none !important;
			animation: none !important;
		}
		</style>
		`

		htmlContent = strings.Replace(
			htmlContent,
			"</head>",
			styleTag+"</head>",
			1,
		)

		// 4. DISABLE JAVASCRIPT THAT MIGHT INTERFERE
		disableJsScript := `
		<script>
		// Prevent default click behaviors
		document.addEventListener('DOMContentLoaded', function() {
			// Disable all existing script tags
			var scripts = document.getElementsByTagName('script');
			for (var i = scripts.length - 1; i >= 0; i--) {
				if (scripts[i].src || scripts[i].textContent.length > 0) {
					scripts[i].parentNode.removeChild(scripts[i]);
				}
			}
			
			// Disable all event handlers
			document.addEventListener('click', function(e) {
				e.stopPropagation();
				e.preventDefault();
				return false;
			}, true);
		});
		</script>
		`

		// ADD AT THE END OF BODY
		if strings.Contains(htmlContent, "</body>") {
			htmlContent = strings.Replace(
				htmlContent,
				"</body>",
				disableJsScript+"</body>",
				1,
			)
		} else {
			// APPEND IF NO BODY CLOSING TAG
			htmlContent += disableJsScript
		}

		// WRITE MODIFIED CONTENT
		c.Data(http.StatusOK, contentType, []byte(htmlContent))
	} else {
		// FOR NON-HTML CONTENT, JUST COPY THE BODY
		c.DataFromReader(http.StatusOK, resp.ContentLength, contentType, resp.Body, nil)
	}
}
