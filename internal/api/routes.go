package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/ui"
)

var MimeTypes = map[string]string{
	".css":   "text/css",
	".js":    "application/javascript",
	".mjs":   "application/javascript",
	".json":  "application/json",
	".html":  "text/html",
	".ico":   "image/x-icon",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".eot":   "application/vnd.ms-fontobject",
}

func SetupRouter() *gin.Engine {
	// SETUP ROUTER - DISABLE AUTOMATIC REDIRECTS
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// DISABLE AUTOMATIC TRAILING SLASH REDIRECTS
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// STATIC FILE ROUTES
	r.Static("/assets", config.AppConfig.StoragePath)
	r.Static("/thumbnails", config.AppConfig.ThumbnailsPath)
	r.Static("/images", "./static/images")

	// API ROUTES - MUST BE DEFINED BEFORE SPA HANDLER
	api := r.Group("/api")
	{
		// JOB ROUTES
		api.POST("/jobs", CreateJob)
		api.GET("/jobs", ListJobs)
		api.GET("/jobs/:id", GetJob)
		api.PUT("/jobs/:id", UpdateJob)
		api.DELETE("/jobs/:id", DeleteJob)
		api.POST("/jobs/:id/start", StartJob)
		api.POST("/jobs/:id/stop", StopJob)
		api.GET("/jobs/:id/assets", GetJobAssets)
		api.GET("/jobs/:id/statistics", GetJobStatistics)

		// ASSET ROUTES
		api.GET("/assets", GetAllAssets)
		api.GET("/assets/:id", GetAsset)
		api.DELETE("/assets/:id", DeleteAsset)
		api.POST("/assets/:id/regenerate-thumbnail", RegenerateThumbnail)

		// TEMPLATE ROUTES
		api.GET("/templates", ListTemplates)
		api.POST("/templates", CreateTemplate)
		api.GET("/templates/:id", GetTemplate)
		api.PUT("/templates/:id", UpdateTemplate)
		api.DELETE("/templates/:id", DeleteTemplate)
		api.POST("/templates/:id/create-job", CreateJobFromTemplate)

		// SETTINGS ROUTES
		api.GET("/settings", GetSettings)
		api.PUT("/settings", UpdateSettings)
		api.GET("/storage/info", GetStorageInfo)
		api.POST("/cache/clear", ClearCache)

		// UI PROXY + VISUAL SELECTOR
		api.GET("/proxy", ProxyHandler)
	}

	// SERVE THE EMBEDDED SVELTE UI - ROOT PATH
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		fileServer := http.FileServer(ui.GetFileSystem())
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// SERVE THE EMBEDDED SVELTE UI - SPECIFIC APP ENTRY POINTS AND STATIC FILES
	// This explicitly handles the Svelte app build files
	r.GET("/_app/*filepath", func(c *gin.Context) {
		path := c.Param("filepath")

		// Set the correct MIME type based on file extension
		ext := filepath.Ext(path)
		if mimeType, ok := MimeTypes[ext]; ok {
			c.Header("Content-Type", mimeType)
		}

		fileServer := http.FileServer(ui.GetFileSystem())
		http.StripPrefix("", fileServer).ServeHTTP(c.Writer, c.Request)
	})

	// SPA ROUTES HANDLER - MUST BE LAST
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// CHECK IF THE REQUEST IS FOR A STATIC ASSET
		ext := filepath.Ext(path)
		if ext != "" {
			// THIS IS LIKELY A STATIC FILE REQUEST
			filePath := path
			if path[0] == '/' {
				filePath = path[1:] // REMOVE LEADING SLASH FOR EMBEDDED FS
			}

			// SET THE CORRECT MIME TYPE
			if mimeType, ok := MimeTypes[ext]; ok {
				c.Header("Content-Type", mimeType)
			}

			// ATTEMPT TO SERVE THE FILE
			file, err := ui.GetFileSystem().Open(filePath)
			if err == nil {
				// FILE EXISTS, SERVE IT
				file.Close()
				fileServer := http.FileServer(ui.GetFileSystem())
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// CHECK IF THIS IS AN API REQUEST
		if strings.HasPrefix(path, "/api/") {
			c.Status(http.StatusNotFound) // LET ACTUAL API 404s REMAIN AS 404s
			return
		}

		// FOR ALL OTHER ROUTES (SPA ROUTES), SERVE THE INDEX.HTML
		c.Header("Content-Type", "text/html")
		// REWRITE THE REQUEST TO THE ROOT PATH
		c.Request.URL.Path = "/"
		fileServer := http.FileServer(ui.GetFileSystem())
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return r
}
