package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/playwright-community/playwright-go"
	"gorm.io/gorm"
)

// ERROR DEFINITIONS
var (
	ErrPlaywrightNotInitialized = errors.New("PLAYWRIGHT NOT INITIALIZED")
	ErrJobAlreadyRunning        = errors.New("JOB IS ALREADY RUNNING")
	ErrJobNotFound              = errors.New("JOB NOT FOUND")
	ErrBrowserCreation          = errors.New("FAILED TO CREATE BROWSER")
	ErrPageCreation             = errors.New("FAILED TO CREATE PAGE")
	ErrTaskNotFound             = errors.New("TASK TYPE NOT FOUND")
	ErrResourceNotFound         = errors.New("RESOURCE NOT FOUND")
	ErrInvalidInput             = errors.New("INVALID TASK INPUT")
)

// ENGINE CORE STRUCT
type Engine struct {
	db              *gorm.DB
	cfg             *config.Config
	runningJobs     map[string]context.CancelFunc
	jobProgress     map[string]JobProgress
	jobStartTimes   map[string]time.Time
	jobDurations    map[string]time.Duration
	mu              sync.Mutex
	playwright      *playwright.Playwright
	browserPool     chan browserInstance
	initialized     bool
	initMu          sync.Mutex
	taskRegistry    *TaskRegistry
	resourceManager *ResourceManager
}

// JOB PROGRESS TRACKING
type JobProgress struct {
	TotalTasks     int                 `json:"totalTasks"`
	CompletedTasks int                 `json:"completedTasks"`
	CurrentStage   string              `json:"currentStage"`
	StageProgress  map[string]int      `json:"stageProgress"`
	Status         string              `json:"status"`
	Errors         []string            `json:"errors"`
	Assets         int                 `json:"assets"`
	TaskResults    map[string]TaskData `json:"taskResults"` // Store task outputs for use as inputs to other tasks
}

// BROWSER INSTANCE
type browserInstance struct {
	browser *playwright.Browser
}

// RESOURCE MANAGER HANDLES JOB RESOURCES
type ResourceManager struct {
	mu        sync.Mutex
	resources map[string]map[string]interface{} // Job ID -> Resource ID -> Resource
}

// NEW RESOURCE MANAGER
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		resources: make(map[string]map[string]interface{}),
	}
}

// CREATE A RESOURCE
func (rm *ResourceManager) CreateResource(jobID, resourceID, resourceType string, value interface{}) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// INITIALIZE JOB RESOURCES MAP IF NOT EXISTS
	if _, ok := rm.resources[jobID]; !ok {
		rm.resources[jobID] = make(map[string]interface{})
	}

	// STORE THE RESOURCE
	rm.resources[jobID][resourceID] = value
}

// GET A RESOURCE
func (rm *ResourceManager) GetResource(jobID, resourceID string) (interface{}, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// CHECK IF JOB RESOURCES MAP EXISTS
	jobResources, ok := rm.resources[jobID]
	if !ok {
		return nil, false
	}

	// GET THE RESOURCE
	resource, ok := jobResources[resourceID]
	return resource, ok
}

// DELETE A RESOURCE
func (rm *ResourceManager) DeleteResource(jobID, resourceID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// CHECK IF JOB RESOURCES MAP EXISTS
	jobResources, ok := rm.resources[jobID]
	if !ok {
		return
	}

	// DELETE THE RESOURCE
	delete(jobResources, resourceID)
}

// DELETE ALL JOB RESOURCES
func (rm *ResourceManager) DeleteJobResources(jobID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// DELETE ALL RESOURCES FOR JOB
	delete(rm.resources, jobID)
}

// TASK DATA FOR INPUT/OUTPUT
type TaskData struct {
	Type  string      `json:"type"` // "string", "number", "boolean", "object", "array", "null"
	Value interface{} `json:"value"`
}

// TASK CONTEXT PASSED TO TASK IMPLEMENTATION
type TaskContext struct {
	JobID           string
	ResourceManager *ResourceManager
	TaskResults     map[string]TaskData
	Engine          *Engine
	Context         context.Context
	Logger          *log.Logger
}

// TASK IMPLEMENTATION INTERFACE
type TaskImplementation interface {
	Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error)
	ValidateConfig(config map[string]interface{}) error
	GetInputSchema() map[string]string
	GetOutputSchema() string
}

// TASK REGISTRY STORES AVAILABLE TASK IMPLEMENTATIONS
type TaskRegistry struct {
	mu              sync.RWMutex
	implementations map[string]TaskImplementation
}

// NEW TASK REGISTRY
func NewTaskRegistry() *TaskRegistry {
	return &TaskRegistry{
		implementations: make(map[string]TaskImplementation),
	}
}

// REGISTER A TASK IMPLEMENTATION
func (tr *TaskRegistry) RegisterTask(taskType string, implementation TaskImplementation) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.implementations[taskType] = implementation
}

// GET A TASK IMPLEMENTATION
func (tr *TaskRegistry) GetTask(taskType string) (TaskImplementation, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	implementation, ok := tr.implementations[taskType]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return implementation, nil
}

// LIST AVAILABLE TASK TYPES
func (tr *TaskRegistry) ListTaskTypes() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	types := make([]string, 0, len(tr.implementations))
	for taskType := range tr.implementations {
		types = append(types, taskType)
	}
	return types
}

