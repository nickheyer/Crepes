package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"maps"

	"github.com/google/uuid"
	"github.com/nickheyer/Crepes/internal/config"
)

// ERROR LEVELS
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
)

// STRUCTURED LOG ENTRY
type LogEntry struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Source    string         `json:"source"`
	File      string         `json:"file,omitempty"`
	Line      int            `json:"line,omitempty"`
	Function  string         `json:"function,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// SCRAPER ERROR WITH CONTEXT
type ScraperError struct {
	ID            string         `json:"id"`
	Message       string         `json:"message"`
	URL           string         `json:"url,omitempty"`
	JobID         string         `json:"jobId,omitempty"`
	StageID       string         `json:"stageId,omitempty"`
	StageName     string         `json:"stageName,omitempty"`
	ItemID        string         `json:"itemId,omitempty"`
	StatusCode    int            `json:"statusCode,omitempty"`
	RawHTML       string         `json:"rawHtml,omitempty"`
	StackTrace    string         `json:"stackTrace,omitempty"`
	ScreenshotURL string         `json:"screenshotUrl,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Temporary     bool           `json:"temporary"` // TRUE IF ERROR IS RECOVERABLE
	RetryCount    int            `json:"retryCount"`
	MaxRetries    int            `json:"maxRetries"`
}

// ERROR IMPLEMENTS THE ERROR INTERFACE
func (e *ScraperError) Error() string {
	if e.URL != "" {
		return fmt.Sprintf("[%s] %s (URL: %s)", e.StageName, e.Message, e.URL)
	}
	return fmt.Sprintf("[%s] %s", e.StageName, e.Message)
}

// ISTEMPORARY INDICATES IF ERROR IS RECOVERABLE
func (e *ScraperError) IsTemporary() bool {
	return e.Temporary
}

// SHOULDRETRY INDICATES IF ERROR SHOULD BE RETRIED
func (e *ScraperError) ShouldRetry() bool {
	return e.Temporary && e.RetryCount < e.MaxRetries
}

// INCREMENTRETRY INCREMENTS THE RETRY COUNTER
func (e *ScraperError) IncrementRetry() {
	e.RetryCount++
}

// LOGGER IS A STRUCTURED LOGGER
type Logger struct {
	mu        sync.Mutex
	logFile   *os.File
	errorFile *os.File
	logPath   string
	errorPath string
	minLevel  string
	console   bool
}

// GLOBAL LOGGER INSTANCE
var defaultLogger *Logger
var loggerOnce sync.Once

// GETLOGGER RETURNS THE SINGLETON LOGGER INSTANCE
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		// CREATE DEFAULT LOGGER
		logger, err := NewLogger(config.AppConfig.LogFile, LevelInfo, true)
		if err != nil {
			log.Printf("Error creating logger: %v, using fallback", err)
			// FALLBACK TO SIMPLE LOGGER
			defaultLogger = &Logger{
				logPath:   "",
				errorPath: "",
				minLevel:  LevelInfo,
				console:   true,
			}
		} else {
			defaultLogger = logger
		}
	})

	return defaultLogger
}

// NEWLOGGER CREATES A NEW LOGGER
func NewLogger(logDir, minLevel string, console bool) (*Logger, error) {
	// ENSURE LOG DIRECTORY EXISTS
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// OPEN LOG FILES
	logPath := filepath.Join(logDir, "crepes.log")
	errorPath := filepath.Join(logDir, "errors.log")

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	errorFile, err := os.OpenFile(errorPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to open error file: %w", err)
	}

	return &Logger{
		logFile:   logFile,
		errorFile: errorFile,
		logPath:   logPath,
		errorPath: errorPath,
		minLevel:  minLevel,
		console:   console,
	}, nil
}

// CLOSE CLOSES THE LOGGER FILES
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		l.logFile.Close()
		l.logFile = nil
	}

	if l.errorFile != nil {
		l.errorFile.Close()
		l.errorFile = nil
	}
}

