package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nickheyer/Crepes/internal/api"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/scheduler"
	"github.com/nickheyer/Crepes/internal/storage"
)

func main() {
	// INITIALIZE CONFIG
	config.InitConfig()

	// SETUP LOG FILE
	setupLogging()

	// CREATE NECESSARY FILES AND DIRECTORIES
	if err := createRequiredDirectories(); err != nil {
		log.Fatalf("Error creating required directories: %v", err)
	}

	// INITIALIZE DATABASE
	dbPath := filepath.Join(config.AppConfig.DataPath, "crepes.db")
	if err := storage.InitDB(dbPath); err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer storage.CloseDB()

	// START PERIODIC SAVE
	storage.StartPeriodicSave(5 * time.Minute)

	// INITIALIZE SCHEDULER
	scheduler.InitScheduler()

	// SETUP ROUTER AND START SERVER
	r := api.SetupRouter()

	// START SERVER
	port := config.AppConfig.Port
	log.Printf("Server starting on port %d", port)
	r.Run(fmt.Sprintf(":%d", port))
}

func setupLogging() {
	// SETUP LOG FILE
	logFile, err := os.OpenFile(config.AppConfig.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func createRequiredDirectories() error {
	dirs := []string{
		config.AppConfig.StoragePath,
		config.AppConfig.ThumbnailsPath,
		config.AppConfig.DataPath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return nil
}