// NEW ENGINE FACTORY
func NewEngine(db *gorm.DB, cfg *config.Config) *Engine {
	log.Printf("CREATING NEW SCRAPER ENGINE")

	// CREATE RESOURCE MANAGER
	resourceManager := NewResourceManager()

	// CREATE TASK REGISTRY
	taskRegistry := NewTaskRegistry()

	engine := &Engine{
		db:              db,
		cfg:             cfg,
		runningJobs:     make(map[string]context.CancelFunc),
		jobProgress:     make(map[string]JobProgress),
		jobStartTimes:   make(map[string]time.Time),
		jobDurations:    make(map[string]time.Duration),
		mu:              sync.Mutex{},
		browserPool:     make(chan browserInstance, cfg.MaxConcurrent),
		initialized:     false,
		initMu:          sync.Mutex{},
		taskRegistry:    taskRegistry,
		resourceManager: resourceManager,
	}

	// INIT PLAYWRIGHT
	log.Printf("INITIALIZING PLAYWRIGHT FOR ENGINE")
	err := engine.initPlaywright()
	if err != nil {
		log.Printf("ERROR INITIALIZING PLAYWRIGHT: %v", err)
		engine.initialized = false
	}

	// REGISTER TASK IMPLEMENTATIONS
	engine.registerTasks()

	return engine
}

// REGISTER ALL AVAILABLE TASK IMPLEMENTATIONS
func (e *Engine) registerTasks() {
	// BROWSER TASKS
	e.taskRegistry.RegisterTask("navigate", &NavigateTask{})
	e.taskRegistry.RegisterTask("back", &BackTask{})
	e.taskRegistry.RegisterTask("forward", &ForwardTask{})
	e.taskRegistry.RegisterTask("reload", &ReloadTask{})
	e.taskRegistry.RegisterTask("waitForLoad", &WaitForLoadTask{})
	e.taskRegistry.RegisterTask("takeScreenshot", &TakeScreenshotTask{})
	e.taskRegistry.RegisterTask("executeScript", &ExecuteScriptTask{})

	// INTERACTION TASKS
	e.taskRegistry.RegisterTask("click", &ClickTask{})
	e.taskRegistry.RegisterTask("type", &TypeTask{})
	e.taskRegistry.RegisterTask("select", &SelectTask{})
	e.taskRegistry.RegisterTask("hover", &HoverTask{})
	e.taskRegistry.RegisterTask("scroll", &ScrollTask{})

	// EXTRACTION TASKS
	e.taskRegistry.RegisterTask("extractText", &ExtractTextTask{})
	e.taskRegistry.RegisterTask("extractAttribute", &ExtractAttributeTask{})
	e.taskRegistry.RegisterTask("extractLinks", &ExtractLinksTask{})
	e.taskRegistry.RegisterTask("extractImages", &ExtractImagesTask{})

	// ASSET TASKS
	e.taskRegistry.RegisterTask("downloadAsset", &DownloadAssetTask{})
	e.taskRegistry.RegisterTask("saveAsset", &SaveAssetTask{})

	// FLOW CONTROL TASKS
	e.taskRegistry.RegisterTask("conditional", &ConditionalTask{})
	e.taskRegistry.RegisterTask("loop", &LoopTask{})
	e.taskRegistry.RegisterTask("wait", &WaitTask{})

	// RESOURCE TASKS
	e.taskRegistry.RegisterTask("createBrowser", &CreateBrowserTask{})
	e.taskRegistry.RegisterTask("createPage", &CreatePageTask{})
	e.taskRegistry.RegisterTask("disposeBrowser", &DisposeBrowserTask{})
	e.taskRegistry.RegisterTask("disposePage", &DisposePageTask{})
}

// INIT PLAYWRIGHT
func (e *Engine) initPlaywright() error {
	e.initMu.Lock()
	defer e.initMu.Unlock()

	log.Printf("PLAYWRIGHT INITIALIZING WITH %d BROWSERS IN POOL", len(e.browserPool))

	// AVOID DOUBLE INITIALIZATION
	if e.initialized {
		log.Printf("PLAYWRIGHT WAS ALREADY INITIALIZED WITH %d BROWSERS IN POOL", len(e.browserPool))
		return nil
	}

	// INSTALL PLAYWRIGHT IF NEEDED
	if err := playwright.Install(); err != nil {
		log.Printf("COULD NOT INSTALL PLAYWRIGHT: %v", err)
		return err
	}

	// START PLAYWRIGHT
	log.Printf("STARTING PLAYWRIGHT")
	pw, err := playwright.Run()
	if err != nil {
		log.Printf("COULD NOT START PLAYWRIGHT: %v", err)
		return err
	}

	e.playwright = pw
	e.initialized = true
	log.Printf("PLAYWRIGHT INITIALIZED WITH %d BROWSERS IN POOL", len(e.browserPool))
	return nil
}

// ENSURE PLAYWRIGHT IS INITIALIZED
func (e *Engine) ensureInitialized() error {
	log.Printf("PLAYWRIGHT INIT CHECK STARTED")
	if !e.initialized {
		e.initMu.Lock()
		defer e.initMu.Unlock()
		if !e.initialized {
			log.Printf("INITIALIZING PLAYWRIGHT")
			return e.initPlaywright()
		}
	}
	log.Printf("PLAYWRIGHT ALREADY INITIALIZED")
	return nil
}