// LOG LOGS A MESSAGE WITH THE SPECIFIED LEVEL
func (l *Logger) Log(level, message string, data map[string]any) {
	// SKIP LOGS BELOW MINIMUM LEVEL
	if !isLevelEnabled(l.minLevel, level) {
		return
	}

	// CREATE LOG ENTRY
	entry := LogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Source:    "crepes",
		Data:      data,
	}

	// ADD CALLER INFORMATION
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.File = filepath.Base(file)
		entry.Line = line
		if fn := runtime.FuncForPC(reflect.ValueOf(l.Log).Pointer()); fn != nil {
			entry.Function = fn.Name()
		}
	}

	// SERIALIZE TO JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}

	// WRITE TO CONSOLE IF ENABLED
	if l.console {
		var levelColor string
		switch level {
		case LevelDebug:
			levelColor = "\033[36m" // CYAN
		case LevelInfo:
			levelColor = "\033[32m" // GREEN
		case LevelWarn:
			levelColor = "\033[33m" // YELLOW
		case LevelError:
			levelColor = "\033[31m" // RED
		case LevelFatal:
			levelColor = "\033[35m" // MAGENTA
		default:
			levelColor = "\033[0m" // NO COLOR
		}

		fmt.Printf("%s[%s] %s\033[0m %s\n", levelColor, level, entry.Timestamp, entry.Message)
		if len(data) > 0 {
			// PRINT DATA IN INDENTED FORMAT
			dataJSON, _ := json.MarshalIndent(data, "  ", "  ")
			fmt.Printf("  %s\n", dataJSON)
		}
	}

	// WRITE TO LOG FILE
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		l.logFile.Write(jsonData)
		l.logFile.Write([]byte("\n"))
	}

	// WRITE ERRORS TO ERROR FILE
	if level == LevelError || level == LevelFatal {
		if l.errorFile != nil {
			l.errorFile.Write(jsonData)
			l.errorFile.Write([]byte("\n"))
		}
	}
}

// DEBUG LOGS A DEBUG MESSAGE
func (l *Logger) Debug(message string, data map[string]any) {
	l.Log(LevelDebug, message, data)
}

// INFO LOGS AN INFO MESSAGE
func (l *Logger) Info(message string, data map[string]any) {
	l.Log(LevelInfo, message, data)
}

// WARN LOGS A WARNING MESSAGE
func (l *Logger) Warn(message string, data map[string]any) {
	l.Log(LevelWarn, message, data)
}

// ERROR LOGS AN ERROR MESSAGE
func (l *Logger) Error(message string, data map[string]any) {
	l.Log(LevelError, message, data)
}

// FATAL LOGS A FATAL ERROR MESSAGE
func (l *Logger) Fatal(message string, data map[string]any) {
	l.Log(LevelFatal, message, data)
}

// ISLEVELENABLED CHECKS IF A LOG LEVEL IS ENABLED
func isLevelEnabled(minLevel, level string) bool {
	levels := map[string]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
		LevelFatal: 4,
	}

	minLevelValue, minOk := levels[minLevel]
	levelValue, levelOk := levels[level]

	if !minOk || !levelOk {
		return true
	}

	return levelValue >= minLevelValue
}

// LOGERROR LOGS A SCRAPER ERROR
func (l *Logger) LogScraperError(err *ScraperError) {
	// CREATE LOG DATA
	data := map[string]any{
		"error_id":    err.ID,
		"job_id":      err.JobID,
		"stage_id":    err.StageID,
		"stage_name":  err.StageName,
		"url":         err.URL,
		"temporary":   err.Temporary,
		"retry_count": err.RetryCount,
		"max_retries": err.MaxRetries,
		"timestamp":   err.Timestamp.Format(time.RFC3339),
	}

	// ADD CUSTOM METADATA
	maps.Copy(data, err.Metadata)

	// LOG THE ERROR
	l.Error(err.Message, data)

	// STORE EXTENDED ERROR DATA FOR DEBUGGING
	if config.AppConfig.LogFile != "" {
		// CREATE ERROR DIRECTORY IF IT DOESN'T EXIST
		errorDir := filepath.Join(config.AppConfig.LogFile, "error_details", err.JobID)
		if err := os.MkdirAll(errorDir, 0755); err != nil {
			l.Error("Failed to create error directory", map[string]any{
				"error": err.Error(),
				"path":  errorDir,
			})
			return
		}

		// WRITE ERROR DETAILS TO FILE
		errorFile := filepath.Join(errorDir, fmt.Sprintf("%s.json", err.ID))
		errorData, _ := json.MarshalIndent(err, "", "  ")
		if err := os.WriteFile(errorFile, errorData, 0644); err != nil {
			l.Error("Failed to write error details", map[string]any{
				"error": err.Error(),
				"path":  errorFile,
			})
			return
		}

		// STORE HTML SNIPPET IF AVAILABLE
		if err.RawHTML != "" {
			htmlFile := filepath.Join(errorDir, fmt.Sprintf("%s.html", err.ID))
			if err := os.WriteFile(htmlFile, []byte(err.RawHTML), 0644); err != nil {
				l.Error("Failed to write error HTML", map[string]any{
					"error": err.Error(),
					"path":  htmlFile,
				})
			}
		}

		// STORE SCREENSHOT IF AVAILABLE
		if err.ScreenshotURL != "" && strings.HasPrefix(err.ScreenshotURL, "data:image/") {
			// CONVERT DATA URL TO BINARY
			parts := strings.Split(err.ScreenshotURL, ",")
			if len(parts) == 2 {
				if data, sErr := base64.StdEncoding.DecodeString(parts[1]); sErr == nil {
					screenshotFile := filepath.Join(errorDir, fmt.Sprintf("%s.png", err.ID))
					if sfErr := os.WriteFile(screenshotFile, data, 0644); sfErr != nil {
						l.Error("Failed to write error screenshot", map[string]any{
							"error": sfErr.Error(),
							"path":  screenshotFile,
						})
					}
				}
			}
		}
	}
}

