package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/database"
	"github.com/nickheyer/Crepes/internal/handlers"
	"github.com/nickheyer/Crepes/internal/middleware"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/scraper"

	"github.com/nickheyer/Crepes/internal/ui"

	"github.com/gorilla/mux"
)

// VERSION INFO
const VERSION = "v0.1.0"

func main() {
	// COMMAND LINE FLAGS
	configPath := flag.String("config", "config.json", "Path to configuration file")
	port := flag.String("port", "", "HTTP port to listen on (overrides config)")
	flag.Parse()

	// LOAD CONFIG
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("WARNING: Failed to load config file: %v, using default settings", err)
		cfg = config.GetDefaultConfig()
	}

	// OVERRIDE PORT IF SPECIFIED
	if *port != "" {
		cfg.Port = *port
	}

	// CREATE REQUIRED DIRECTORIES
	createDirs(cfg)

	// SETUP DATABASE
	db, err := database.SetupDatabase(cfg.DataPath)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}

	// DEFER CLOSE OF UNDERLYING SQLDB
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get db from GORM: %v", err)
	}
	defer sqlDB.Close()

	// AUTO MIGRATE SCHEMAS
	if err := db.AutoMigrate(&models.Job{}, &models.Asset{}, &models.Template{}, &models.Setting{}); err != nil {
		log.Fatalf("Failed to migrate database schemas: %v", err)
	}

	// SET DEFAULT SETTINGS IF NEEDED
	database.EnsureDefaultSettings(db)

	// SETUP SCRAPER ENGINE
	scraperEngine := scraper.NewEngine(db, cfg)

	// SETUP SCHEDULED JOBS
	jobScheduler := scraper.NewScheduler(db, scraperEngine)
	jobScheduler.Start()
	defer jobScheduler.Stop()

	// SETUP ROUTER
	router := mux.NewRouter()

	// MIDDLEWARE
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.CORSMiddleware)

	// API ROUTES
	apiRouter := router.PathPrefix("/api").Subrouter()
	handlers.RegisterJobHandlers(apiRouter, db, scraperEngine, jobScheduler)
	handlers.RegisterAssetHandlers(apiRouter, db, cfg)
	handlers.RegisterTemplateHandlers(apiRouter, db)
	handlers.RegisterSettingsHandlers(apiRouter, db, cfg)
	handlers.RegisterStorageHandlers(apiRouter, cfg)
	handlers.RegisterProxyHandler(apiRouter)

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

	// CREATE SERVER
	addr := ":" + cfg.Port
	srv := &http.Server{
		Handler:      router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// RUN SERVER IN GOROUTINE
	go func() {
		log.Printf("Crepes %s starting on http://localhost%s", VERSION, addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// GRACEFUL SHUTDOWN
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// CREATE CONTEXT WITH TIMEOUT
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// SHUTDOWN SERVER
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

// CREATE REQUIRED DIRECTORIES
func createDirs(cfg *config.Config) {
	dirs := []string{
		cfg.StoragePath,
		cfg.ThumbnailsPath,
		cfg.DataPath,
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Printf("WARNING: Failed to create directory: %s, %v", dir, err)
			}
		}
		// MAKE PATH ABSOLUTE
		absPath, err := filepath.Abs(dir)
		if err != nil {
			log.Printf("WARNING: Failed to get absolute path for %s: %v", dir, err)
		} else {
			log.Printf("Using directory: %s", absPath)
		}
	}
}