// LAUNCH BROWSER WITH STEALTH MODE
func (e *Engine) launchBrowser(headless bool) (*playwright.Browser, error) {
	log.Printf("LAUNCHING BROWSER (HEADLESS: %v)", headless)
	if err := e.ensureInitialized(); err != nil {
		log.Printf("PLAYWRIGHT INIT CHECK FAILED: %v", err)
		return nil, err
	}

	if e.playwright == nil {
		log.Printf("PLAYWRIGHT WAS NOT INITIALIZED!")
		return nil, ErrPlaywrightNotInitialized
	}

	// LAUNCH BROWSER WITH STEALTH OPTIONS
	log.Printf("OPENING BROWSER")
	browser, err := e.playwright.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--disable-gpu",
			"--disable-dev-shm-usage",
			"--disable-setuid-sandbox",
			"--no-sandbox",
			"--disable-blink-features=AutomationControlled",
			"--disable-features=IsolateOrigins,site-per-process",
			"--disable-site-isolation-trials",
			"--ignore-certificate-errors",
			"--disable-web-security",
			"--allow-running-insecure-content",
		},
	})

	if err != nil {
		log.Printf("BROWSER LAUNCH FAILED: %v", err)
		return nil, fmt.Errorf("COULD NOT LAUNCH BROWSER: %v", err)
	}

	log.Printf("BROWSER LAUNCHED SUCCESSFULLY")
	return &browser, nil
}