// NEWSCRAPER ERROR CREATES A NEW SCRAPER ERROR
func NewScraperError(message string, url string, jobID string, stageID string, stageName string) *ScraperError {
	return &ScraperError{
		ID:         uuid.New().String(),
		Message:    message,
		URL:        url,
		JobID:      jobID,
		StageID:    stageID,
		StageName:  stageName,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]any),
		Temporary:  false,
		RetryCount: 0,
		MaxRetries: 3,
	}
}

// NEWTEMPORARYSCRAPERRERROR CREATES A NEW TEMPORARY SCRAPER ERROR
func NewTemporaryScraperError(message string, url string, jobID string, stageID string, stageName string, maxRetries int) *ScraperError {
	return &ScraperError{
		ID:         uuid.New().String(),
		Message:    message,
		URL:        url,
		JobID:      jobID,
		StageID:    stageID,
		StageName:  stageName,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]any),
		Temporary:  true,
		RetryCount: 0,
		MaxRetries: maxRetries,
	}
}

// WITHSTATUSCODE ADDS A STATUS CODE TO THE ERROR
func (e *ScraperError) WithStatusCode(statusCode int) *ScraperError {
	e.StatusCode = statusCode
	return e
}

// WITHHTML ADDS HTML CONTENT TO THE ERROR
func (e *ScraperError) WithHTML(html string) *ScraperError {
	// TRUNCATE LONG HTML
	if len(html) > 10000 {
		e.RawHTML = html[:10000] + "... [truncated]"
	} else {
		e.RawHTML = html
	}
	return e
}

// WITHSCREENSHOT ADDS A SCREENSHOT TO THE ERROR
func (e *ScraperError) WithScreenshot(screenshotURL string) *ScraperError {
	e.ScreenshotURL = screenshotURL
	return e
}

// WITHSTACKTRACE ADDS A STACK TRACE TO THE ERROR
func (e *ScraperError) WithStackTrace() *ScraperError {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	e.StackTrace = string(buf[:n])
	return e
}

// WITHMETADATA ADDS METADATA TO THE ERROR
func (e *ScraperError) WithMetadata(key string, value any) *ScraperError {
	e.Metadata[key] = value
	return e
}

// CONCURRENT ERROR GROUP - FOR HANDLING ERRORS IN CONCURRENT OPERATIONS
type ErrorGroup struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	errChan   chan error
	errOnce   sync.Once
	firstErr  error
	logger    *Logger
	errorData []ScraperError
	mu        sync.Mutex
}

// NEWERRORGROUP CREATES A NEW ERROR GROUP
func NewErrorGroup(ctx context.Context) (*ErrorGroup, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &ErrorGroup{
		ctx:       ctx,
		cancel:    cancel,
		errChan:   make(chan error, 1),
		logger:    GetLogger(),
		errorData: make([]ScraperError, 0),
	}, ctx
}

// GO RUNS A FUNCTION IN A GOROUTINE
func (g *ErrorGroup) Go(f func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := f(); err != nil {
			// LOG THE ERROR
			if scraperErr, ok := err.(*ScraperError); ok {
				g.mu.Lock()
				g.errorData = append(g.errorData, *scraperErr)
				g.mu.Unlock()

				// LOG SCRAPER ERROR
				g.logger.LogScraperError(scraperErr)

				// IF NOT TEMPORARY, CANCEL THE GROUP
				if !scraperErr.Temporary {
					g.errOnce.Do(func() {
						g.firstErr = err
						g.cancel()
					})
				}
			} else {
				// REGULAR ERROR
				g.logger.Error(err.Error(), nil)

				g.errOnce.Do(func() {
					g.firstErr = err
					g.cancel()
				})
			}
		}
	}()
}

// WAIT WAITS FOR ALL GOROUTINES TO COMPLETE
func (g *ErrorGroup) Wait() error {
	g.wg.Wait()
	g.cancel()
	return g.firstErr
}

// GETERRORS RETURNS ALL ERRORS
func (g *ErrorGroup) GetErrors() []ScraperError {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.errorData
}
