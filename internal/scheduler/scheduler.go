package scheduler

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/scraper"
)

var (
	Scheduler *gocron.Scheduler
)

// INITSCHEDULER INITIALIZES THE JOB SCHEDULER
func InitScheduler() {
	Scheduler = gocron.NewScheduler(time.UTC)
	Scheduler.StartAsync()
}

// SCHEDULEJOB SCHEDULES A JOB WITH THE GIVEN CRON EXPRESSION
func ScheduleJob(job *models.ScrapingJob) {
	_, err := Scheduler.Cron(job.Schedule).Do(func() {
		// CHECK IF JOB IS ALREADY RUNNING
		job.Mutex.Lock()
		if job.Status == "running" {
			job.Mutex.Unlock()
			return
		}
		job.Mutex.Unlock()

		go scraper.RunJob(job)
	})

	if err != nil {
		log.Printf("Error scheduling job %s: %v", job.ID, err)
	}
}

// STOPSCHEDULER STOPS THE SCHEDULER
func StopScheduler() {
	if Scheduler != nil {
		Scheduler.Stop()
	}
}