// RUN JOB
func (e *Engine) RunJob(jobID string) error {
	log.Printf("STARTING JOB %s", jobID)
	if err := e.ensureInitialized(); err != nil {
		log.Printf("PLAYWRIGHT NOT INITIALIZED FOR JOB %s: %v", jobID, err)
		return err
	}

	e.mu.Lock()
	// CHECK IF JOB IS ALREADY RUNNING
	if _, running := e.runningJobs[jobID]; running {
		log.Printf("JOB %s IS ALREADY RUNNING", jobID)
		e.mu.Unlock()
		return ErrJobAlreadyRunning
	}
	e.mu.Unlock()

	// GET JOB FROM DATABASE
	var job models.Job
	if err := e.db.First(&job, "id = ?", jobID).Error; err != nil {
		log.Printf("JOB %s NOT FOUND: %v", jobID, err)
		return fmt.Errorf("FAILED TO FIND JOB: %v", err)
	}

	// UPDATE JOB STATUS
	log.Printf("UPDATING JOB %s STATUS TO RUNNING", jobID)
	e.db.Model(&job).Updates(map[string]any{
		"status":   "running",
		"last_run": time.Now(),
	})

	// CREATE CONTEXT WITH TIMEOUT
	timeout := time.Duration(e.cfg.DefaultTimeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// RECORD JOB START
	e.mu.Lock()
	e.runningJobs[jobID] = cancel
	e.jobStartTimes[jobID] = time.Now()

	// INITIALIZE JOB PROGRESS
	e.jobProgress[jobID] = JobProgress{
		TotalTasks:     0, // WILL BE CALCULATED FROM PIPELINE
		CompletedTasks: 0,
		CurrentStage:   "",
		StageProgress:  make(map[string]int),
		Status:         "running",
		Errors:         []string{},
		Assets:         0,
		TaskResults:    make(map[string]TaskData),
	}
	e.mu.Unlock()

	log.Printf("JOB %s REGISTERED AND STARTING", jobID)

	// RUN JOB IN GOROUTINE WITH IMPROVED ERROR HANDLING
	go e.executePipeline(ctx, cancel, jobID, &job)

	return nil
}

// EXECUTE JOB PIPELINE
func (e *Engine) executePipeline(ctx context.Context, cancel context.CancelFunc, jobID string, job *models.Job) {
	defer cancel()
	defer e.finishJob(jobID)

	log.Printf("JOB %s PIPELINE EXECUTION STARTED", jobID)

	// CREATE LOGGER FOR THIS JOB
	jobLogger := log.New(log.Writer(), fmt.Sprintf("[JOB %s] ", jobID), log.LstdFlags)

	// GET PIPELINE STAGES
	var pipeline []models.Stage
	if err := json.Unmarshal([]byte(job.Pipeline), &pipeline); err != nil {
		jobLogger.Printf("FAILED TO PARSE PIPELINE: %v", err)
		e.updateJobStatus(jobID, "error")
		e.addJobError(jobID, fmt.Sprintf("Failed to parse pipeline: %v", err))
		return
	}

	// COUNT TOTAL TASKS FOR PROGRESS TRACKING
	totalTasks := 0
	for _, stage := range pipeline {
		totalTasks += len(stage.Tasks)
	}

	e.mu.Lock()
	progress := e.jobProgress[jobID]
	progress.TotalTasks = totalTasks
	e.jobProgress[jobID] = progress
	e.mu.Unlock()

	// EXECUTE EACH STAGE IN SEQUENCE
	for stageIndex, stage := range pipeline {
		jobLogger.Printf("STARTING STAGE %d: %s", stageIndex+1, stage.Name)

		e.mu.Lock()
		progress := e.jobProgress[jobID]
		progress.CurrentStage = stage.Name
		e.jobProgress[jobID] = progress
		e.mu.Unlock()

		// CHECK IF STAGE HAS A CONDITION AND EVALUATE IT
		if stage.Condition.Type != "" && stage.Condition.Type != "always" {
			shouldExecute, err := e.evaluateCondition(ctx, jobID, stage.Condition)
			if err != nil {
				jobLogger.Printf("FAILED TO EVALUATE STAGE CONDITION: %v", err)
				e.addJobError(jobID, fmt.Sprintf("Failed to evaluate stage condition: %v", err))
				continue // SKIP THIS STAGE BUT CONTINUE PIPELINE
			}

			if !shouldExecute {
				jobLogger.Printf("SKIPPING STAGE %s DUE TO CONDITION", stage.Name)
				continue
			}
		}

		// EXECUTE TASKS BASED ON PARALLELISM CONFIG
		switch stage.Parallelism.Mode {
		case "sequential":
			err := e.executeSequentialTasks(ctx, jobID, job, stage, jobLogger)
			if err != nil {
				jobLogger.Printf("ERROR EXECUTING SEQUENTIAL TASKS: %v", err)
				if ctx.Err() != nil {
					// TIMEOUT OR CANCELLED
					return
				}
			}

		case "parallel":
			err := e.executeParallelTasks(ctx, jobID, job, stage, jobLogger)
			if err != nil {
				jobLogger.Printf("ERROR EXECUTING PARALLEL TASKS: %v", err)
				if ctx.Err() != nil {
					// TIMEOUT OR CANCELLED
					return
				}
			}

		case "worker-per-item":
			// SPECIAL PARALLELISM MODE WHERE EACH ITEM IN THE INPUT GETS ITS OWN WORKER
			err := e.executeWorkerPerItemTasks(ctx, jobID, job, stage, jobLogger)
			if err != nil {
				jobLogger.Printf("ERROR EXECUTING WORKER-PER-ITEM TASKS: %v", err)
				if ctx.Err() != nil {
					// TIMEOUT OR CANCELLED
					return
				}
			}

		default:
			// DEFAULT TO SEQUENTIAL
			err := e.executeSequentialTasks(ctx, jobID, job, stage, jobLogger)
			if err != nil {
				jobLogger.Printf("ERROR EXECUTING DEFAULT SEQUENTIAL TASKS: %v", err)
				if ctx.Err() != nil {
					// TIMEOUT OR CANCELLED
					return
				}
			}
		}

		// CHECK CONTEXT BEFORE CONTINUING TO NEXT STAGE
		if ctx.Err() != nil {
			jobLogger.Printf("CONTEXT DONE, STOPPING PIPELINE: %v", ctx.Err())
			return
		}
	}

	// PIPELINE COMPLETED SUCCESSFULLY
	jobLogger.Printf("PIPELINE EXECUTION COMPLETED SUCCESSFULLY")
	e.updateJobStatus(jobID, "completed")
}

// EXECUTE TASKS SEQUENTIALLY
func (e *Engine) executeSequentialTasks(ctx context.Context, jobID string, job *models.Job, stage models.Stage, logger *log.Logger) error {
	logger.Printf("EXECUTING %d TASKS SEQUENTIALLY", len(stage.Tasks))

	for taskIndex, task := range stage.Tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// EXECUTE TASK
			logger.Printf("EXECUTING TASK %d: %s (%s)", taskIndex+1, task.Name, task.Type)

			// CHECK TASK CONDITION
			if task.Condition.Type != "" && task.Condition.Type != "always" {
				shouldExecute, err := e.evaluateCondition(ctx, jobID, task.Condition)
				if err != nil {
					logger.Printf("FAILED TO EVALUATE TASK CONDITION: %v", err)
					e.addJobError(jobID, fmt.Sprintf("Failed to evaluate task condition: %v", err))
					continue // SKIP THIS TASK BUT CONTINUE
				}

				if !shouldExecute {
					logger.Printf("SKIPPING TASK %s DUE TO CONDITION", task.Name)
					continue
				}
			}

			// PREPARE TASK INPUTS
			taskInputs, err := e.prepareTaskInputs(jobID, task)
			if err != nil {
				logger.Printf("FAILED TO PREPARE TASK INPUTS: %v", err)
				e.addJobError(jobID, fmt.Sprintf("Failed to prepare task inputs: %v", err))
				continue
			}

			// EXECUTE THE TASK
			result, err := e.executeTask(ctx, jobID, task, taskInputs, logger)
			if err != nil {
				logger.Printf("TASK EXECUTION FAILED: %v", err)
				e.addJobError(jobID, fmt.Sprintf("Task execution failed: %v", err))

				// IF TASK HAS RETRY CONFIG, ATTEMPT RETRIES
				if task.RetryConfig.MaxRetries > 0 {
					retryResult, retryErr := e.retryTask(ctx, jobID, task, taskInputs, logger)
					if retryErr == nil {
						// RETRY SUCCEEDED
						result = retryResult
						err = nil
					}
				}

				if err != nil && ctx.Err() != nil {
					// TIMEOUT OR CANCELLED
					return ctx.Err()
				}
			}

			// STORE TASK RESULT
			if err == nil {
				e.mu.Lock()
				progress := e.jobProgress[jobID]
				progress.TaskResults[task.ID] = result
				progress.CompletedTasks++
				e.jobProgress[jobID] = progress
				e.mu.Unlock()

				logger.Printf("TASK %s COMPLETED SUCCESSFULLY", task.Name)
			}
		}
	}

	return nil
}

