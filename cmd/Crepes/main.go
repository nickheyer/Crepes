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

	"github.com/nickheyer/Crepes/internal/api"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/database"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/scraper"
)

const VERSION = "v0.1.0"

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file")
	port := flag.String("port", "", "HTTP port to listen on (overrides config)")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("WARNING: Failed to load config file: %v, using default settings", err)
		cfg = config.GetDefaultConfig()
	}

	if *port != "" {
		cfg.Port = *port
	}

	createDirs(cfg)

	db, err := database.SetupDatabase(cfg.DataPath)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get db from GORM: %v", err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(&models.Job{}, &models.Asset{}, &models.Setting{}); err != nil {
		log.Fatalf("Failed to migrate database schemas: %v", err)
	}

	database.EnsureDefaultSettings(db)

	scraperEngine := scraper.NewEngine(db, cfg)

	jobScheduler := scraper.NewScheduler(db, scraperEngine)
	jobScheduler.Start()
	defer jobScheduler.Stop()

	routerConfig := api.RouterConfig{
		DB:            db,
		Config:        cfg,
		ScraperEngine: scraperEngine,
		JobScheduler:  jobScheduler,
	}
	router := api.SetupRouter(routerConfig)

	addr := ":" + cfg.Port
	srv := &http.Server{
		Handler:      router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Crepes %s starting on http://localhost%s", VERSION, addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

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
		absPath, err := filepath.Abs(dir)
		if err != nil {
			log.Printf("WARNING: Failed to get absolute path for %s: %v", dir, err)
		} else {
			log.Printf("Using directory: %s", absPath)
		}
	}
}
