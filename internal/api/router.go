package api

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/handlers"
	"github.com/nickheyer/Crepes/internal/middleware"
	"github.com/nickheyer/Crepes/internal/scraper"
	"github.com/nickheyer/Crepes/internal/ui"
	"gorm.io/gorm"
)

type RouterConfig struct {
	DB            *gorm.DB
	Config        *config.Config
	ScraperEngine *scraper.Engine
	JobScheduler  *scraper.Scheduler
}

func SetupRouter(cfg RouterConfig) *mux.Router {
	// SETUP MAIN ROUTER
	router := mux.NewRouter()

	// MIDDLEWARE
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.CORSMiddleware)

	// API ROUTES
	apiRouter := router.PathPrefix("/api").Subrouter()

	// SETUP ALL API ROUTES
	setupJobRoutes(apiRouter, cfg.DB, cfg.ScraperEngine, cfg.JobScheduler)
	setupAssetRoutes(apiRouter, cfg.DB, cfg.Config)
	setupSettingsRoutes(apiRouter, cfg.DB, cfg.Config)
	setupStorageRoutes(apiRouter, cfg.Config)
	setupProxyRoutes(apiRouter)

	// UI ROUTES
	fileServer := http.FileServer(ui.GetFileSystem())
	router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CHECK IF PATH EXISTS IN STATIC ASSETS
		if _, err := ui.GetFileSystem().Open(r.URL.Path); os.IsNotExist(err) {
			// SERVE INDEX.HTML FOR ALL NON-ASSET ROUTES (SPA)
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}))

	return router
}

// JOBS ROUTES
func setupJobRoutes(router *mux.Router, db *gorm.DB, engine *scraper.Engine, scheduler *scraper.Scheduler) {
	// GET ALL JOBS
	router.HandleFunc("/jobs", handlers.GetAllJobs(db)).Methods("GET")

	// GET JOB BY ID
	router.HandleFunc("/jobs/{id}", handlers.GetJobByID(db)).Methods("GET")

	// CREATE JOB
	router.HandleFunc("/jobs", handlers.CreateJob(db, scheduler)).Methods("POST")

	// UPDATE JOB
	router.HandleFunc("/jobs/{id}", handlers.UpdateJob(db, scheduler)).Methods("PUT")

	// DELETE JOB
	router.HandleFunc("/jobs/{id}", handlers.DeleteJob(db, engine, scheduler)).Methods("DELETE")

	// START JOB
	router.HandleFunc("/jobs/{id}/start", handlers.StartJob(db, engine)).Methods("POST")

	// STOP JOB
	router.HandleFunc("/jobs/{id}/stop", handlers.StopJob(db, engine)).Methods("POST")

	// GET JOB ASSETS
	router.HandleFunc("/jobs/{id}/assets", handlers.GetJobAssets(db)).Methods("GET")

	// GET JOB STATISTICS
	router.HandleFunc("/jobs/{id}/statistics", handlers.GetJobStatistics(db, engine)).Methods("GET")
}

// ASSETS ROUTES
func setupAssetRoutes(router *mux.Router, db *gorm.DB, cfg *config.Config) {
	// GET ALL ASSETS WITH OPTIONAL FILTERS
	router.HandleFunc("/assets", handlers.GetAllAssets(db)).Methods("GET")

	// GET ASSET BY ID
	router.HandleFunc("/assets/{id}", handlers.GetAssetByID(db)).Methods("GET")

	// DELETE ASSET
	router.HandleFunc("/assets/{id}", handlers.DeleteAsset(db, cfg)).Methods("DELETE")

	// REGENERATE THUMBNAIL
	router.HandleFunc("/assets/{id}/regenerate-thumbnail", handlers.RegenerateThumbnail(db, cfg)).Methods("POST")

	// GET ASSET COUNTS BY TYPE
	router.HandleFunc("/assets/counts", handlers.GetAssetCounts(db)).Methods("GET")

	// SERVE ASSET FILES
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/api/assets/", http.FileServer(http.Dir(cfg.StoragePath))))

	// SERVE THUMBNAIL FILES
	router.PathPrefix("/thumbnails/").Handler(http.StripPrefix("/api/thumbnails/", http.FileServer(http.Dir(cfg.ThumbnailsPath))))
}

// SETTINGS ROUTES
func setupSettingsRoutes(router *mux.Router, db *gorm.DB, cfg *config.Config) {
	// GET ALL SETTINGS
	router.HandleFunc("/settings", handlers.GetSettings(db, cfg)).Methods("GET")

	// UPDATE SETTINGS
	router.HandleFunc("/settings", handlers.UpdateSettings(db, cfg)).Methods("PUT")

	// CLEAR CACHE
	router.HandleFunc("/cache/clear", handlers.ClearCache()).Methods("POST")
}

// STORAGE ROUTES
func setupStorageRoutes(router *mux.Router, cfg *config.Config) {
	// GET STORAGE INFO
	router.HandleFunc("/storage/info", handlers.GetStorageInfo(cfg)).Methods("GET")
}

// PROXY ROUTES
func setupProxyRoutes(router *mux.Router) {
	// PROXY HANDLER FOR FRONTEND VISUAL SELECTOR
	router.HandleFunc("/proxy", handlers.ProxyHandler()).Methods("GET")
}
