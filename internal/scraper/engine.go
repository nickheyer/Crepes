package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/storage"
)

// STAGE TYPE CONSTANTS
const (
	StageFetch      = "fetch"
	StageExtract    = "extract"
	StageFilter     = "filter"
	StageTransform  = "transform"
	StageStore      = "store"
	StageMedia      = "media"
	StageProcess    = "process"
	StageFollow     = "follow"
	StagePagination = "pagination"
)

// ITEM IS THE BASIC UNIT OF DATA FLOWING THROUGH THE PIPELINE
type Item struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Content     string            `json:"content,omitempty"`
	Data        map[string]any    `json:"data,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ParentID    string            `json:"parentId,omitempty"`
	Depth       int               `json:"depth"`
	Error       error             `json:"-"`
	ErrorString string            `json:"error,omitempty"`
}

// STATUSUPDATE REPRESENTS THE CURRENT STATE OF A PIPELINE EXECUTION
type StatusUpdate struct {
	JobID           string    `json:"jobId"`
	Stage           string    `json:"stage"`
	StageID         string    `json:"stageId"`
	ItemsProcessed  int       `json:"itemsProcessed"`
	ItemsPending    int       `json:"itemsPending"`
	CurrentURL      string    `json:"currentUrl,omitempty"`
	LastError       string    `json:"lastError,omitempty"`
	LastSuccess     string    `json:"lastSuccess,omitempty"`
	LastUpdate      time.Time `json:"lastUpdate"`
	IsComplete      bool      `json:"isComplete"`
	PercentComplete float64   `json:"percentComplete"`
}

// STAGE REPRESENTS A SINGLE PROCESSING UNIT IN THE PIPELINE
type Stage struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	NextStages  []string       `json:"nextStages"`
	Concurrency int            `json:"concurrency"`

	// RUNTIME STATE - NOT PERSISTED
	inputChan  chan Item
	outputChan chan Item
	wg         *sync.WaitGroup
	metrics    struct {
		processed int64
		errors    int64
		lastItem  time.Time
	}
}

// PIPELINE ORCHESTRATES THE FLOW OF DATA THROUGH STAGES
type Pipeline struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Stages      map[string]*Stage `json:"stages"`
	EntryPoints []string          `json:"entryPoints"`
	MaxWorkers  int               `json:"maxWorkers"`
	// RUNTIME STATE - NOT PERSISTED
	jobCtx         context.Context
	cancelFunc     context.CancelFunc
	statusChan     chan StatusUpdate
	errorChan      chan error
	globalWg       *sync.WaitGroup
	stageMetrics   map[string]*StageMetrics
	processedItems map[string]bool
	itemsMutex     *sync.RWMutex
}

// STAGEMETRICS TRACKS PERFORMANCE AND STATUS OF A STAGE
type StageMetrics struct {
	Processed   int64
	Errors      int64
	LastSuccess time.Time
	LastError   time.Time
	AverageTime time.Duration
	TotalTime   time.Duration
	Times       []time.Duration
}

// NEWPIPELINE CREATES A NEW PIPELINE FROM A JOB DEFINITION
func NewPipeline(job *models.ScrapingJob) (*Pipeline, error) {
	// CONVERT THE JOB TO A PIPELINE
	var pipeline Pipeline

	// IF JOB ALREADY HAS A PIPELINE DEFINITION, USE IT
	if job.Pipeline != "" {
		if err := json.Unmarshal([]byte(job.Pipeline), &pipeline); err != nil {
			// IF IT FAILS, CREATE A NEW PIPELINE FROM THE JOB
			pipeline = ConvertJobToPipeline(job)
		}
	} else {
		// CREATE A NEW PIPELINE FROM THE JOB
		pipeline = ConvertJobToPipeline(job)
	}

	// PROPERLY INITIALIZE RUNTIME STATE
	pipeline.statusChan = make(chan StatusUpdate, 100)
	pipeline.errorChan = make(chan error, 100)
	pipeline.globalWg = &sync.WaitGroup{}
	pipeline.stageMetrics = make(map[string]*StageMetrics)
	pipeline.processedItems = make(map[string]bool)
	pipeline.itemsMutex = &sync.RWMutex{} // ENSURE MUTEX IS INITIALIZED

	// INITIALIZE METRICS FOR EACH STAGE
	for id := range pipeline.Stages {
		pipeline.stageMetrics[id] = &StageMetrics{
			Times: make([]time.Duration, 0, 100),
		}

		// ENSURE STAGE HAS INITIALIZED CHANNELS
		if pipeline.Stages[id] != nil {
			pipeline.Stages[id].inputChan = nil
			pipeline.Stages[id].outputChan = nil
			pipeline.Stages[id].wg = nil
		}
	}
	return &pipeline, nil
}

// CONVERTJOBTOPIPELINE CONVERTS A LEGACY JOB TO A PIPELINE
func ConvertJobToPipeline(job *models.ScrapingJob) Pipeline {
	pipeline := Pipeline{
		ID:          job.ID,
		Name:        fmt.Sprintf("Job %s", job.ID),
		Description: fmt.Sprintf("Pipeline for %s", job.BaseURL),
		Stages:      make(map[string]*Stage),
		MaxWorkers:  job.Rules.MaxConcurrent,
		EntryPoints: []string{},
	}

	// CREATE FETCH STAGE
	fetchStageID := uuid.New().String()
	fetchStage := &Stage{
		ID:   fetchStageID,
		Name: "Fetch Initial Page",
		Type: StageFetch,
		Config: map[string]any{
			"url":          job.BaseURL,
			"userAgent":    job.Rules.UserAgent,
			"timeout":      job.Rules.Timeout,
			"maxRedirects": 10,
		},
		NextStages:  []string{},
		Concurrency: 1,
	}
	pipeline.Stages[fetchStageID] = fetchStage
	pipeline.EntryPoints = append(pipeline.EntryPoints, fetchStageID)

	// FIND LINK SELECTORS
	for i, selector := range job.Selectors {
		if selector.Purpose == "links" {
			linkStageID := uuid.New().String()
			linkStage := &Stage{
				ID:   linkStageID,
				Name: fmt.Sprintf("Extract Links %d", i),
				Type: StageExtract,
				Config: map[string]any{
					"selector":   selector.Value,
					"attribute":  "href",
					"type":       selector.Type,
					"isOptional": selector.IsOptional,
				},
				NextStages:  []string{},
				Concurrency: 2,
			}
			pipeline.Stages[linkStageID] = linkStage

			// CONNECT FETCH TO LINK EXTRACTION
			fetchStage.NextStages = append(fetchStage.NextStages, linkStageID)

			// ADD FOLLOW STAGE
			followStageID := uuid.New().String()
			followStage := &Stage{
				ID:   followStageID,
				Name: "Follow Links",
				Type: StageFollow,
				Config: map[string]any{
					"maxDepth":     job.Rules.MaxDepth,
					"includeRegex": job.Rules.IncludeURLPattern,
					"excludeRegex": job.Rules.ExcludeURLPattern,
				},
				NextStages:  []string{}, // THIS WILL CONNECT BACK TO FETCH
				Concurrency: job.Rules.MaxConcurrent,
			}
			pipeline.Stages[followStageID] = followStage

			// CONNECT LINK EXTRACTION TO FOLLOW
			pipeline.Stages[linkStageID].NextStages = append(pipeline.Stages[linkStageID].NextStages, followStageID)

			// CIRCULAR REFERENCE: FOLLOW CONNECTS BACK TO FETCH
			// THIS CREATES THE CRAWLING LOOP
			followStage.NextStages = append(followStage.NextStages, fetchStageID)
		}
	}

	// FIND ASSET SELECTORS AND CREATE ASSET EXTRACTION STAGES
	for i, selector := range job.Selectors {
		if selector.Purpose == "assets" || selector.Purpose == "video" {
			assetStageID := uuid.New().String()
			assetStage := &Stage{
				ID:   assetStageID,
				Name: fmt.Sprintf("Extract Assets %d", i),
				Type: StageExtract,
				Config: map[string]any{
					"selector":   selector.Value,
					"attribute":  selector.Attribute,
					"type":       selector.Type,
					"isOptional": selector.IsOptional,
				},
				NextStages:  []string{},
				Concurrency: 2,
			}
			pipeline.Stages[assetStageID] = assetStage

			// CONNECT FETCH TO ASSET EXTRACTION
			fetchStage.NextStages = append(fetchStage.NextStages, assetStageID)

			// IF IT'S A MEDIA STAGE, ADD MEDIA PROCESSING
			if selector.Purpose == "video" {
				mediaStageID := uuid.New().String()
				mediaStage := &Stage{
					ID:   mediaStageID,
					Name: "Process Media",
					Type: StageMedia,
					Config: map[string]any{
						"maxSize":  job.Rules.MaxSize,
						"headless": job.Rules.VideoExtractionHeadless,
					},
					NextStages:  []string{},
					Concurrency: 1,
				}
				pipeline.Stages[mediaStageID] = mediaStage

				// CONNECT ASSET EXTRACTION TO MEDIA PROCESSING
				assetStage.NextStages = append(assetStage.NextStages, mediaStageID)

				// ADD STORAGE STAGE
				storeStageID := uuid.New().String()
				storeStage := &Stage{
					ID:   storeStageID,
					Name: "Store Media",
					Type: StageStore,
					Config: map[string]any{
						"path":      fmt.Sprintf("%s/%s", config.AppConfig.StoragePath, job.ID),
						"thumbnail": true,
					},
					NextStages:  []string{},
					Concurrency: 2,
				}
				pipeline.Stages[storeStageID] = storeStage

				// CONNECT MEDIA PROCESSING TO STORAGE
				mediaStage.NextStages = append(mediaStage.NextStages, storeStageID)
			} else {
				// REGULAR ASSET, ADD STORAGE DIRECTLY
				storeStageID := uuid.New().String()
				storeStage := &Stage{
					ID:   storeStageID,
					Name: "Store Asset",
					Type: StageStore,
					Config: map[string]any{
						"path":      fmt.Sprintf("%s/%s", config.AppConfig.StoragePath, job.ID),
						"thumbnail": true,
					},
					NextStages:  []string{},
					Concurrency: 2,
				}
				pipeline.Stages[storeStageID] = storeStage

				// CONNECT ASSET EXTRACTION TO STORAGE
				assetStage.NextStages = append(assetStage.NextStages, storeStageID)
			}
		}
	}

	// FIND PAGINATION SELECTORS AND CREATE PAGINATION STAGES
	for i, selector := range job.Selectors {
		if selector.Purpose == "pagination" {
			paginationStageID := uuid.New().String()
			paginationStage := &Stage{
				ID:   paginationStageID,
				Name: fmt.Sprintf("Handle Pagination %d", i),
				Type: StagePagination,
				Config: map[string]any{
					"selector":  selector.Value,
					"attribute": "href",
					"type":      selector.Type,
					"maxPages":  job.Rules.MaxPages,
				},
				NextStages:  []string{fetchStageID}, // CONNECT BACK TO FETCH
				Concurrency: 1,
			}
			pipeline.Stages[paginationStageID] = paginationStage

			// CONNECT FETCH TO PAGINATION
			fetchStage.NextStages = append(fetchStage.NextStages, paginationStageID)
		}
	}

	return pipeline
}

// EXECUTE RUNS THE PIPELINE WITH THE GIVEN CONTEXT
func (p *Pipeline) Execute(rootURL string) error {
	p.jobCtx, p.cancelFunc = context.WithTimeout(context.Background(), 30*time.Minute)

	// INITIALIZE CHANNELS FOR EACH STAGE
	for _, stage := range p.Stages {
		if stage == nil {
			continue // SKIP NIL STAGES
		}
		stage.inputChan = make(chan Item, 100)
		stage.outputChan = make(chan Item, 100)
		stage.wg = &sync.WaitGroup{}
	}

	// START WORKERS FOR EACH STAGE
	for id, stage := range p.Stages {
		if stage == nil {
			continue // SKIP NIL STAGES
		}

		log.Printf("Starting stage %s: %s", id, stage.Name)
		workers := stage.Concurrency
		if workers <= 0 {
			workers = 1
		}
		for i := 0; i < workers; i++ {
			p.globalWg.Add(1)
			stage.wg.Add(1)
			go func(s *Stage, workerID int) {
				defer p.globalWg.Done()
				defer s.wg.Done()
				p.stageWorker(s, workerID)
			}(stage, i)
		}
	}

	// CONNECT STAGE OUTPUTS TO INPUTS OF NEXT STAGES
	for _, stage := range p.Stages {
		for _, nextStageID := range stage.NextStages {
			nextStage, exists := p.Stages[nextStageID]
			if !exists {
				log.Printf("Warning: Next stage %s not found for stage %s", nextStageID, stage.ID)
				continue
			}

			// START A GOROUTINE TO FORWARD ITEMS
			p.globalWg.Add(1)
			go func(source, target *Stage) {
				defer p.globalWg.Done()

				for {
					select {
					case item, ok := <-source.outputChan:
						if !ok {
							return // CHANNEL CLOSED
						}

						select {
						case target.inputChan <- item:
							// ITEM FORWARDED
						case <-p.jobCtx.Done():
							return // CONTEXT CANCELED
						}

					case <-p.jobCtx.Done():
						return // CONTEXT CANCELED
					}
				}
			}(stage, nextStage)
		}
	}

	// INJECT INITIAL ITEM
	for _, entryPointID := range p.EntryPoints {
		entryStage, exists := p.Stages[entryPointID]
		if !exists {
			continue
		}

		// CREATE INITIAL ITEM
		initialItem := Item{
			ID:    uuid.New().String(),
			URL:   rootURL,
			Depth: 0,
			Metadata: map[string]string{
				"entryPoint": "true",
				"timestamp":  time.Now().Format(time.RFC3339),
			},
			Data: make(map[string]any),
		}

		// SEND TO ENTRY POINT STAGE
		entryStage.inputChan <- initialItem
		log.Printf("Added initial URL to pipeline: %s", rootURL)
	}

	// WAIT FOR ALL WORK TO COMPLETE OR CONTEXT TO BE CANCELED
	done := make(chan struct{})
	go func() {
		p.globalWg.Wait()
		close(done)
	}()

	// WAIT FOR COMPLETION OR CANCELLATION
	select {
	case <-done:
		log.Printf("Pipeline execution completed successfully")
	case <-p.jobCtx.Done():
		if p.jobCtx.Err() == context.DeadlineExceeded {
			log.Printf("Pipeline execution timed out")
			return fmt.Errorf("pipeline execution timed out")
		}
		log.Printf("Pipeline execution canceled")
		return fmt.Errorf("pipeline execution canceled")
	}

	return nil
}

// STAGEWORKER PROCESSES ITEMS FOR A SPECIFIC STAGE
func (p *Pipeline) stageWorker(stage *Stage, workerID int) {
	if stage == nil {
		log.Printf("Warning: nil stage passed to worker %d", workerID)
		return
	}

	if stage.inputChan == nil {
		log.Printf("Warning: nil input channel for stage %s, worker %d", stage.ID, workerID)
		return
	}

	log.Printf("Started worker %d for stage %s (%s)", workerID, stage.ID, stage.Name)

	for {
		select {
		case item, ok := <-stage.inputChan:
			if !ok {
				log.Printf("Input channel closed for stage %s, worker %d exiting", stage.ID, workerID)
				return
			}

			// CHECK IF ITEM HAS ALREADY BEEN PROCESSED BY THIS STAGE
			if p.itemsMutex == nil {
				p.itemsMutex = &sync.RWMutex{} // SAFETY CHECK
			}

			p.itemsMutex.Lock()
			itemKey := fmt.Sprintf("%s:%s", stage.ID, item.ID)
			alreadyProcessed := p.processedItems[itemKey]
			p.itemsMutex.Unlock()

			if alreadyProcessed {
				log.Printf("Item %s already processed by stage %s, skipping", item.ID, stage.ID)
				continue
			}

			// MARK ITEM AS PROCESSED
			p.itemsMutex.Lock()
			p.processedItems[itemKey] = true
			p.itemsMutex.Unlock()

			// PROCESS THE ITEM
			startTime := time.Now()

			// SEND STATUS UPDATE BEFORE PROCESSING
			p.statusChan <- StatusUpdate{
				JobID:          p.ID,
				Stage:          stage.Name,
				StageID:        stage.ID,
				ItemsProcessed: int(stage.metrics.processed),
				CurrentURL:     item.URL,
				LastUpdate:     time.Now(),
			}

			// EXECUTE THE STAGE'S PROCESSOR BASED ON TYPE
			results, err := p.executeStageProcessor(stage, item)

			// UPDATE METRICS
			stage.metrics.lastItem = time.Now()
			processingTime := time.Since(startTime)

			// UPDATE STAGE METRICS
			p.updateStageMetrics(stage.ID, processingTime, err == nil)

			if err != nil {
				stage.metrics.errors++
				log.Printf("Error processing item %s in stage %s: %v", item.ID, stage.Name, err)

				// SEND ERROR TO ERROR CHANNEL
				errorWithContext := &ContextualError{
					URL:       item.URL,
					Stage:     stage.Name,
					StageID:   stage.ID,
					RawError:  err,
					Timestamp: time.Now(),
				}

				if len(p.errorChan) < cap(p.errorChan) {
					p.errorChan <- errorWithContext
				}

				// IF ITEM HAS ERROR HANDLING CONFIG, PROCESS IT
				if errorConfig, ok := stage.Config["onError"].(map[string]any); ok {
					if failAction, ok := errorConfig["action"].(string); ok {
						switch failAction {
						case "continue":
							// JUST CONTINUE TO NEXT ITEM
						case "retry":
							maxRetries := 3
							if maxRetriesConfig, ok := errorConfig["maxRetries"].(int); ok {
								maxRetries = maxRetriesConfig
							}

							retryCount, ok := item.Data["retryCount"].(int)
							if !ok {
								retryCount = 0
							}

							if retryCount < maxRetries {
								// INCREMENT RETRY COUNT AND REQUEUE
								item.Data["retryCount"] = retryCount + 1
								item.Data["lastError"] = err.Error()

								// REQUEUE WITH DELAY
								time.AfterFunc(time.Second*time.Duration(retryCount+1), func() {
									select {
									case stage.inputChan <- item:
										// REQUEUED SUCCESSFULLY
									case <-p.jobCtx.Done():
										// CONTEXT CANCELED
									}
								})
							}
						case "abort":
							// CANCEL THE WHOLE PIPELINE
							p.cancelFunc()
							return
						}
					}
				}

				// EVEN IN CASE OF ERROR, WE MIGHT HAVE PARTIAL RESULTS
				if results == nil {
					results = []Item{}
				}
			}

			// INCREMENT PROCESSED COUNT
			stage.metrics.processed++

			// SEND RESULTS TO OUTPUT CHANNEL
			for _, result := range results {
				select {
				case stage.outputChan <- result:
					// RESULT SENT SUCCESSFULLY
				case <-p.jobCtx.Done():
					return // CONTEXT CANCELED
				}
			}

			// SEND STATUS UPDATE AFTER PROCESSING
			p.statusChan <- StatusUpdate{
				JobID:          p.ID,
				Stage:          stage.Name,
				StageID:        stage.ID,
				ItemsProcessed: int(stage.metrics.processed),
				ItemsPending:   len(stage.inputChan),
				CurrentURL:     item.URL,
				LastUpdate:     time.Now(),
				LastError:      err.Error(),
				LastSuccess:    item.URL,
			}

		case <-p.jobCtx.Done():
			log.Printf("Context canceled for stage %s, worker %d exiting", stage.ID, workerID)
			return
		}
	}
}

// EXECUTESTAGEPROCESSOR CALLS THE APPROPRIATE PROCESSOR BASED ON STAGE TYPE
func (p *Pipeline) executeStageProcessor(stage *Stage, item Item) ([]Item, error) {
	switch stage.Type {
	case StageFetch:
		return p.processFetchStage(stage, item)
	case StageExtract:
		return p.processExtractStage(stage, item)
	case StageFilter:
		return p.processFilterStage(stage, item)
	case StageTransform:
		return p.processTransformStage(stage, item)
	case StageStore:
		return p.processStoreStage(stage, item)
	case StageMedia:
		return p.processMediaStage(stage, item)
	case StageFollow:
		return p.processFollowStage(stage, item)
	case StagePagination:
		return p.processPaginationStage(stage, item)
	default:
		return nil, fmt.Errorf("unknown stage type: %s", stage.Type)
	}
}

// UPDATESTAGEMETRICS UPDATES THE METRICS FOR A STAGE
func (p *Pipeline) updateStageMetrics(stageID string, duration time.Duration, success bool) {
	metrics, exists := p.stageMetrics[stageID]
	if !exists {
		return
	}

	if success {
		metrics.Processed++
		metrics.LastSuccess = time.Now()
		metrics.Times = append(metrics.Times, duration)
		metrics.TotalTime += duration

		// KEEP A ROLLING WINDOW OF THE LAST 100 TIMES
		if len(metrics.Times) > 100 {
			removed := metrics.Times[0]
			metrics.Times = metrics.Times[1:]
			metrics.TotalTime -= removed
		}

		// CALCULATE AVERAGE
		if len(metrics.Times) > 0 {
			metrics.AverageTime = metrics.TotalTime / time.Duration(len(metrics.Times))
		}
	} else {
		metrics.Errors++
		metrics.LastError = time.Now()
	}
}

// PROCESSFETCHSTAGE HANDLES FETCHING CONTENT FROM A URL
func (p *Pipeline) processFetchStage(stage *Stage, item Item) ([]Item, error) {
	// GET CONFIG VALUES
	url := item.URL
	if configURL, ok := stage.Config["url"].(string); ok && url == "" {
		url = configURL
	}

	timeout := 30 * time.Second
	if configTimeout, ok := stage.Config["timeout"].(float64); ok {
		timeout = time.Duration(configTimeout) * time.Second
	}

	userAgent := config.GetRandomUserAgent()
	if configUserAgent, ok := stage.Config["userAgent"].(string); ok && configUserAgent != "" {
		userAgent = configUserAgent
	}

	// CREATE REQUEST CONTEXT WITH TIMEOUT
	ctx, cancel := context.WithTimeout(p.jobCtx, timeout)
	defer cancel()

	// FETCH THE PAGE
	content, err := FetchWithContext(ctx, url, userAgent)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	// CREATE RESULT ITEM
	resultItem := Item{
		ID:       uuid.New().String(),
		URL:      url,
		Content:  content,
		ParentID: item.ID,
		Depth:    item.Depth,
		Data:     make(map[string]any),
		Metadata: make(map[string]string),
	}

	// COPY PARENT METADATA
	for k, v := range item.Metadata {
		resultItem.Metadata[k] = v
	}

	// ADD FETCH-SPECIFIC METADATA
	resultItem.Metadata["fetchTime"] = time.Now().Format(time.RFC3339)
	resultItem.Metadata["contentLength"] = fmt.Sprintf("%d", len(content))

	return []Item{resultItem}, nil
}

// PROCESSEXTRACTSTAGE HANDLES EXTRACTING DATA FROM PAGE CONTENT
func (p *Pipeline) processExtractStage(stage *Stage, item Item) ([]Item, error) {
	if item.Content == "" {
		return nil, fmt.Errorf("no content to extract from")
	}

	// GET CONFIG VALUES
	selector, _ := stage.Config["selector"].(string)
	if selector == "" {
		return nil, fmt.Errorf("no selector specified")
	}

	attribute, _ := stage.Config["attribute"].(string)
	if attribute == "" {
		attribute = "text" // DEFAULT TO TEXT CONTENT
	}

	selectorType, _ := stage.Config["type"].(string)
	if selectorType == "" {
		selectorType = "css" // DEFAULT TO CSS
	}

	isOptional, _ := stage.Config["isOptional"].(bool)

	// EXTRACT DATA USING THE SELECTOR
	var results []string
	var err error

	if selectorType == "css" {
		results, err = ExtractWithCSS(item.Content, selector, attribute)
	} else if selectorType == "xpath" {
		results, err = ExtractWithXPath(item.Content, selector, attribute)
	} else {
		return nil, fmt.Errorf("unknown selector type: %s", selectorType)
	}

	if err != nil {
		if isOptional {
			// JUST LOG THE ERROR AND CONTINUE WITH EMPTY RESULTS
			log.Printf("Optional selector failed: %v", err)
			results = []string{}
		} else {
			return nil, fmt.Errorf("extraction error: %w", err)
		}
	}

	if len(results) == 0 && !isOptional {
		return nil, fmt.Errorf("no results found for selector: %s", selector)
	}

	// CREATE RESULT ITEMS
	var resultItems []Item

	for _, result := range results {
		resultItem := Item{
			ID:       uuid.New().String(),
			ParentID: item.ID,
			URL:      item.URL,
			Content:  result,
			Depth:    item.Depth,
			Data:     make(map[string]any),
			Metadata: make(map[string]string),
		}

		// COPY PARENT METADATA
		for k, v := range item.Metadata {
			resultItem.Metadata[k] = v
		}

		// ADD EXTRACTION-SPECIFIC METADATA
		resultItem.Metadata["extractorSelector"] = selector
		resultItem.Metadata["extractorAttribute"] = attribute
		resultItem.Metadata["extractorType"] = selectorType

		resultItems = append(resultItems, resultItem)
	}

	return resultItems, nil
}

// PROCESSFILTER HANDLES FILTERING ITEMS BASED ON CONDITIONS
func (p *Pipeline) processFilterStage(stage *Stage, item Item) ([]Item, error) {
	// GET CONFIG VALUES
	pattern, _ := stage.Config["pattern"].(string)
	if pattern == "" {
		return []Item{item}, nil // NO FILTER, PASS THROUGH
	}

	includeMatches, _ := stage.Config["includeMatches"].(bool)
	targetField, _ := stage.Config["field"].(string)
	if targetField == "" {
		targetField = "content" // DEFAULT TO CONTENT
	}

	// GET THE VALUE TO FILTER ON
	var value string

	switch targetField {
	case "content":
		value = item.Content
	case "url":
		value = item.URL
	default:
		// CHECK IF IT'S IN METADATA
		if v, ok := item.Metadata[targetField]; ok {
			value = v
		} else if v, ok := item.Data[targetField]; ok {
			// CHECK IF IT'S IN DATA
			if strValue, ok := v.(string); ok {
				value = strValue
			} else {
				// TRY TO CONVERT TO STRING
				value = fmt.Sprintf("%v", v)
			}
		}
	}

	// COMPILE AND MATCH REGEX
	matched, err := MatchPattern(value, pattern)
	if err != nil {
		return nil, fmt.Errorf("filter error: %w", err)
	}

	// DETERMINE IF ITEM PASSES FILTER
	if matched == includeMatches {
		return []Item{item}, nil
	}

	// ITEM FILTERED OUT
	return []Item{}, nil
}

// PROCESSTRANSFORMSTAGE HANDLES TRANSFORMING ITEM DATA
func (p *Pipeline) processTransformStage(stage *Stage, item Item) ([]Item, error) {
	// GET TRANSFORM TYPE
	transformType, _ := stage.Config["transformType"].(string)
	if transformType == "" {
		return nil, fmt.Errorf("no transform type specified")
	}

	// CLONE THE ITEM FOR TRANSFORMATION
	resultItem := Item{
		ID:       uuid.New().String(),
		ParentID: item.ID,
		URL:      item.URL,
		Content:  item.Content,
		Depth:    item.Depth,
		Data:     make(map[string]any),
		Metadata: make(map[string]string),
	}

	// COPY DATA AND METADATA
	for k, v := range item.Data {
		resultItem.Data[k] = v
	}

	for k, v := range item.Metadata {
		resultItem.Metadata[k] = v
	}

	// APPLY TRANSFORMATION
	switch transformType {
	case "json":
		// PARSE CONTENT AS JSON
		var jsonData map[string]any
		if err := json.Unmarshal([]byte(item.Content), &jsonData); err != nil {
			return nil, fmt.Errorf("JSON transform error: %w", err)
		}

		// STORE PARSED DATA
		for k, v := range jsonData {
			resultItem.Data[k] = v
		}

	case "trim":
		// TRIM WHITESPACE
		resultItem.Content = strings.TrimSpace(item.Content)

	case "replace":
		// REPLACE TEXT
		pattern, _ := stage.Config["pattern"].(string)
		replacement, _ := stage.Config["replacement"].(string)

		if pattern != "" {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("replace pattern error: %w", err)
			}

			resultItem.Content = regex.ReplaceAllString(item.Content, replacement)
		}

	case "extract":
		// EXTRACT PART OF CONTENT WITH REGEX
		pattern, _ := stage.Config["pattern"].(string)
		if pattern == "" {
			return nil, fmt.Errorf("no extract pattern specified")
		}

		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("extract pattern error: %w", err)
		}

		matches := regex.FindStringSubmatch(item.Content)
		if len(matches) > 1 {
			// USE FIRST CAPTURING GROUP
			resultItem.Content = matches[1]
		} else if len(matches) > 0 {
			// USE ENTIRE MATCH
			resultItem.Content = matches[0]
		} else {
			return nil, fmt.Errorf("extract pattern didn't match")
		}

	case "normalize-url":
		// NORMALIZE URL
		normalizedURL, err := NormalizeURL(item.Content)
		if err != nil {
			return nil, fmt.Errorf("URL normalization error: %w", err)
		}

		resultItem.Content = normalizedURL

		// IF THIS IS A URL FIELD, ALSO UPDATE THE URL
		if targetField, ok := stage.Config["field"].(string); ok && targetField == "url" {
			resultItem.URL = normalizedURL
		}

	default:
		return nil, fmt.Errorf("unknown transform type: %s", transformType)
	}

	return []Item{resultItem}, nil
}

// PROCESSSTOREASTAGE HANDLES STORING CONTENT TO DISK
func (p *Pipeline) processStoreStage(stage *Stage, item Item) ([]Item, error) {
	// GET CONFIG VALUES
	basePath, _ := stage.Config["path"].(string)
	if basePath == "" {
		basePath = fmt.Sprintf("%s/%s", config.AppConfig.StoragePath, p.ID)
	}

	generateThumbnail, _ := stage.Config["thumbnail"].(bool)

	// ENSURE DIRECTORY EXISTS
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// GENERATE FILENAME
	filename := item.ID
	if filenameTemplate, ok := stage.Config["filenameTemplate"].(string); ok && filenameTemplate != "" {
		// TEMPLATE CAN USE PLACEHOLDERS LIKE {{id}}, {{timestamp}}, ETC.
		filename = ApplyTemplate(filenameTemplate, item)
	}

	// DETERMINE FILE EXTENSION
	extension := ""
	if contentType, ok := item.Metadata["contentType"]; ok {
		extension = GetExtensionFromContentType(contentType)
	} else {
		// GUESS FROM URL OR CONTENT
		extension = GuessExtension(item.URL, item.Content)
	}

	if extension == "" {
		extension = ".bin" // DEFAULT
	}

	// FULL FILE PATH
	filePath := filepath.Join(basePath, filename+extension)

	// STORE THE FILE
	if err := os.WriteFile(filePath, []byte(item.Content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// UPDATE ITEM WITH STORAGE INFORMATION
	item.Data["stored"] = true
	item.Data["filePath"] = filePath
	item.Data["relativePath"] = filepath.Join(filepath.Base(basePath), filename+extension)

	// GENERATE THUMBNAIL IF NEEDED
	if generateThumbnail {
		thumbPath, err := GenerateThumbnail(filePath, item.Metadata["contentType"])
		if err != nil {
			log.Printf("Warning: Failed to generate thumbnail: %v", err)
		} else {
			item.Data["thumbnailPath"] = thumbPath
		}
	}

	// CREATE ASSET ENTRY IF THIS IS AN ASSET
	if asset, ok := stage.Config["asAsset"].(bool); ok && asset {
		// CREATE ASSET RECORD
		assetID := uuid.New().String()
		asset := models.Asset{
			ID:            assetID,
			URL:           item.URL,
			Title:         item.Metadata["title"],
			Description:   item.Metadata["description"],
			Author:        item.Metadata["author"],
			Date:          item.Metadata["date"],
			Type:          DetermineAssetType(item),
			Size:          int64(len(item.Content)),
			LocalPath:     item.Data["relativePath"].(string),
			ThumbnailPath: item.Data["thumbnailPath"].(string),
			Metadata:      item.Metadata,
			Downloaded:    true,
		}

		// SAVE ASSET TO STORAGE
		if err := storage.AddAsset(p.ID, &asset); err != nil {
			log.Printf("Warning: Failed to save asset record: %v", err)
		}

		// ADD ASSET ID TO ITEM
		item.Data["assetId"] = assetID
	}

	return []Item{item}, nil
}

// PROCESSMEDIA HANDLES SPECIAL MEDIA EXTRACTION FROM PAGES
func (p *Pipeline) processMediaStage(stage *Stage, item Item) ([]Item, error) {
	// THIS STAGE HANDLES VIDEO/MEDIA EXTRACTION WITH BROWSER AUTOMATION

	// GET CONFIG VALUES
	headless, _ := stage.Config["headless"].(bool)

	// EXTRACT MEDIA URLS USING BROWSER AUTOMATION
	urls, err := ExtractMediaURLs(p.jobCtx, item.URL, headless)
	if err != nil {
		return nil, fmt.Errorf("media extraction error: %w", err)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no media found at URL: %s", item.URL)
	}

	// CREATE RESULT ITEMS FOR EACH MEDIA URL
	var resultItems []Item

	for _, mediaURL := range urls {
		mediaItem := Item{
			ID:       uuid.New().String(),
			URL:      mediaURL,
			ParentID: item.ID,
			Depth:    item.Depth,
			Content:  "", // WILL BE FILLED BY FETCH
			Data:     make(map[string]any),
			Metadata: make(map[string]string),
		}

		// COPY PARENT METADATA
		for k, v := range item.Metadata {
			mediaItem.Metadata[k] = v
		}

		// ADD MEDIA-SPECIFIC METADATA
		mediaItem.Metadata["sourceURL"] = item.URL
		mediaItem.Metadata["mediaType"] = GuessMediaType(mediaURL)

		resultItems = append(resultItems, mediaItem)
	}

	return resultItems, nil
}

// PROCESSFOLLOW HANDLES FOLLOWING LINKS TO NEW PAGES
func (p *Pipeline) processFollowStage(stage *Stage, item Item) ([]Item, error) {
	// THIS STAGE TAKES URLS (TYPICALLY FROM EXTRACT) AND CREATES ITEMS TO CONTINUE CRAWLING

	// CHECK IF URL IS VALID
	if item.Content == "" {
		return nil, fmt.Errorf("no URL to follow")
	}

	// GET CONFIG VALUES
	maxDepth, _ := stage.Config["maxDepth"].(float64)
	includeRegex, _ := stage.Config["includeRegex"].(string)
	excludeRegex, _ := stage.Config["excludeRegex"].(string)

	// CHECK DEPTH LIMIT
	if maxDepth > 0 && item.Depth >= int(maxDepth) {
		// REACHED MAX DEPTH, DON'T FOLLOW
		return []Item{}, nil
	}

	// NORMALIZE URL
	url := strings.TrimSpace(item.Content)

	// APPLY INCLUDE/EXCLUDE PATTERNS
	if includeRegex != "" {
		matched, err := MatchPattern(url, includeRegex)
		if err != nil {
			log.Printf("Warning: Invalid include pattern: %v", err)
		} else if !matched {
			// URL DOESN'T MATCH INCLUDE PATTERN
			return []Item{}, nil
		}
	}

	if excludeRegex != "" {
		matched, err := MatchPattern(url, excludeRegex)
		if err != nil {
			log.Printf("Warning: Invalid exclude pattern: %v", err)
		} else if matched {
			// URL MATCHES EXCLUDE PATTERN
			return []Item{}, nil
		}
	}

	// CREATE FOLLOW ITEM
	followItem := Item{
		ID:       uuid.New().String(),
		URL:      url,
		ParentID: item.ID,
		Depth:    item.Depth + 1,
		Data:     make(map[string]any),
		Metadata: make(map[string]string),
	}

	// COPY PARENT METADATA
	for k, v := range item.Metadata {
		followItem.Metadata[k] = v
	}

	// ADD FOLLOW-SPECIFIC METADATA
	followItem.Metadata["parentURL"] = item.URL
	followItem.Metadata["depth"] = fmt.Sprintf("%d", followItem.Depth)

	return []Item{followItem}, nil
}

// PROCESSPAGINATION HANDLES PAGINATION LINKS
func (p *Pipeline) processPaginationStage(stage *Stage, item Item) ([]Item, error) {
	if item.Content == "" {
		return nil, fmt.Errorf("no content to extract pagination from")
	}

	// GET CONFIG VALUES
	selector, _ := stage.Config["selector"].(string)
	if selector == "" {
		return []Item{}, nil // NO PAGINATION SELECTOR, SKIP
	}

	selectorType, _ := stage.Config["type"].(string)
	if selectorType == "" {
		selectorType = "css" // DEFAULT TO CSS
	}

	attribute, _ := stage.Config["attribute"].(string)
	if attribute == "" {
		attribute = "href" // DEFAULT TO HREF
	}

	maxPages, _ := stage.Config["maxPages"].(float64)

	// CHECK IF WE'VE REACHED MAX PAGES
	currentPage, ok := item.Data["pageNum"].(int)
	if !ok {
		currentPage = 1
	}

	if maxPages > 0 && currentPage >= int(maxPages) {
		// REACHED MAX PAGES, DON'T PAGINATE FURTHER
		return []Item{}, nil
	}

	// EXTRACT PAGINATION LINK
	var nextPageURLs []string
	var err error

	if selectorType == "css" {
		nextPageURLs, err = ExtractWithCSS(item.Content, selector, attribute)
	} else if selectorType == "xpath" {
		nextPageURLs, err = ExtractWithXPath(item.Content, selector, attribute)
	} else {
		return nil, fmt.Errorf("unknown selector type for pagination: %s", selectorType)
	}

	if err != nil || len(nextPageURLs) == 0 {
		// NO NEXT PAGE
		return []Item{}, nil
	}

	// TAKE FIRST PAGINATION LINK
	nextPageURL := nextPageURLs[0]

	// MAKE ABSOLUTE URL IF NEEDED
	if !IsAbsoluteURL(nextPageURL) {
		nextPageURL = MakeAbsoluteURL(item.URL, nextPageURL)
	}

	// CHECK IF NEXT PAGE IS DIFFERENT FROM CURRENT
	if nextPageURL == item.URL {
		// SAME URL, DON'T CREATE LOOP
		return []Item{}, nil
	}

	// CREATE PAGINATION ITEM
	nextPageItem := Item{
		ID:       uuid.New().String(),
		URL:      nextPageURL,
		ParentID: item.ID,
		Depth:    item.Depth, // KEEP SAME DEPTH FOR PAGINATION
		Data: map[string]any{
			"pageNum": currentPage + 1,
		},
		Metadata: make(map[string]string),
	}

	// COPY PARENT METADATA
	for k, v := range item.Metadata {
		nextPageItem.Metadata[k] = v
	}

	// ADD PAGINATION-SPECIFIC METADATA
	nextPageItem.Metadata["isPagination"] = "true"
	nextPageItem.Metadata["previousPage"] = item.URL
	nextPageItem.Metadata["pageNumber"] = fmt.Sprintf("%d", currentPage+1)

	return []Item{nextPageItem}, nil
}

// SHUTDOWN GRACEFULLY CLEANS UP THE PIPELINE
func (p *Pipeline) Shutdown() {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}

	log.Printf("Pipeline shutdown initiated")

	// WAIT FOR EVERYTHING TO FINISH WITH A TIMEOUT
	done := make(chan struct{})
	go func() {
		p.globalWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("Pipeline shutdown completed gracefully")
	case <-time.After(30 * time.Second):
		log.Printf("Pipeline shutdown timed out")
	}
}

// CONTEXTUALERROR CAPTURES THE FULL CONTEXT OF AN ERROR
type ContextualError struct {
	URL         string
	Stage       string
	StageID     string
	RawError    error
	HTML        string // TRUNCATED HTML FOR CONTEXT
	Screenshot  []byte // OPTIONAL SCREENSHOT
	Timestamp   time.Time
	Recoverable bool
}

func (e *ContextualError) Error() string {
	return fmt.Sprintf("Error in stage %s (%s) at URL %s: %v",
		e.Stage, e.StageID, e.URL, e.RawError)
}

// LOG WRITES DETAILED ERROR INFORMATION
func (e *ContextualError) Log() {
	// CREATE LOG ENTRY
	entry := map[string]any{
		"url":         e.URL,
		"stage":       e.Stage,
		"stage_id":    e.StageID,
		"error":       e.RawError.Error(),
		"timestamp":   e.Timestamp.Format(time.RFC3339),
		"recoverable": e.Recoverable,
	}

	// LOG TO CONSOLE
	log.Printf("Scraping error: %v", e.RawError)

	// SAVE HTML SNIPPET FOR CONTEXT
	if e.HTML != "" {
		htmlSnippet := TruncateString(e.HTML, 500)
		entry["html_snippet"] = htmlSnippet
	}

	// SAVE SCREENSHOT IF AVAILABLE
	if len(e.Screenshot) > 0 {
		errorID := uuid.New().String()
		screenshotPath := filepath.Join(config.AppConfig.ErrorsPath, errorID+".png")

		// ENSURE DIRECTORY EXISTS
		os.MkdirAll(config.AppConfig.ErrorsPath, 0755)

		if err := os.WriteFile(screenshotPath, e.Screenshot, 0644); err == nil {
			entry["screenshot_path"] = screenshotPath
		}
	}

	// WRITE TO STRUCTURED LOG FILE
	errorFilePath := filepath.Join(config.AppConfig.LogsPath, "scraper_errors.jsonl")

	// ENSURE DIRECTORY EXISTS
	os.MkdirAll(config.AppConfig.LogsPath, 0755)

	// APPEND TO ERROR LOG
	f, err := os.OpenFile(errorFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()

		data, _ := json.Marshal(entry)
		f.Write(data)
		f.Write([]byte("\n"))
	}
}

// UTILITY FUNCTIONS

// TRUNCATESTRING TRUNCATES A STRING TO THE SPECIFIED LENGTH
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// MATCHPATTERN TESTS IF A STRING MATCHES A REGEX PATTERN
func MatchPattern(s, pattern string) (bool, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return regex.MatchString(s), nil
}

// NORMALIZEURL NORMALIZES A URL
func NormalizeURL(rawURL string) (string, error) {
	// PARSE URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// ENSURE SCHEME
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}

	// REMOVE FRAGMENTS
	parsedURL.Fragment = ""

	// REMOVE EMPTY QUERY PARAMETERS
	q := parsedURL.Query()
	for k, v := range q {
		if len(v) == 0 || (len(v) == 1 && v[0] == "") {
			q.Del(k)
		}
	}
	parsedURL.RawQuery = q.Encode()

	return parsedURL.String(), nil
}

// ISEXISTINGFILE CHECKS IF A FILE EXISTS AT THE GIVEN PATH
func IsExistingFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ISABSOLUTEURL CHECKS IF A URL IS ABSOLUTE
func IsAbsoluteURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
}

// MAKEABSOLUTEURL CONVERTS A RELATIVE URL TO ABSOLUTE
func MakeAbsoluteURL(baseURL, relativeURL string) string {
	// PARSE BASE URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL
	}

	// PARSE RELATIVE URL
	rel, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL
	}

	// RESOLVE RELATIVE TO BASE
	absoluteURL := base.ResolveReference(rel)

	return absoluteURL.String()
}

// GETEXTENSIONFROMCONTENTTYPE GETS FILE EXTENSION FROM CONTENT TYPE
func GetExtensionFromContentType(contentType string) string {
	switch {
	case strings.Contains(contentType, "image/jpeg"):
		return ".jpg"
	case strings.Contains(contentType, "image/png"):
		return ".png"
	case strings.Contains(contentType, "image/gif"):
		return ".gif"
	case strings.Contains(contentType, "image/webp"):
		return ".webp"
	case strings.Contains(contentType, "image/svg+xml"):
		return ".svg"
	case strings.Contains(contentType, "video/mp4"):
		return ".mp4"
	case strings.Contains(contentType, "video/webm"):
		return ".webm"
	case strings.Contains(contentType, "audio/mpeg"):
		return ".mp3"
	case strings.Contains(contentType, "audio/wav"):
		return ".wav"
	case strings.Contains(contentType, "application/pdf"):
		return ".pdf"
	case strings.Contains(contentType, "application/json"):
		return ".json"
	case strings.Contains(contentType, "text/html"):
		return ".html"
	case strings.Contains(contentType, "text/plain"):
		return ".txt"
	case strings.Contains(contentType, "text/css"):
		return ".css"
	case strings.Contains(contentType, "text/javascript"), strings.Contains(contentType, "application/javascript"):
		return ".js"
	default:
		return ""
	}
}

// GUESSEXTENSION GUESSES FILE EXTENSION FROM URL OR CONTENT
func GuessExtension(url, content string) string {
	// FIRST TRY URL
	if strings.Contains(url, ".jpg") || strings.Contains(url, ".jpeg") {
		return ".jpg"
	} else if strings.Contains(url, ".png") {
		return ".png"
	} else if strings.Contains(url, ".gif") {
		return ".gif"
	} else if strings.Contains(url, ".webp") {
		return ".webp"
	} else if strings.Contains(url, ".mp4") {
		return ".mp4"
	} else if strings.Contains(url, ".webm") {
		return ".webm"
	} else if strings.Contains(url, ".mp3") {
		return ".mp3"
	} else if strings.Contains(url, ".pdf") {
		return ".pdf"
	}

	// GUESS FROM CONTENT
	if len(content) > 5 {
		// CHECK FOR COMMON FILE SIGNATURES
		switch {
		case strings.HasPrefix(content, "\xFF\xD8\xFF"):
			return ".jpg"
		case strings.HasPrefix(content, "\x89PNG\r\n\x1A\n"):
			return ".png"
		case strings.HasPrefix(content, "GIF87a") || strings.HasPrefix(content, "GIF89a"):
			return ".gif"
		case strings.HasPrefix(content, "\x1A\x45\xDF\xA3"):
			return ".webm"
		case strings.HasPrefix(content, "%PDF"):
			return ".pdf"
		case strings.HasPrefix(content, "{") || strings.HasPrefix(content, "["):
			// LIKELY JSON
			return ".json"
		case strings.HasPrefix(content, "<!DOCTYPE html") || strings.HasPrefix(content, "<html"):
			return ".html"
		default:
			// DEFAULT FOR TEXT CONTENT
			return ".txt"
		}
	}

	return ".bin" // DEFAULT BINARY FORMAT
}

// DETERMINEASSETTYPE DETERMINES ASSET TYPE FROM ITEM
func DetermineAssetType(item Item) string {
	// CHECK METADATA FIRST
	if contentType, ok := item.Metadata["contentType"]; ok {
		if strings.Contains(contentType, "image/") {
			return "image"
		} else if strings.Contains(contentType, "video/") {
			return "video"
		} else if strings.Contains(contentType, "audio/") {
			return "audio"
		} else if strings.Contains(contentType, "application/pdf") {
			return "document"
		} else if strings.Contains(contentType, "text/") || strings.Contains(contentType, "application/json") {
			return "document"
		}
	}

	// CHECK URL PATTERNS
	url := item.URL
	if strings.Contains(url, ".jpg") || strings.Contains(url, ".jpeg") ||
		strings.Contains(url, ".png") || strings.Contains(url, ".gif") ||
		strings.Contains(url, ".webp") || strings.Contains(url, ".svg") {
		return "image"
	} else if strings.Contains(url, ".mp4") || strings.Contains(url, ".webm") ||
		strings.Contains(url, ".avi") || strings.Contains(url, ".mov") {
		return "video"
	} else if strings.Contains(url, ".mp3") || strings.Contains(url, ".wav") ||
		strings.Contains(url, ".ogg") || strings.Contains(url, ".flac") {
		return "audio"
	} else if strings.Contains(url, ".pdf") || strings.Contains(url, ".doc") ||
		strings.Contains(url, ".txt") || strings.Contains(url, ".json") {
		return "document"
	}

	// CONTENT INSPECTION
	content := item.Content
	if strings.HasPrefix(content, "\xFF\xD8\xFF") || strings.HasPrefix(content, "\x89PNG\r\n\x1A\n") {
		return "image"
	} else if strings.HasPrefix(content, "%PDF") {
		return "document"
	} else if len(content) > 100 && (strings.Contains(content[:100], "<!DOCTYPE html") || strings.Contains(content[:100], "<html")) {
		return "document"
	}

	// DEFAULT TO BINARY
	return "binary"
}

// APPLYTEMPLATE APPLIES A TEMPLATE STRING WITH ITEM DATA
func ApplyTemplate(template string, item Item) string {
	result := template

	// REPLACE {{id}} WITH ITEM ID
	result = strings.ReplaceAll(result, "{{id}}", item.ID)

	// REPLACE {{timestamp}} WITH CURRENT TIME
	result = strings.ReplaceAll(result, "{{timestamp}}", time.Now().Format("20060102-150405"))

	// REPLACE {{depth}} WITH ITEM DEPTH
	result = strings.ReplaceAll(result, "{{depth}}", fmt.Sprintf("%d", item.Depth))

	// REPLACE METADATA PLACEHOLDERS
	for k, v := range item.Metadata {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}

	// REPLACE DATA PLACEHOLDERS
	for k, v := range item.Data {
		if strValue, ok := v.(string); ok {
			result = strings.ReplaceAll(result, "{{"+k+"}}", strValue)
		} else {
			result = strings.ReplaceAll(result, "{{"+k+"}}", fmt.Sprintf("%v", v))
		}
	}

	// SANITIZE RESULT FOR FILENAME
	return SanitizeFilename(result)
}

// SANITIZEFILENAME REMOVES INVALID CHARACTERS FROM FILENAMES
func SanitizeFilename(filename string) string {
	// REPLACE INVALID CHARACTERS WITH UNDERSCORE
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	return result
}

// GUESSMEDIATYPE GUESSES MEDIA TYPE FROM URL
func GuessMediaType(url string) string {
	url = strings.ToLower(url)

	if strings.Contains(url, ".mp4") || strings.Contains(url, ".webm") ||
		strings.Contains(url, ".avi") || strings.Contains(url, ".mov") ||
		strings.Contains(url, "video") {
		return "video"
	} else if strings.Contains(url, ".mp3") || strings.Contains(url, ".wav") ||
		strings.Contains(url, ".ogg") || strings.Contains(url, ".flac") ||
		strings.Contains(url, "audio") {
		return "audio"
	} else if strings.Contains(url, ".jpg") || strings.Contains(url, ".jpeg") ||
		strings.Contains(url, ".png") || strings.Contains(url, ".gif") ||
		strings.Contains(url, ".webp") || strings.Contains(url, "image") {
		return "image"
	} else if strings.Contains(url, ".m3u8") || strings.Contains(url, ".mpd") ||
		strings.Contains(url, "playlist") || strings.Contains(url, "manifest") {
		return "playlist"
	} else if strings.Contains(url, ".ts") || strings.Contains(url, "segment") ||
		strings.Contains(url, "chunk") {
		return "segment"
	}

	return "unknown"
}

// GENERATETHUMBNAIL GENERATES A THUMBNAIL FOR AN ASSET
func GenerateThumbnail(filePath, contentType string) (string, error) {
	// ENSURE THUMBNAILS DIRECTORY EXISTS
	thumbDir := filepath.Join(config.AppConfig.ThumbnailsPath, filepath.Dir(filePath))
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// GENERATE THUMBNAIL FILENAME
	thumbName := filepath.Base(filePath)
	ext := filepath.Ext(thumbName)
	thumbName = strings.TrimSuffix(thumbName, ext) + "_thumb.jpg"
	thumbPath := filepath.Join(thumbDir, thumbName)

	// CHECK FILE TYPE
	var err error

	if contentType == "" {
		// GUESS FROM FILE EXTENSION
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			err = GenerateImageThumbnail(filePath, thumbPath)
		case ".mp4", ".webm", ".avi", ".mov":
			err = GenerateVideoThumbnail(filePath, thumbPath)
		case ".pdf":
			err = GeneratePDFThumbnail(filePath, thumbPath)
		default:
			err = GenerateGenericThumbnail(filePath, thumbPath)
		}
	} else {
		// USE CONTENT TYPE
		if strings.Contains(contentType, "image/") {
			err = GenerateImageThumbnail(filePath, thumbPath)
		} else if strings.Contains(contentType, "video/") {
			err = GenerateVideoThumbnail(filePath, thumbPath)
		} else if strings.Contains(contentType, "application/pdf") {
			err = GeneratePDFThumbnail(filePath, thumbPath)
		} else {
			err = GenerateGenericThumbnail(filePath, thumbPath)
		}
	}

	if err != nil {
		return "", err
	}

	// RETURN RELATIVE PATH
	relPath, err := filepath.Rel(config.AppConfig.ThumbnailsPath, thumbPath)
	if err != nil {
		return thumbPath, nil
	}

	return relPath, nil
}

// GENERATEIMAGETHUMBNAIL GENERATES A THUMBNAIL FOR AN IMAGE
func GenerateImageThumbnail(srcPath, dstPath string) error {
	// USE FFMPEG (AVAILABLE IN ORIGINAL CODEBASE)
	cmd := exec.Command(
		"ffmpeg",
		"-i", srcPath,
		"-vf", "scale=320:-1",
		"-y",
		dstPath,
	)

	return cmd.Run()
}

// GENERATEVIDEOTHUMBNAIL GENERATES A THUMBNAIL FOR A VIDEO
func GenerateVideoThumbnail(srcPath, dstPath string) error {
	// EXTRACT FRAME USING FFMPEG
	cmd := exec.Command(
		"ffmpeg",
		"-i", srcPath,
		"-ss", "00:00:01", // TAKE FRAME AT 1 SECOND
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-y",
		dstPath,
	)

	return cmd.Run()
}

// GENERATEPDFTHUMBNAIL GENERATES A THUMBNAIL FOR A PDF
func GeneratePDFThumbnail(srcPath, dstPath string) error {
	// USE IMAGEMAGICK OR GHOSTSCRIPT IF AVAILABLE, FALLBACK TO GENERIC
	cmd := exec.Command(
		"convert",
		"-density", "150",
		"-thumbnail", "320x",
		srcPath+"[0]", // FIRST PAGE
		dstPath,
	)

	err := cmd.Run()
	if err != nil {
		// TRY GHOSTSCRIPT
		cmd = exec.Command(
			"gs",
			"-sDEVICE=jpeg",
			"-dFirstPage=1",
			"-dLastPage=1",
			"-dJPEGQ=75",
			"-dNOPAUSE",
			"-dBATCH",
			"-dSAFER",
			"-r150",
			"-sOutputFile="+dstPath,
			srcPath,
		)

		err = cmd.Run()
		if err != nil {
			// FALLBACK TO GENERIC
			return GenerateGenericThumbnail(srcPath, dstPath)
		}
	}

	return nil
}

// GENERATEGENERIC THUMBNAIL CREATES A GENERIC THUMBNAIL WITH FILE TYPE ICON
func GenerateGenericThumbnail(srcPath, dstPath string) error {
	// CREATE A SIMPLE COLORED SQUARE WITH TEXT
	ext := strings.ToLower(filepath.Ext(srcPath))

	// DETERMINE BACKGROUND COLOR BASED ON FILE TYPE
	bgColor := "gray"
	switch {
	case ext == ".pdf":
		bgColor = "red"
	case ext == ".txt", ext == ".json", ext == ".csv":
		bgColor = "green"
	case ext == ".doc", ext == ".docx":
		bgColor = "blue"
	case ext == ".xls", ext == ".xlsx":
		bgColor = "yellow"
	case ext == ".zip", ext == ".rar", ext == ".tar", ext == ".gz":
		bgColor = "purple"
	}

	// CREATE THUMBNAIL USING IMAGEMAGICK IF AVAILABLE
	cmd := exec.Command(
		"convert",
		"-size", "320x240",
		"xc:"+bgColor,
		"-gravity", "center",
		"-pointsize", "24",
		"-annotate", "0", filepath.Base(srcPath),
		dstPath,
	)

	err := cmd.Run()
	if err != nil {
		// FALLBACK TO FFMPEG SOLID COLOR
		width, height := 320, 240
		img := make([]byte, width*height*3)

		// FILL WITH COLOR
		var r, g, b byte
		switch bgColor {
		case "red":
			r, g, b = 180, 30, 30
		case "green":
			r, g, b = 30, 180, 30
		case "blue":
			r, g, b = 30, 30, 180
		case "yellow":
			r, g, b = 180, 180, 30
		case "purple":
			r, g, b = 180, 30, 180
		default:
			r, g, b = 100, 100, 100
		}

		for i := 0; i < width*height; i++ {
			img[i*3] = r
			img[i*3+1] = g
			img[i*3+2] = b
		}

		// WRITE RAW IMAGE
		rawFile := dstPath + ".raw"
		if err := os.WriteFile(rawFile, img, 0644); err != nil {
			return err
		}

		// CONVERT WITH FFMPEG
		cmd = exec.Command(
			"ffmpeg",
			"-f", "rawvideo",
			"-pixel_format", "rgb24",
			"-video_size", fmt.Sprintf("%dx%d", width, height),
			"-i", rawFile,
			"-y",
			dstPath,
		)

		err = cmd.Run()
		os.Remove(rawFile)

		if err != nil {
			// LAST RESORT - JUST WRITE A SIMPLE JPG
			return os.WriteFile(dstPath, []byte{
				0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00,
				0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB,
				0x00, 0x43, 0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07,
				0x07, 0x07, 0x09, 0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B,
				0x0B, 0x0C, 0x19, 0x12, 0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E,
				0x1D, 0x1A, 0x1C, 0x1C, 0x20, 0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C,
				0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29, 0x2C, 0x30, 0x31, 0x34, 0x34,
				0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32, 0x3C, 0x2E, 0x33, 0x34,
				0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01,
				0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00,
				0x3F, 0x00, 0x2A, 0xFF, 0xD9,
			}, 0644)
		}
	}

	return nil
}

func FetchWithContext(ctx context.Context, url, userAgent string) (string, error) {
	return FetchWithHTTP(ctx, url, userAgent)
}

// ExtractWithCSS extracts content from HTML using CSS selectors
func ExtractWithCSS(content, selector, attribute string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var results []string
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		var result string
		if attribute == "text" {
			result = s.Text()
		} else if attribute == "html" {
			html, err := s.Html()
			if err == nil {
				result = html
			}
		} else {
			result, _ = s.Attr(attribute)
		}
		if result != "" {
			results = append(results, result)
		}
	})

	return results, nil
}

// ExtractWithXPath extracts content from HTML using XPath
func ExtractWithXPath(content, xpath, attribute string) ([]string, error) {
	// This is a placeholder... idk if xpath is even worth the time.
	return ExtractWithCSS(content, xpath, attribute)
}

// ExtractMediaURLs extracts media URLs from a page
func ExtractMediaURLs(ctx context.Context, url string, headless bool) ([]string, error) {
	browser, err := GetBrowser(ctx, headless)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser: %w", err)
	}
	defer ReleaseBrowser(browser)

	tab, err := browser.GetTab(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab: %w", err)
	}

	if err := tab.Navigate(ctx, url, 30*time.Second); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	mediaURLs := []string{}
	// Run Javascript to extract URLs
	result, err := tab.ExecuteJavaScript(ctx, `
        (function() {
            const urls = [];
            // Extract video source elements
            document.querySelectorAll('video source').forEach(source => {
                if (source.src) urls.push(source.src);
            });
            // Extract video elements with direct sources
            document.querySelectorAll('video').forEach(video => {
                if (video.src) urls.push(video.src);
            });
            return urls;
        })()
    `)

	if err != nil {
		return nil, fmt.Errorf("javascript execution failed: %w", err)
	}

	// Parse result into array of strings
	if urlsArray, ok := result.([]any); ok {
		for _, u := range urlsArray {
			if urlStr, ok := u.(string); ok {
				mediaURLs = append(mediaURLs, urlStr)
			}
		}
	}

	return mediaURLs, nil
}