// EXECUTE TASKS IN PARALLEL
func (e *Engine) executeParallelTasks(ctx context.Context, jobID string, job *models.Job, stage models.Stage, logger *log.Logger) error {
	// DETERMINE MAX WORKERS
	maxWorkers := stage.Parallelism.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 5 // DEFAULT
	}
	if maxWorkers > len(stage.Tasks) {
		maxWorkers = len(stage.Tasks)
	}

	logger.Printf("EXECUTING %d TASKS WITH %d PARALLEL WORKERS", len(stage.Tasks), maxWorkers)

	// CREATE WAIT GROUP AND ERROR CHANNEL
	var wg sync.WaitGroup
	errChan := make(chan error, len(stage.Tasks))

	// CREATE TASK QUEUE
	taskQueue := make(chan models.Task, len(stage.Tasks))

	// ADD TASKS TO QUEUE
	for _, task := range stage.Tasks {
		// CHECK TASK CONDITION BEFORE ADDING TO QUEUE
		if task.Condition.Type != "" && task.Condition.Type != "always" {
			shouldExecute, err := e.evaluateCondition(ctx, jobID, task.Condition)
			if err != nil {
				logger.Printf("FAILED TO EVALUATE TASK CONDITION: %v", err)
				e.addJobError(jobID, fmt.Sprintf("Failed to evaluate task condition: %v", err))
				continue
			}

			if !shouldExecute {
				logger.Printf("SKIPPING TASK %s DUE TO CONDITION", task.Name)
				continue
			}
		}

		taskQueue <- task
	}
	close(taskQueue)

	// START WORKERS
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerLogger := log.New(logger.Writer(), fmt.Sprintf("[WORKER %d] ", workerID), 0)

			for task := range taskQueue {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				default:
					workerLogger.Printf("EXECUTING TASK: %s (%s)", task.Name, task.Type)

					// PREPARE TASK INPUTS
					taskInputs, err := e.prepareTaskInputs(jobID, task)
					if err != nil {
						workerLogger.Printf("FAILED TO PREPARE TASK INPUTS: %v", err)
						e.addJobError(jobID, fmt.Sprintf("Failed to prepare task inputs: %v", err))
						continue
					}

					// EXECUTE THE TASK
					result, err := e.executeTask(ctx, jobID, task, taskInputs, workerLogger)
					if err != nil {
						workerLogger.Printf("TASK EXECUTION FAILED: %v", err)
						e.addJobError(jobID, fmt.Sprintf("Task execution failed: %v", err))

						// IF TASK HAS RETRY CONFIG, ATTEMPT RETRIES
						if task.RetryConfig.MaxRetries > 0 {
							retryResult, retryErr := e.retryTask(ctx, jobID, task, taskInputs, workerLogger)
							if retryErr == nil {
								// RETRY SUCCEEDED
								result = retryResult
								err = nil
							}
						}

						if err != nil && ctx.Err() != nil {
							errChan <- ctx.Err()
							return
						}
					}

					// STORE TASK RESULT
					if err == nil {
						e.mu.Lock()
						progress := e.jobProgress[jobID]
						progress.TaskResults[task.ID] = result
						progress.CompletedTasks++
						e.jobProgress[jobID] = progress
						e.mu.Unlock()

						workerLogger.Printf("TASK %s COMPLETED SUCCESSFULLY", task.Name)
					}
				}
			}
		}(i)
	}

	// WAIT FOR ALL WORKERS OR CONTEXT CANCELLATION
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		// ALL WORKERS COMPLETED
		logger.Printf("ALL PARALLEL TASKS COMPLETED")

	case err := <-errChan:
		// WORKER ENCOUNTERED ERROR
		return err

	case <-ctx.Done():
		// CONTEXT CANCELLED
		return ctx.Err()
	}

	return nil
}

