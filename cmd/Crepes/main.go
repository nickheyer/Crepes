package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/nickheyer/Crepes/internal/api"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/scheduler"
	"github.com/nickheyer/Crepes/internal/storage"
	"github.com/nickheyer/Crepes/internal/templates"
)

func main() {
	// INITIALIZE CONFIG
	config.InitConfig()

	// SETUP LOG FILE
	setupLogging()

	// CREATE NECESSARY FILES AND DIRECTORIES
	if err := templates.CreateTemplates(); err != nil {
		log.Printf("Error creating templates: %v", err)
	}

	if err := templates.CreateStaticFiles(); err != nil {
		log.Printf("Error creating static files: %v", err)
	}

	// INITIALIZE SCHEDULER
	scheduler.InitScheduler()

	// LOAD EXISTING JOBS
	storage.LoadJobs()

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
