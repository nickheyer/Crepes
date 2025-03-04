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
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".html": "text/html",
	".ico":  "image/x-icon",
	".png":  "image/png",
	".svg":  "image/svg+xml",
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

	// API ROUTES - MUST BE DEFINED BEFORE SPA HANDLER
	api := r.Group("/api")
	{
		// JOB ROUTES
		api.POST("/jobs", CreateJob)
		api.GET("/jobs", ListJobs)
		api.GET("/jobs/:id", GetJob)
		api.DELETE("/jobs/:id", DeleteJob)
		api.POST("/jobs/:id/start", StartJob)
		api.POST("/jobs/:id/stop", StopJob)
		api.GET("/jobs/:id/assets", GetJobAssets)

		// ASSET ROUTES
		api.GET("/assets", GetAllAssets)
		api.GET("/assets/:id", GetAsset)
		api.DELETE("/assets/:id", DeleteAsset)
		api.POST("/assets/:id/regenerate-thumbnail", RegenerateThumbnail)
	}

	// SERVE STATIC FILES FROM THE EMBEDDED FILESYSTEM
	fileServer := http.FileServer(ui.GetFileSystem())

	// SERVE THE EMBEDDED SVELTE UI - ROOT PATH
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", MimeTypes[".html"])
		fileServer.ServeHTTP(c.Writer, c.Request)
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

			// ATTEMPT TO SERVE THE FILE
			file, err := ui.GetFileSystem().Open(filePath)
			if err == nil {
				// FILE EXISTS, SERVE IT
				file.Close()
				c.Header("Content-Type", MimeTypes[ext])
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
		c.Header("Content-Type", MimeTypes[".html"])

		// REWRITE THE REQUEST TO THE ROOT PATH
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return r
}