// EXECUTE TASKS WITH WORKER-PER-ITEM PARALLELISM
func (e *Engine) executeWorkerPerItemTasks(ctx context.Context, jobID string, job *models.Job, stage models.Stage, logger *log.Logger) error {
	if len(stage.Tasks) == 0 {
		logger.Printf("NO TASKS TO EXECUTE")
		return nil
	}

	// THIS MODE TYPICALLY WORKS WITH ONE TASK THAT PROCESSES MULTIPLE ITEMS
	primaryTask := stage.Tasks[0]
	logger.Printf("EXECUTING WORKER-PER-ITEM FOR TASK: %s", primaryTask.Name)

	// CHECK INPUTS TO IDENTIFY THE ITEM SOURCE
	itemSourceID := ""
	for _, inputRef := range primaryTask.InputRefs {
		// CHECK IF THIS INPUT CONTAINS AN ARRAY
		e.mu.Lock()
		inputData, exists := e.jobProgress[jobID].TaskResults[inputRef]
		e.mu.Unlock()

		if exists && inputData.Type == "array" {
			itemSourceID = inputRef
			break
		}
	}

	if itemSourceID == "" {
		err := fmt.Errorf("NO ARRAY INPUT FOUND FOR WORKER-PER-ITEM TASK")
		logger.Printf("%v", err)
		return err
	}

	// GET THE ITEMS TO PROCESS
	e.mu.Lock()
	itemsData := e.jobProgress[jobID].TaskResults[itemSourceID]
	e.mu.Unlock()

	items, ok := itemsData.Value.([]interface{})
	if !ok {
		err := fmt.Errorf("INVALID ITEM SOURCE DATA TYPE")
		logger.Printf("%v", err)
		return err
	}

	logger.Printf("PROCESSING %d ITEMS WITH WORKER-PER-ITEM", len(items))

	// DETERMINE MAX WORKERS
	maxWorkers := stage.Parallelism.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 5 // DEFAULT
	}
	if maxWorkers > len(items) {
		maxWorkers = len(items)
	}

	// CREATE WAIT GROUP AND ERROR CHANNEL
	var wg sync.WaitGroup
	errChan := make(chan error, len(items))
	resultChan := make(chan struct {
		index  int
		result TaskData
	}, len(items))

	// CREATE ITEM QUEUE
	type queueItem struct {
		index int
		item  interface{}
	}
	itemQueue := make(chan queueItem, len(items))

	// ADD ITEMS TO QUEUE
	for i, item := range items {
		itemQueue <- queueItem{index: i, item: item}
	}
	close(itemQueue)

	// START WORKERS
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerLogger := log.New(logger.Writer(), fmt.Sprintf("[WORKER %d] ", workerID), 0)

			for qItem := range itemQueue {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				default:
					workerLogger.Printf("PROCESSING ITEM %d", qItem.index)

					// CREATE A COPY OF THE TASK WITH A UNIQUE ID
					taskCopy := primaryTask
					taskCopy.ID = fmt.Sprintf("%s_item_%d", primaryTask.ID, qItem.index)

					// CREATE A CUSTOM INPUT WITH THIS ITEM
					itemInputID := fmt.Sprintf("%s_item_%d", itemSourceID, qItem.index)

					e.mu.Lock()
					progress := e.jobProgress[jobID]
					progress.TaskResults[itemInputID] = TaskData{
						Type:  "object",
						Value: qItem.item,
					}
					e.mu.Unlock()

					// REPLACE THE ARRAY INPUT WITH THE SINGLE ITEM INPUT
					taskInputs := make(map[string]interface{})
					for _, inputRef := range taskCopy.InputRefs {
						if inputRef == itemSourceID {
							taskInputs["item"] = qItem.item
						} else {
							e.mu.Lock()
							inputData, exists := e.jobProgress[jobID].TaskResults[inputRef]
							e.mu.Unlock()

							if exists {
								taskInputs[inputRef] = inputData.Value
							}
						}
					}

					// EXECUTE THE TASK
					result, err := e.executeTask(ctx, jobID, taskCopy, taskInputs, workerLogger)
					if err != nil {
						workerLogger.Printf("TASK EXECUTION FAILED FOR ITEM %d: %v", qItem.index, err)
						e.addJobError(jobID, fmt.Sprintf("Task execution failed for item %d: %v", qItem.index, err))

						// IF TASK HAS RETRY CONFIG, ATTEMPT RETRIES
						if taskCopy.RetryConfig.MaxRetries > 0 {
							retryResult, retryErr := e.retryTask(ctx, jobID, taskCopy, taskInputs, workerLogger)
							if retryErr == nil {
								// RETRY SUCCEEDED
								result = retryResult
								err = nil
							}
						}

						if err != nil && ctx.Err() != nil {
							errChan <- ctx.Err()
							return
						}
					}

					// STORE INDIVIDUAL RESULT
					if err == nil {
						resultChan <- struct {
							index  int
							result TaskData
						}{index: qItem.index, result: result}

						e.mu.Lock()
						progress := e.jobProgress[jobID]
						progress.TaskResults[taskCopy.ID] = result
						progress.CompletedTasks++
						e.jobProgress[jobID] = progress
						e.mu.Unlock()

						workerLogger.Printf("ITEM %d PROCESSED SUCCESSFULLY", qItem.index)
					}
				}
			}
		}(i)
	}

	// WAIT FOR ALL WORKERS OR CONTEXT CANCELLATION
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
		close(resultChan)
	}()

	// COLLECT RESULTS
	results := make([]interface{}, len(items))
	resultCollector := make(chan struct{})

	go func() {
		for r := range resultChan {
			results[r.index] = r.result.Value
		}
		close(resultCollector)
	}()

	select {
	case <-waitChan:
		// WAIT FOR RESULT COLLECTION TO COMPLETE
		<-resultCollector

		// STORE COMBINED RESULTS
		e.mu.Lock()
		progress := e.jobProgress[jobID]
		progress.TaskResults[primaryTask.ID+"_combined"] = TaskData{
			Type:  "array",
			Value: results,
		}
		e.jobProgress[jobID] = progress
		e.mu.Unlock()

		logger.Printf("ALL ITEMS PROCESSED SUCCESSFULLY")

	case err := <-errChan:
		// WORKER ENCOUNTERED ERROR
		return err

	case <-ctx.Done():
		// CONTEXT CANCELLED
		return ctx.Err()
	}

	return nil
}

