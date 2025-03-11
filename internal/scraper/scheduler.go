package scraper

import (
	"log"
	"sync"

	"github.com/nickheyer/Crepes/internal/models"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// JOB SCHEDULER
type Scheduler struct {
	db     *gorm.DB
	engine *Engine
	cron   *cron.Cron
	jobs   map[string]cron.EntryID
	mu     sync.Mutex
}

// CREATE NEW SCHEDULER
func NewScheduler(db *gorm.DB, engine *Engine) *Scheduler {
	return &Scheduler{
		db:     db,
		engine: engine,
		cron:   cron.New(),
		jobs:   make(map[string]cron.EntryID),
		mu:     sync.Mutex{},
	}
}

// START THE SCHEDULER
func (s *Scheduler) Start() {
	// START CRON SCHEDULER
	s.cron.Start()

	// LOAD ALL SCHEDULED JOBS FROM DATABASE
	var jobs []models.Job
	s.db.Where("schedule != ''").Find(&jobs)

	// SCHEDULE EACH JOB
	for _, job := range jobs {
		s.ScheduleJob(&job)
	}

	log.Printf("Job scheduler started with %d scheduled jobs", len(jobs))
}

// STOP THE SCHEDULER
func (s *Scheduler) Stop() {
	// STOP CRON SCHEDULER
	ctx := s.cron.Stop()
	<-ctx.Done() // WAIT FOR JOBS TO FINISH

	log.Println("Job scheduler stopped")
}

// SCHEDULE A JOB
func (s *Scheduler) ScheduleJob(job *models.Job) {
	if job.Schedule == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// REMOVE EXISTING SCHEDULE IF ANY
	if entryID, exists := s.jobs[job.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, job.ID)
	}

	// CREATE CRON JOB
	entryID, err := s.cron.AddFunc(job.Schedule, func() {
		log.Printf("Running scheduled job: %s", job.ID)
		err := s.engine.RunJob(job.ID)
		if err != nil {
			log.Printf("Failed to run scheduled job %s: %v", job.ID, err)
		}
	})

	if err != nil {
		log.Printf("Failed to schedule job %s: %v", job.ID, err)
		return
	}

	// STORE ENTRY ID
	s.jobs[job.ID] = entryID

	// UPDATE NEXT RUN TIME
	entry := s.cron.Entry(entryID)
	if !entry.Next.IsZero() {
		job.NextRun = entry.Next
		s.db.Model(job).Update("next_run", job.NextRun)
	}

	log.Printf("Job %s scheduled with cron: %s, next run: %v", job.ID, job.Schedule, job.NextRun)
}

// REMOVE A JOB FROM THE SCHEDULER
func (s *Scheduler) RemoveJob(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[jobID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, jobID)
		log.Printf("Job %s removed from scheduler", jobID)
	}
}

// UPDATE NEXT RUN TIMES FOR ALL JOBS
func (s *Scheduler) UpdateNextRunTimes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for jobID, entryID := range s.jobs {
		entry := s.cron.Entry(entryID)
		if !entry.Next.IsZero() {
			s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("next_run", entry.Next)
		}
	}
}
