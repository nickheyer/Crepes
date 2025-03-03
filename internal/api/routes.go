package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nickheyer/Crepes/internal/config"
)

// SETUPROUTER CONFIGURES THE APPLICATION ROUTER WITH ALL ROUTES
func SetupRouter() *gin.Engine {
	// SETUP ROUTER
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// STATIC FILE ROUTES
	r.Static("/assets", config.AppConfig.StoragePath)
	r.Static("/thumbnails", config.AppConfig.ThumbnailsPath)
	r.Static("/static", "./web/static")
	r.LoadHTMLGlob("web/templates/*")

	// API ROUTES
	api := r.Group("/api")
	{
		api.POST("/jobs", CreateJob)
		api.GET("/jobs", ListJobs)
		api.GET("/jobs/:id", GetJob)
		api.DELETE("/jobs/:id", DeleteJob)
		api.POST("/jobs/:id/start", StartJob)
		api.POST("/jobs/:id/stop", StopJob)
		api.POST("/jobs/:id/next-page", NextJobPage)
		api.GET("/jobs/:id/assets", GetJobAssets)
		api.GET("/assets", GetAllAssets)
		api.GET("/assets/:id", GetAsset)
		api.DELETE("/assets/:id", DeleteAsset)
		api.POST("/assets/:id/regenerate-thumbnail", RegenerateThumbnail)
	}

	// WEB INTERFACE ROUTES
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Web Scraper",
		})
	})

	r.GET("/jobs/:id", func(c *gin.Context) {
		jobID := c.Param("id")
		c.HTML(http.StatusOK, "job.html", gin.H{
			"jobID": jobID,
		})
	})

	r.GET("/gallery", func(c *gin.Context) {
		c.HTML(http.StatusOK, "gallery.html", gin.H{
			"title": "Asset Gallery",
		})
	})

	return r
}