// PREPARE TASK INPUTS
func (e *Engine) prepareTaskInputs(jobID string, task models.Task) (map[string]interface{}, error) {
	inputs := make(map[string]interface{})

	// GET TASK IMPLEMENTATION
	taskImpl, err := e.taskRegistry.GetTask(task.Type)
	if err != nil {
		return nil, err
	}

	// GET INPUT SCHEMA
	inputSchema := taskImpl.GetInputSchema()

	// FOR EACH INPUT REFERENCE, GET THE TASK RESULT
	for inputName, inputType := range inputSchema {
		// CHECK IF INPUT IS IN CONFIGURATION
		if val, ok := task.Config[inputName]; ok {
			inputs[inputName] = val
			continue
		}

		// LOOK FOR INPUT IN TASK REFERENCES
		found := false
		for _, inputRef := range task.InputRefs {
			e.mu.Lock()
			inputData, exists := e.jobProgress[jobID].TaskResults[inputRef]
			e.mu.Unlock()

			if exists {
				// CHECK TYPE COMPATIBILITY
				if inputData.Type == inputType || inputType == "any" {
					inputs[inputName] = inputData.Value
					found = true
					break
				}
			}
		}

		// IF REQUIRED INPUT NOT FOUND, RETURN ERROR
		if !found && !strings.HasSuffix(inputType, "?") { // OPTIONAL INPUTS END WITH ?
			return nil, fmt.Errorf("REQUIRED INPUT %s NOT FOUND FOR TASK %s", inputName, task.Name)
		}
	}

	return inputs, nil
}

// EXECUTE A SINGLE TASK
func (e *Engine) executeTask(ctx context.Context, jobID string, task models.Task, inputs map[string]interface{}, logger *log.Logger) (TaskData, error) {
	// GET TASK IMPLEMENTATION
	taskImpl, err := e.taskRegistry.GetTask(task.Type)
	if err != nil {
		return TaskData{}, err
	}

	// VALIDATE TASK CONFIG
	if err := taskImpl.ValidateConfig(task.Config); err != nil {
		return TaskData{}, fmt.Errorf("INVALID TASK CONFIG: %v", err)
	}

	// MERGE INPUTS WITH CONFIG
	config := make(map[string]interface{})
	for k, v := range task.Config {
		config[k] = v
	}
	for k, v := range inputs {
		config[k] = v
	}

	// CREATE TASK CONTEXT
	taskCtx := &TaskContext{
		JobID:           jobID,
		ResourceManager: e.resourceManager,
		Context:         ctx,
		Logger:          logger,
		Engine:          e,
	}

	// EXECUTE TASK
	logger.Printf("EXECUTING TASK %s (%s)", task.Name, task.Type)
	return taskImpl.Execute(taskCtx, config)
}

// RETRY A FAILED TASK
func (e *Engine) retryTask(ctx context.Context, jobID string, task models.Task, inputs map[string]interface{}, logger *log.Logger) (TaskData, error) {
	maxRetries := task.RetryConfig.MaxRetries
	delayMS := task.RetryConfig.DelayMS
	backoffRate := task.RetryConfig.BackoffRate

	if backoffRate <= 0 {
		backoffRate = 1.5 // DEFAULT BACKOFF RATE
	}

	if delayMS <= 0 {
		delayMS = 1000 // DEFAULT DELAY 1 SECOND
	}

	var lastErr error
	var result TaskData

	for retry := 1; retry <= maxRetries; retry++ {
		// WAIT BEFORE RETRY WITH EXPONENTIAL BACKOFF
		delay := time.Duration(float64(delayMS) * (float64(backoffRate) * float64(retry-1)))
		logger.Printf("RETRYING TASK %s (ATTEMPT %d/%d) AFTER %v DELAY", task.Name, retry, maxRetries, delay)

		select {
		case <-time.After(time.Duration(delay) * time.Millisecond):
			// CONTINUE WITH RETRY
		case <-ctx.Done():
			// CONTEXT CANCELLED
			return TaskData{}, ctx.Err()
		}

		// EXECUTE TASK AGAIN
		result, lastErr = e.executeTask(ctx, jobID, task, inputs, logger)
		if lastErr == nil {
			// RETRY SUCCEEDED
			logger.Printf("TASK %s RETRY SUCCESSFUL (ATTEMPT %d)", task.Name, retry)
			return result, nil
		}

		logger.Printf("TASK %s RETRY FAILED (ATTEMPT %d): %v", task.Name, retry, lastErr)

		// CHECK FOR CONTEXT CANCELLATION
		if ctx.Err() != nil {
			return TaskData{}, ctx.Err()
		}
	}

	// ALL RETRIES FAILED
	logger.Printf("ALL RETRIES FAILED FOR TASK %s", task.Name)
	return TaskData{}, lastErr
}

// EVALUATE A CONDITION
func (e *Engine) evaluateCondition(ctx context.Context, jobID string, condition models.Condition) (bool, error) {
	switch condition.Type {
	case "always":
		return true, nil

	case "never":
		return false, nil

	case "javascript":
		// IN A REAL IMPLEMENTATION, WOULD USE A JS ENGINE (E.G., GOJA)
		// FOR NOW, JUST A MOCK IMPLEMENTATION
		return true, nil

	case "comparison":
		// SIMPLE COMPARISON CONDITION
		left, leftOk := condition.Config["left"]
		right, rightOk := condition.Config["right"]
		operator, operatorOk := condition.Config["operator"].(string)

		if !leftOk || !rightOk || !operatorOk {
			return false, fmt.Errorf("INVALID COMPARISON CONDITION")
		}

		switch operator {
		case "eq":
			return left == right, nil
		case "neq":
			return left != right, nil
		case "gt":
			// WOULD NEED TYPE CHECKING IN REAL IMPLEMENTATION
			return false, nil
		case "lt":
			// WOULD NEED TYPE CHECKING IN REAL IMPLEMENTATION
			return false, nil
		default:
			return false, fmt.Errorf("UNKNOWN OPERATOR: %s", operator)
		}

	default:
		return false, fmt.Errorf("UNKNOWN CONDITION TYPE: %s", condition.Type)
	}
}

