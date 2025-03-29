package handlers

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nickheyer/Crepes/internal/utils"
)

func ProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURLStr := r.URL.Query().Get("url")
		if targetURLStr == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "URL parameter is required")
			return
		}
		targetURL, err := url.Parse(targetURLStr)
		if err != nil || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid URL provided")
			return
		}
		client := &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}
		proxyReq, err := http.NewRequest(http.MethodGet, targetURLStr, nil)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create request")
			return
		}
		proxyReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		proxyReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		proxyReq.Header.Set("Accept-Language", "en-US,en;q=0.5")
		proxyReq.Header.Set("Connection", "keep-alive")
		proxyReq.Header.Set("Upgrade-Insecure-Requests", "1")
		resp, err := client.Do(proxyReq)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadGateway, "Failed to fetch URL: "+err.Error())
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to read response body")
			return
		}
		for key, values := range resp.Header {
			if strings.ToLower(key) == "content-encoding" ||
				strings.ToLower(key) == "content-length" ||
				strings.ToLower(key) == "transfer-encoding" ||
				strings.ToLower(key) == "connection" {
				continue
			}
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		if strings.Contains(w.Header().Get("Content-Type"), "text/html") {
			bodyStr := string(body)
			injectScript := `<script>
				window.addEventListener('load', function() {
					if (window.parent) {
						window.parent.postMessage({type: 'IFRAME_LOADED'}, '*');
					}
				});
			</script>`
			if strings.Contains(bodyStr, "</head>") {
				bodyStr = strings.Replace(bodyStr, "</head>", injectScript+"</head>", 1)
			} else if strings.Contains(bodyStr, "</body>") {
				bodyStr = strings.Replace(bodyStr, "</body>", injectScript+"</body>", 1)
			} else {
				bodyStr = bodyStr + injectScript
			}
			body = []byte(bodyStr)
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	}
}