// ADD JOB ERROR
func (e *Engine) addJobError(jobID string, errorMsg string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	progress := e.jobProgress[jobID]
	progress.Errors = append(progress.Errors, errorMsg)
	e.jobProgress[jobID] = progress
}

// FINISH JOB AND CLEANUP
func (e *Engine) finishJob(jobID string) {
	log.Printf("FINISHING JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	if startTime, ok := e.jobStartTimes[jobID]; ok {
		duration := time.Since(startTime)
		e.jobDurations[jobID] = duration
		delete(e.jobStartTimes, jobID)
		log.Printf("JOB %s DURATION: %v", jobID, duration)
	}

	delete(e.runningJobs, jobID)

	// CLEAN UP RESOURCES
	e.resourceManager.DeleteJobResources(jobID)

	log.Printf("JOB %s FINISHED AND CLEANED UP", jobID)
}

// UPDATE JOB STATUS
func (e *Engine) updateJobStatus(jobID string, status string) {
	log.Printf("UPDATING JOB %s STATUS: %s", jobID, status)

	e.mu.Lock()
	if progress, ok := e.jobProgress[jobID]; ok {
		progress.Status = status
		e.jobProgress[jobID] = progress
	}
	e.mu.Unlock()

	if err := e.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", status).Error; err != nil {
		log.Printf("STATUS UPDATE ERROR: %v", err)
	} else {
		log.Printf("JOB %s STATUS UPDATED TO %s", jobID, status)
	}
}

// STOP JOB
func (e *Engine) StopJob(jobID string) error {
	log.Printf("STOPPING JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	cancel, running := e.runningJobs[jobID]
	if !running {
		log.Printf("JOB %s IS NOT RUNNING", jobID)
		return fmt.Errorf("JOB %s NOT RUNNING", jobID)
	}

	log.Printf("CANCELLING JOB: %s", jobID)
	cancel()

	// UPDATE JOB STATUS
	progress := e.jobProgress[jobID]
	progress.Status = "stopped"
	e.jobProgress[jobID] = progress

	return nil
}

// GET JOB PROGRESS
func (e *Engine) GetJobProgress(jobID string) (JobProgress, error) {
	log.Printf("GETTING PROGRESS FOR JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	progress, exists := e.jobProgress[jobID]
	if !exists {
		log.Printf("JOB %s NOT FOUND", jobID)
		return JobProgress{}, ErrJobNotFound
	}

	log.Printf("JOB %s PROGRESS: %d/%d TASKS", jobID, progress.CompletedTasks, progress.TotalTasks)
	return progress, nil
}

// GET JOB DURATION
func (e *Engine) GetJobDuration(jobID string) (time.Duration, error) {
	log.Printf("GETTING DURATION FOR JOB: %s", jobID)
	e.mu.Lock()
	defer e.mu.Unlock()

	if startTime, running := e.jobStartTimes[jobID]; running {
		duration := time.Since(startTime)
		log.Printf("JOB %s RUNNING DURATION: %v", jobID, duration)
		return duration, nil
	}

	duration, exists := e.jobDurations[jobID]
	if !exists {
		log.Printf("JOB %s NOT FOUND", jobID)
		return 0, ErrJobNotFound
	}

	log.Printf("JOB %s COMPLETED DURATION: %v", jobID, duration)
	return duration, nil
}

// GENERATE A UNIQUE ID
func generateID(prefix string) string {
	id := uuid.New().String()
	return fmt.Sprintf("%s_%s", prefix, id)
}

// CLEAN UP RESOURCES
func (e *Engine) Close() {
	log.Printf("ENGINE SHUTDOWN STARTED")

	// STOP ALL JOBS
	e.mu.Lock()
	jobCount := len(e.runningJobs)
	log.Printf("STOPPING %d RUNNING JOBS", jobCount)
	for jobID, cancel := range e.runningJobs {
		log.Printf("CANCELLING JOB: %s", jobID)
		cancel()
		e.updateJobStatus(jobID, "stopped")
	}
	e.runningJobs = make(map[string]context.CancelFunc)
	e.mu.Unlock()

	log.Printf("ALL JOBS STOPPED")

	// DRAIN POOL AND CLOSE BROWSERS
	log.Printf("DRAINING BROWSER POOL")
	close(e.browserPool)
	browserCount := 0
	for browser := range e.browserPool {
		browserCount++
		if browser.browser != nil {
			log.Printf("CLOSING BROWSER %d", browserCount)
			(*browser.browser).Close()
		}
	}

	log.Printf("%d BROWSERS CLOSED", browserCount)

	// STOP PLAYWRIGHT
	e.initMu.Lock()
	if e.playwright != nil {
		log.Printf("STOPPING PLAYWRIGHT")
		e.playwright.Stop()
		e.playwright = nil
		log.Printf("PLAYWRIGHT STOPPED")
	}
	e.initialized = false
	e.initMu.Unlock()

	log.Printf("ENGINE SHUTDOWN COMPLETE")
}
