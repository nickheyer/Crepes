package scraper

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nickheyer/Crepes/internal/models"
	"github.com/nickheyer/Crepes/internal/utils"
	"github.com/playwright-community/playwright-go"
)

// COMMON ERRORS
var (
	ErrMissingRequiredInput = errors.New("MISSING REQUIRED INPUT")
	ErrInvalidSelector      = errors.New("INVALID SELECTOR")
	ErrElementNotFound      = errors.New("ELEMENT NOT FOUND")
	ErrPageNotFound         = errors.New("PAGE NOT FOUND")
	ErrBrowserNotFound      = errors.New("BROWSER NOT FOUND")
	ErrOperationFailed      = errors.New("OPERATION FAILED")
)

// HELPER FUNCTION TO GET PAGE FROM RESOURCE MANAGER
func getPage(ctx *TaskContext, pageIdInput interface{}) (playwright.Page, error) {
	var pageId string

	switch id := pageIdInput.(type) {
	case string:
		pageId = id
	case map[string]interface{}:
		// HANDLE CASE WHERE PAGE ID IS NESTED IN JSON
		if val, ok := id["pageId"].(string); ok {
			pageId = val
		} else {
			return nil, fmt.Errorf("INVALID PAGE ID FORMAT")
		}
	default:
		return nil, fmt.Errorf("INVALID PAGE ID TYPE: %T", pageIdInput)
	}

	// GET PAGE FROM RESOURCE MANAGER
	resource, exists := ctx.ResourceManager.GetResource(ctx.JobID, pageId)
	if !exists {
		return nil, ErrPageNotFound
	}

	page, ok := resource.(playwright.Page)
	if !ok {
		return nil, fmt.Errorf("RESOURCE IS NOT A PAGE")
	}

	return page, nil
}

// HELPER FUNCTION TO GET BROWSER FROM RESOURCE MANAGER
func getBrowser(ctx *TaskContext, browserIdInput interface{}) (playwright.Browser, error) {
	var browserId string

	switch id := browserIdInput.(type) {
	case string:
		browserId = id
	case map[string]interface{}:
		// HANDLE CASE WHERE BROWSER ID IS NESTED IN JSON
		if val, ok := id["browserId"].(string); ok {
			browserId = val
		} else {
			return nil, fmt.Errorf("INVALID BROWSER ID FORMAT")
		}
	default:
		return nil, fmt.Errorf("INVALID BROWSER ID TYPE: %T", browserIdInput)
	}

	// GET BROWSER FROM RESOURCE MANAGER
	resource, exists := ctx.ResourceManager.GetResource(ctx.JobID, browserId)
	if !exists {
		return nil, ErrBrowserNotFound
	}

	browser, ok := resource.(playwright.Browser)
	if !ok {
		return nil, fmt.Errorf("RESOURCE IS NOT A BROWSER")
	}

	return browser, nil
}

// HELPER FUNCTION TO FIND ELEMENT BY SELECTOR
func findElement(page playwright.Page, selector string, timeout float64) (playwright.ElementHandle, error) {
	// SET DEFAULT TIMEOUT IF NOT SPECIFIED
	if timeout <= 0 {
		timeout = 5000 // 5 SECONDS DEFAULT
	}

	// WAIT FOR SELECTOR
	err := page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrElementNotFound, err)
	}

	// GET ELEMENT
	element, err := page.QuerySelector(selector)
	if err != nil || element == nil {
		return nil, fmt.Errorf("%w: %v", ErrElementNotFound, err)
	}

	return element, nil
}

//
// BROWSER RESOURCE TASKS
//

// CREATE BROWSER TASK
type CreateBrowserTask struct{}

func (t *CreateBrowserTask) GetInputSchema() map[string]string {
	return map[string]string{
		"headless":  "boolean?", // OPTIONAL
		"userAgent": "string?",  // OPTIONAL
	}
}

func (t *CreateBrowserTask) GetOutputSchema() string {
	return "object" // RETURNS BROWSER ID
}

func (t *CreateBrowserTask) ValidateConfig(config map[string]interface{}) error {
	// NO REQUIRED FIELDS
	return nil
}

func (t *CreateBrowserTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET HEADLESS MODE FROM CONFIG (DEFAULT TRUE)
	headless := true
	if val, ok := config["headless"].(bool); ok {
		headless = val
	}

	ctx.Logger.Printf("CREATING BROWSER (HEADLESS: %v)", headless)

	// GENERATE BROWSER ID
	browserId := fmt.Sprintf("browser_%s", utils.GenerateID(""))

	// LAUNCH BROWSER WITH STEALTH MODE
	browser, err := ctx.Engine.launchBrowser(headless)
	if err != nil {
		return TaskData{}, err
	}

	// STORE BROWSER IN RESOURCE MANAGER
	ctx.ResourceManager.CreateResource(ctx.JobID, browserId, "browser", *browser)

	ctx.Logger.Printf("BROWSER CREATED WITH ID: %s", browserId)

	// RETURN BROWSER ID
	return TaskData{
		Type: "object",
		Value: map[string]interface{}{
			"browserId": browserId,
		},
	}, nil
}

// CREATE PAGE TASK
type CreatePageTask struct{}

func (t *CreatePageTask) GetInputSchema() map[string]string {
	return map[string]string{
		"browserId":   "string",   // REQUIRED
		"userAgent":   "string?",  // OPTIONAL
		"viewport":    "object?",  // OPTIONAL
		"locale":      "string?",  // OPTIONAL
		"recordVideo": "boolean?", // OPTIONAL
	}
}

func (t *CreatePageTask) GetOutputSchema() string {
	return "object" // RETURNS PAGE ID
}

func (t *CreatePageTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["browserId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *CreatePageTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET BROWSER FROM RESOURCE MANAGER
	browser, err := getBrowser(ctx, config["browserId"])
	if err != nil {
		return TaskData{}, err
	}

	ctx.Logger.Printf("CREATING PAGE FOR BROWSER")

	// PAGE OPTIONS
	pageOptions := playwright.BrowserNewPageOptions{}

	// SET USER AGENT IF PROVIDED
	if userAgent, ok := config["userAgent"].(string); ok && userAgent != "" {
		pageOptions.UserAgent = playwright.String(userAgent)
	}

	// SET VIEWPORT IF PROVIDED
	if viewport, ok := config["viewport"].(map[string]interface{}); ok {
		width, hasWidth := viewport["width"].(float64)
		height, hasHeight := viewport["height"].(float64)
		if hasWidth && hasHeight {
			pageOptions.Viewport.Height = int(height)
			pageOptions.Viewport.Width = int(width)
		}
	}

	// SET LOCALE IF PROVIDED
	if locale, ok := config["locale"].(string); ok && locale != "" {
		pageOptions.Locale = playwright.String(locale)
	}

	// SET RECORD VIDEO IF PROVIDED
	if recordVideo, ok := config["recordVideo"].(bool); ok && recordVideo {
		pageOptions.RecordVideo = &playwright.RecordVideo{
			Dir: "videos",
			Size: &playwright.Size{
				Width:  1280,
				Height: 720,
			},
		}
	}

	// CREATE PAGE
	page, err := browser.NewPage(pageOptions)
	if err != nil {
		return TaskData{}, fmt.Errorf("%w: %v", ErrPageCreation, err)
	}

	// GENERATE PAGE ID
	pageId := fmt.Sprintf("page_%s", utils.GenerateID(""))

	// STORE PAGE IN RESOURCE MANAGER
	ctx.ResourceManager.CreateResource(ctx.JobID, pageId, "page", page)

	ctx.Logger.Printf("PAGE CREATED WITH ID: %s", pageId)

	// RETURN PAGE ID
	return TaskData{
		Type: "object",
		Value: map[string]interface{}{
			"pageId": pageId,
		},
	}, nil
}

// DISPOSE BROWSER TASK
type DisposeBrowserTask struct{}

func (t *DisposeBrowserTask) GetInputSchema() map[string]string {
	return map[string]string{
		"browserId": "string", // REQUIRED
	}
}

func (t *DisposeBrowserTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *DisposeBrowserTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["browserId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *DisposeBrowserTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET BROWSER FROM RESOURCE MANAGER
	browser, err := getBrowser(ctx, config["browserId"])
	if err != nil {
		return TaskData{}, err
	}

	browserId, _ := config["browserId"].(string)
	ctx.Logger.Printf("DISPOSING BROWSER: %s", browserId)

	// CLOSE BROWSER
	err = browser.Close()
	if err != nil {
		return TaskData{}, fmt.Errorf("FAILED TO CLOSE BROWSER: %v", err)
	}

	// REMOVE FROM RESOURCE MANAGER
	ctx.ResourceManager.DeleteResource(ctx.JobID, browserId)

	ctx.Logger.Printf("BROWSER %s DISPOSED", browserId)

	return TaskData{
		Type:  "boolean",
		Value: true,
	}, nil
}

// DISPOSE PAGE TASK
type DisposePageTask struct{}

func (t *DisposePageTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId": "string", // REQUIRED
	}
}

func (t *DisposePageTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *DisposePageTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *DisposePageTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	pageId, _ := config["pageId"].(string)
	ctx.Logger.Printf("DISPOSING PAGE: %s", pageId)

	// CLOSE PAGE
	err = page.Close()
	if err != nil {
		return TaskData{}, fmt.Errorf("FAILED TO CLOSE PAGE: %v", err)
	}

	// REMOVE FROM RESOURCE MANAGER
	ctx.ResourceManager.DeleteResource(ctx.JobID, pageId)

	ctx.Logger.Printf("PAGE %s DISPOSED", pageId)

	return TaskData{
		Type:  "boolean",
		Value: true,
	}, nil
}

//
// NAVIGATION TASKS
//

// NAVIGATE TASK
type NavigateTask struct{}

func (t *NavigateTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":    "string",  // REQUIRED
		"url":       "string",  // REQUIRED
		"waitUntil": "string?", // OPTIONAL (load, domcontentloaded, networkidle)
		"timeout":   "number?", // OPTIONAL
	}
}

func (t *NavigateTask) GetOutputSchema() string {
	return "object" // RETURNS NAVIGATION RESULT (STATUS, URL)
}

func (t *NavigateTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["url"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *NavigateTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET URL TO NAVIGATE TO
	url, _ := config["url"].(string)
	ctx.Logger.Printf("NAVIGATING TO URL: %s", url)

	// SET NAVIGATION OPTIONS
	options := playwright.PageGotoOptions{}

	// SET WAIT UNTIL IF PROVIDED
	if waitUntil, ok := config["waitUntil"].(string); ok && waitUntil != "" {
		switch waitUntil {
		case "load":
			options.WaitUntil = playwright.WaitUntilStateLoad
		case "domcontentloaded":
			options.WaitUntil = playwright.WaitUntilStateDomcontentloaded
		case "networkidle":
			options.WaitUntil = playwright.WaitUntilStateNetworkidle
		}
	} else {
		// DEFAULT TO DOMCONTENTLOADED
		options.WaitUntil = playwright.WaitUntilStateDomcontentloaded
	}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// PERFORM NAVIGATION
	response, err := page.Goto(url, options)
	if err != nil {
		return TaskData{}, fmt.Errorf("NAVIGATION FAILED: %v", err)
	}

	// GET RESULT INFORMATION
	status := 0
	if response != nil {
		status = response.Status()
	}

	currentUrl := page.URL()

	ctx.Logger.Printf("NAVIGATION COMPLETE: %s (STATUS: %d)", currentUrl, status)

	// RETURN NAVIGATION RESULT
	return TaskData{
		Type: "object",
		Value: map[string]interface{}{
			"status": status,
			"url":    currentUrl,
			"ok":     status >= 200 && status < 400,
		},
	}, nil
}

// BACK TASK
type BackTask struct{}

func (t *BackTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":    "string",  // REQUIRED
		"waitUntil": "string?", // OPTIONAL
		"timeout":   "number?", // OPTIONAL
	}
}

func (t *BackTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *BackTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *BackTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	ctx.Logger.Printf("NAVIGATING BACK")

	// SET NAVIGATION OPTIONS
	options := playwright.PageGoBackOptions{}

	// SET WAIT UNTIL IF PROVIDED
	if waitUntil, ok := config["waitUntil"].(string); ok && waitUntil != "" {
		switch waitUntil {
		case "load":
			options.WaitUntil = playwright.WaitUntilStateLoad
		case "domcontentloaded":
			options.WaitUntil = playwright.WaitUntilStateDomcontentloaded
		case "networkidle":
			options.WaitUntil = playwright.WaitUntilStateNetworkidle
		}
	}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// PERFORM BACK NAVIGATION
	response, err := page.GoBack(options)
	if err != nil {
		return TaskData{}, fmt.Errorf("BACK NAVIGATION FAILED: %v", err)
	}

	success := response != nil

	ctx.Logger.Printf("BACK NAVIGATION %s", map[bool]string{true: "SUCCEEDED", false: "FAILED"}[success])

	return TaskData{
		Type:  "boolean",
		Value: success,
	}, nil
}

// FORWARD TASK
type ForwardTask struct{}

func (t *ForwardTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":    "string",  // REQUIRED
		"waitUntil": "string?", // OPTIONAL
		"timeout":   "number?", // OPTIONAL
	}
}

func (t *ForwardTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *ForwardTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ForwardTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	ctx.Logger.Printf("NAVIGATING FORWARD")

	// SET NAVIGATION OPTIONS
	options := playwright.PageGoForwardOptions{}

	// SET WAIT UNTIL IF PROVIDED
	if waitUntil, ok := config["waitUntil"].(string); ok && waitUntil != "" {
		switch waitUntil {
		case "load":
			options.WaitUntil = playwright.WaitUntilStateLoad
		case "domcontentloaded":
			options.WaitUntil = playwright.WaitUntilStateDomcontentloaded
		case "networkidle":
			options.WaitUntil = playwright.WaitUntilStateNetworkidle
		}
	}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// PERFORM FORWARD NAVIGATION
	response, err := page.GoForward(options)
	if err != nil {
		return TaskData{}, fmt.Errorf("FORWARD NAVIGATION FAILED: %v", err)
	}

	success := response != nil

	ctx.Logger.Printf("FORWARD NAVIGATION %s", map[bool]string{true: "SUCCEEDED", false: "FAILED"}[success])

	return TaskData{
		Type:  "boolean",
		Value: success,
	}, nil
}

// RELOAD TASK
type ReloadTask struct{}

func (t *ReloadTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":    "string",  // REQUIRED
		"waitUntil": "string?", // OPTIONAL
		"timeout":   "number?", // OPTIONAL
	}
}

func (t *ReloadTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *ReloadTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ReloadTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	ctx.Logger.Printf("RELOADING PAGE")

	// SET RELOAD OPTIONS
	options := playwright.PageReloadOptions{}

	// SET WAIT UNTIL IF PROVIDED
	if waitUntil, ok := config["waitUntil"].(string); ok && waitUntil != "" {
		switch waitUntil {
		case "load":
			options.WaitUntil = playwright.WaitUntilStateLoad
		case "domcontentloaded":
			options.WaitUntil = playwright.WaitUntilStateDomcontentloaded
		case "networkidle":
			options.WaitUntil = playwright.WaitUntilStateNetworkidle
		}
	}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// PERFORM RELOAD
	response, err := page.Reload(options)
	if err != nil {
		return TaskData{}, fmt.Errorf("RELOAD FAILED: %v", err)
	}

	success := response != nil

	ctx.Logger.Printf("PAGE RELOAD %s", map[bool]string{true: "SUCCEEDED", false: "FAILED"}[success])

	return TaskData{
		Type:  "boolean",
		Value: success,
	}, nil
}

// WAIT FOR LOAD TASK
type WaitForLoadTask struct{}

func (t *WaitForLoadTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":  "string",  // REQUIRED
		"state":   "string?", // OPTIONAL (load, domcontentloaded, networkidle)
		"timeout": "number?", // OPTIONAL
	}
}

func (t *WaitForLoadTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *WaitForLoadTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *WaitForLoadTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// DETERMINE WAIT STATE
	state := "load" // DEFAULT
	if stateVal, ok := config["state"].(string); ok && stateVal != "" {
		state = stateVal
	}

	ctx.Logger.Printf("WAITING FOR PAGE %s", state)

	// PLACEHOLDER DOM CONTENT LOADED FOR NOW
	waitState := playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded}

	// GET TIMEOUT
	timeout := float64(30000) // DEFAULT 30 SECONDS
	if timeoutVal, ok := config["timeout"].(float64); ok && timeoutVal > 0 {
		timeout = timeoutVal
	}

	// WAIT FOR LOAD STATE
	err = page.WaitForLoadState(waitState, playwright.PageWaitForLoadStateOptions{
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return TaskData{}, fmt.Errorf("WAIT FOR LOAD STATE FAILED: %v", err)
	}

	ctx.Logger.Printf("PAGE %s STATE REACHED", state)

	return TaskData{
		Type:  "boolean",
		Value: true,
	}, nil
}

// TAKE SCREENSHOT TASK
type TakeScreenshotTask struct{}

func (t *TakeScreenshotTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":   "string",   // REQUIRED
		"selector": "string?",  // OPTIONAL (if provided, screenshots just that element)
		"fullPage": "boolean?", // OPTIONAL
		"path":     "string?",  // OPTIONAL
		"quality":  "number?",  // OPTIONAL (0-100, for jpeg only)
		"type":     "string?",  // OPTIONAL (png, jpeg)
	}
}

func (t *TakeScreenshotTask) GetOutputSchema() string {
	return "object" // RETURNS SCREENSHOT DATA
}

func (t *TakeScreenshotTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *TakeScreenshotTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	ctx.Logger.Printf("TAKING SCREENSHOT")

	// SETUP SCREENSHOT OPTIONS
	options := playwright.PageScreenshotOptions{}

	// SET FULL PAGE IF PROVIDED
	if fullPage, ok := config["fullPage"].(bool); ok {
		options.FullPage = playwright.Bool(fullPage)
	}

	// SET TYPE IF PROVIDED
	screenshotType := "png" // DEFAULT
	if typeVal, ok := config["type"].(string); ok && (typeVal == "png" || typeVal == "jpeg") {
		screenshotType = typeVal
		screenshotGo := playwright.ScreenshotType(typeVal)
		options.Type = &screenshotGo
	}

	// SET QUALITY IF PROVIDED AND TYPE IS JPEG
	if quality, ok := config["quality"].(float64); ok && quality >= 0 && quality <= 100 && screenshotType == "jpeg" {
		options.Quality = playwright.Int(int(quality))
	}

	// SET PATH IF PROVIDED
	var screenshotPath string
	if path, ok := config["path"].(string); ok && path != "" {
		// ENSURE DIRECTORY EXISTS
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return TaskData{}, fmt.Errorf("FAILED TO CREATE DIRECTORY: %v", err)
		}

		screenshotPath = path
		options.Path = playwright.String(path)
	} else {
		// GENERATE TEMPORARY PATH FOR SCREENSHOT
		screenshotPath = filepath.Join(
			os.TempDir(),
			fmt.Sprintf("screenshot_%s.%s", utils.GenerateID(""), screenshotType),
		)
		options.Path = playwright.String(screenshotPath)
	}

	// TAKE SCREENSHOT
	var screenshotData []byte

	if selector, ok := config["selector"].(string); ok && selector != "" {
		// TAKE SCREENSHOT OF SPECIFIC ELEMENT
		ctx.Logger.Printf("TAKING SCREENSHOT OF ELEMENT: %s", selector)

		// FIND ELEMENT
		element, err := findElement(page, selector, 5000)
		if err != nil {
			return TaskData{}, err
		}

		// SCREENSHOT ELEMENT
		screenshotData, err = element.Screenshot(playwright.ElementHandleScreenshotOptions{
			Path:    options.Path,
			Type:    options.Type,
			Quality: options.Quality,
		})
	} else {
		// TAKE SCREENSHOT OF ENTIRE PAGE
		ctx.Logger.Printf("TAKING SCREENSHOT OF PAGE")
		screenshotData, err = page.Screenshot(options)
	}

	if err != nil {
		return TaskData{}, fmt.Errorf("SCREENSHOT FAILED: %v", err)
	}

	// ENCODE SCREENSHOT DATA AS BASE64
	base64Data := base64.StdEncoding.EncodeToString(screenshotData)

	ctx.Logger.Printf("SCREENSHOT SAVED TO: %s", screenshotPath)

	// RETURN SCREENSHOT DATA
	return TaskData{
		Type: "object",
		Value: map[string]interface{}{
			"path":      screenshotPath,
			"type":      screenshotType,
			"data":      base64Data,
			"size":      len(screenshotData),
			"timestamp": time.Now().Unix(),
		},
	}, nil
}

// EXECUTE SCRIPT TASK
type ExecuteScriptTask struct{}

func (t *ExecuteScriptTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId": "string", // REQUIRED
		"script": "string", // REQUIRED
		"args":   "array?", // OPTIONAL
	}
}

func (t *ExecuteScriptTask) GetOutputSchema() string {
	return "any" // RETURNS SCRIPT RESULT
}

func (t *ExecuteScriptTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["script"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ExecuteScriptTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SCRIPT TO EXECUTE
	script, _ := config["script"].(string)

	ctx.Logger.Printf("EXECUTING SCRIPT ON PAGE")

	// PREPARE ARGUMENTS
	var args []interface{}
	if argsVal, ok := config["args"].([]interface{}); ok {
		args = argsVal
	}

	// EXECUTE SCRIPT
	result, err := page.Evaluate(script, args)
	if err != nil {
		return TaskData{}, fmt.Errorf("SCRIPT EXECUTION FAILED: %v", err)
	}

	// DETERMINE RESULT TYPE
	var resultType string
	switch result.(type) {
	case string:
		resultType = "string"
	case float64, int, int64:
		resultType = "number"
	case bool:
		resultType = "boolean"
	case map[string]interface{}:
		resultType = "object"
	case []interface{}:
		resultType = "array"
	case nil:
		resultType = "null"
	default:
		resultType = "any"
	}

	ctx.Logger.Printf("SCRIPT EXECUTION COMPLETED")

	// RETURN SCRIPT RESULT
	return TaskData{
		Type:  resultType,
		Value: result,
	}, nil
}

//
// INTERACTION TASKS
//

// CLICK TASK
type ClickTask struct{}

func (t *ClickTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":     "string",   // REQUIRED
		"selector":   "string",   // REQUIRED
		"button":     "string?",  // OPTIONAL (left, right, middle)
		"clickCount": "number?",  // OPTIONAL
		"timeout":    "number?",  // OPTIONAL
		"force":      "boolean?", // OPTIONAL
	}
}

func (t *ClickTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *ClickTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["selector"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ClickTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR
	selector, _ := config["selector"].(string)

	ctx.Logger.Printf("CLICKING ELEMENT: %s", selector)

	// SETUP CLICK OPTIONS
	options := playwright.PageClickOptions{}

	// SET BUTTON IF PROVIDED
	if button, ok := config["button"].(string); ok {
		switch button {
		case "right":
			options.Button = playwright.MouseButtonRight
		case "middle":
			options.Button = playwright.MouseButtonMiddle
		default:
			options.Button = playwright.MouseButtonLeft
		}
	}

	// SET CLICK COUNT IF PROVIDED
	if clickCount, ok := config["clickCount"].(float64); ok && clickCount > 0 {
		options.ClickCount = playwright.Int(int(clickCount))
	}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// SET FORCE IF PROVIDED
	if force, ok := config["force"].(bool); ok {
		options.Force = playwright.Bool(force)
	}

	// PERFORM CLICK
	err = page.Click(selector, options)
	if err != nil {
		return TaskData{}, fmt.Errorf("CLICK FAILED: %v", err)
	}

	ctx.Logger.Printf("CLICK PERFORMED SUCCESSFULLY")

	return TaskData{
		Type:  "boolean",
		Value: true,
	}, nil
}

// TYPE TASK
type TypeTask struct{}

func (t *TypeTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":   "string",   // REQUIRED
		"selector": "string",   // REQUIRED
		"text":     "string",   // REQUIRED
		"delay":    "number?",  // OPTIONAL
		"clear":    "boolean?", // OPTIONAL (clear field before typing)
		"timeout":  "number?",  // OPTIONAL
	}
}

func (t *TypeTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *TypeTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["selector"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["text"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *TypeTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR AND TEXT
	selector, _ := config["selector"].(string)
	text, _ := config["text"].(string)

	ctx.Logger.Printf("TYPING TEXT INTO ELEMENT: %s", selector)

	// CLEAR FIELD FIRST IF REQUESTED
	if clear, ok := config["clear"].(bool); ok && clear {
		// CLICK THE FIELD FIRST
		err = page.Locator(selector).Click(playwright.LocatorClickOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("FIELD CLICK FAILED: %v", err)
		}

		// SELECT ALL TEXT
		err = page.Keyboard().Press("Control+A", playwright.KeyboardPressOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("SELECT ALL FAILED: %v", err)
		}

		// DELETE SELECTED TEXT
		err = page.Keyboard().Press("Delete", playwright.KeyboardPressOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("DELETE FAILED: %v", err)
		}
	}

	// SETUP TYPE OPTIONS
	options := playwright.PageFillOptions{}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// FILL THE FIELD
	err = page.Fill(selector, text, options)
	if err != nil {
		return TaskData{}, fmt.Errorf("TYPING FAILED: %v", err)
	}

	// APPLY DELAY BETWEEN KEYSTROKES IF NEEDED
	if delay, ok := config["delay"].(float64); ok && delay > 0 {
		// TYPE WITH DELAY
		ctx.Logger.Printf("TYPING WITH DELAY: %f MS", delay)

		// CLEAR FIELD FIRST
		err = page.Click(selector, playwright.PageClickOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("FIELD CLICK FAILED: %v", err)
		}

		err = page.Keyboard().Press("Control+A", playwright.KeyboardPressOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("SELECT ALL FAILED: %v", err)
		}

		err = page.Keyboard().Press("Delete", playwright.KeyboardPressOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("DELETE FAILED: %v", err)
		}

		// NOW TYPE WITH DELAY
		for _, char := range text {
			err = page.Type(selector, string(char), playwright.PageTypeOptions{})
			if err != nil {
				return TaskData{}, fmt.Errorf("TYPING CHAR FAILED: %v", err)
			}

			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}

	ctx.Logger.Printf("TEXT TYPED SUCCESSFULLY")

	return TaskData{
		Type:  "boolean",
		Value: true,
	}, nil
}

// SELECT TASK
type SelectTask struct{}

func (t *SelectTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":   "string",  // REQUIRED
		"selector": "string",  // REQUIRED
		"values":   "array",   // REQUIRED (array of values to select)
		"timeout":  "number?", // OPTIONAL
	}
}

func (t *SelectTask) GetOutputSchema() string {
	return "array" // RETURNS SELECTED VALUES
}

func (t *SelectTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["selector"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["values"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *SelectTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR AND VALUES
	selector, _ := config["selector"].(string)
	valuesAny, _ := config["values"].([]interface{})

	// CONVERT VALUES TO STRINGS
	values := make([]string, len(valuesAny))
	for i, v := range valuesAny {
		if str, ok := v.(string); ok {
			values[i] = str
		} else {
			// CONVERT TO STRING
			values[i] = fmt.Sprintf("%v", v)
		}
	}

	ctx.Logger.Printf("SELECTING OPTIONS IN ELEMENT: %s", selector)

	// SETUP SELECT OPTIONS
	options := playwright.LocatorSelectOptionOptions{}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// PERFORM SELECT
	selected, err := page.Locator(selector).SelectOption(playwright.SelectOptionValues{Values: &values}, options)
	if err != nil {
		return TaskData{}, fmt.Errorf("SELECT FAILED: %v", err)
	}

	ctx.Logger.Printf("SELECTION PERFORMED SUCCESSFULLY")

	// CONVERT SELECTED VALUES TO INTERFACE SLICE
	selectedValues := make([]interface{}, len(selected))
	for i, v := range selected {
		selectedValues[i] = v
	}

	return TaskData{
		Type:  "array",
		Value: selectedValues,
	}, nil
}

// HOVER TASK
type HoverTask struct{}

func (t *HoverTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":   "string",  // REQUIRED
		"selector": "string",  // REQUIRED
		"timeout":  "number?", // OPTIONAL
		"position": "object?", // OPTIONAL (x, y coordinates)
	}
}

func (t *HoverTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *HoverTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["selector"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *HoverTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR
	selector, _ := config["selector"].(string)

	ctx.Logger.Printf("HOVERING OVER ELEMENT: %s", selector)

	// SETUP HOVER OPTIONS
	options := playwright.PageHoverOptions{}

	// SET TIMEOUT IF PROVIDED
	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		options.Timeout = playwright.Float(timeout)
	}

	// SET POSITION IF PROVIDED
	if posObj, ok := config["position"].(map[string]interface{}); ok {
		if x, xOk := posObj["x"].(float64); xOk {
			if y, yOk := posObj["y"].(float64); yOk {
				options.Position = &playwright.Position{
					X: x,
					Y: y,
				}
			}
		}
	}

	// PERFORM HOVER
	err = page.Hover(selector, options)
	if err != nil {
		return TaskData{}, fmt.Errorf("HOVER FAILED: %v", err)
	}

	ctx.Logger.Printf("HOVER PERFORMED SUCCESSFULLY")

	return TaskData{
		Type:  "boolean",
		Value: true,
	}, nil
}

// SCROLL TASK
type ScrollTask struct{}

func (t *ScrollTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":    "string",  // REQUIRED
		"selector":  "string?", // OPTIONAL (if not provided, scrolls the page)
		"direction": "string?", // OPTIONAL (up, down, left, right)
		"distance":  "number?", // OPTIONAL (pixels to scroll)
		"behavior":  "string?", // OPTIONAL (auto, smooth)
		"toElement": "string?", // OPTIONAL (selector of element to scroll to)
	}
}

func (t *ScrollTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *ScrollTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ScrollTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	ctx.Logger.Printf("PERFORMING SCROLL")

	// IF ELEMENT TO SCROLL TO IS PROVIDED
	if toElement, ok := config["toElement"].(string); ok && toElement != "" {
		ctx.Logger.Printf("SCROLLING TO ELEMENT: %s", toElement)

		// SCROLL ELEMENT INTO VIEW
		_, err = page.Locator(toElement).Evaluate("element => element.scrollIntoView({behavior: 'smooth', block: 'center'})", nil)
		if err != nil {
			return TaskData{}, fmt.Errorf("SCROLL TO ELEMENT FAILED: %v", err)
		}

		// WAIT A BIT FOR SCROLL TO COMPLETE
		time.Sleep(500 * time.Millisecond)

		ctx.Logger.Printf("SCROLLED TO ELEMENT SUCCESSFULLY")

		return TaskData{
			Type:  "boolean",
			Value: true,
		}, nil
	}

	// GET SCROLL PARAMETERS
	direction := "down" // DEFAULT
	if dir, ok := config["direction"].(string); ok && dir != "" {
		direction = dir
	}

	distance := float64(300) // DEFAULT
	if dist, ok := config["distance"].(float64); ok && dist > 0 {
		distance = dist
	}

	behavior := "auto" // DEFAULT
	if beh, ok := config["behavior"].(string); ok && (beh == "auto" || beh == "smooth") {
		behavior = beh
	}

	// DETERMINE X AND Y VALUES BASED ON DIRECTION
	x, y := 0.0, 0.0
	switch direction {
	case "up":
		y = -distance
	case "down":
		y = distance
	case "left":
		x = -distance
	case "right":
		x = distance
	}

	// IF SELECTOR IS PROVIDED, SCROLL THAT ELEMENT
	if selector, ok := config["selector"].(string); ok && selector != "" {
		ctx.Logger.Printf("SCROLLING ELEMENT: %s", selector)

		script := fmt.Sprintf(`(element) => {
			element.scrollBy({
				top: %f,
				left: %f,
				behavior: '%s'
			});
			return true;
		}`, y, x, behavior)

		_, err = page.Locator(selector).Evaluate(script, nil)
		if err != nil {
			return TaskData{}, fmt.Errorf("ELEMENT SCROLL FAILED: %v", err)
		}
	} else {
		// SCROLL THE PAGE
		ctx.Logger.Printf("SCROLLING PAGE")

		script := fmt.Sprintf(`() => {
			window.scrollBy({
				top: %f,
				left: %f,
				behavior: '%s'
			});
			return true;
		}`, y, x, behavior)

		_, err = page.Evaluate(script, nil)
		if err != nil {
			return TaskData{}, fmt.Errorf("PAGE SCROLL FAILED: %v", err)
		}
	}

	// WAIT A BIT FOR SCROLL TO COMPLETE
	time.Sleep(500 * time.Millisecond)

	ctx.Logger.Printf("SCROLL PERFORMED SUCCESSFULLY")

	return TaskData{
		Type:  "boolean",
		Value: true,
	}, nil
}

//
// EXTRACTION TASKS
//

// EXTRACT TEXT TASK
type ExtractTextTask struct{}

func (t *ExtractTextTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":   "string",   // REQUIRED
		"selector": "string",   // REQUIRED
		"multiple": "boolean?", // OPTIONAL (get text from multiple elements)
		"trim":     "boolean?", // OPTIONAL
		"timeout":  "number?",  // OPTIONAL
	}
}

func (t *ExtractTextTask) GetOutputSchema() string {
	return "any" // RETURNS STRING OR ARRAY OF STRINGS
}

func (t *ExtractTextTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["selector"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ExtractTextTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR
	selector, _ := config["selector"].(string)

	// CHECK IF EXTRACTING MULTIPLE
	multiple := false
	if mult, ok := config["multiple"].(bool); ok {
		multiple = mult
	}

	// CHECK IF SHOULD TRIM TEXT
	trim := true
	if trimVal, ok := config["trim"].(bool); ok {
		trim = trimVal
	}

	// GET TIMEOUT
	timeout := float64(5000) // DEFAULT 5 SECONDS
	if timeoutVal, ok := config["timeout"].(float64); ok && timeoutVal > 0 {
		timeout = timeoutVal
	}

	if multiple {
		ctx.Logger.Printf("EXTRACTING TEXT FROM MULTIPLE ELEMENTS: %s", selector)

		// WAIT FOR SELECTOR TO BE PRESENT
		err = page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(timeout),
		})

		if err != nil {
			return TaskData{}, fmt.Errorf("WAIT FOR SELECTOR FAILED: %v", err)
		}

		// EXTRACT TEXT FROM ALL MATCHING ELEMENTS
		script := `(selector, trim) => {
			const elements = Array.from(document.querySelectorAll(selector));
			return elements.map(el => trim ? el.textContent.trim() : el.textContent);
		}`

		result, err := page.Evaluate(script, []interface{}{selector, trim})
		if err != nil {
			return TaskData{}, fmt.Errorf("TEXT EXTRACTION FAILED: %v", err)
		}

		// CONVERT RESULT TO STRING ARRAY
		textArray, ok := result.([]interface{})
		if !ok {
			return TaskData{}, fmt.Errorf("UNEXPECTED RESULT TYPE: %T", result)
		}

		ctx.Logger.Printf("EXTRACTED %d TEXT ITEMS", len(textArray))

		return TaskData{
			Type:  "array",
			Value: textArray,
		}, nil
	} else {
		ctx.Logger.Printf("EXTRACTING TEXT FROM ELEMENT: %s", selector)

		// WAIT FOR SELECTOR TO BE PRESENT
		err = page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(timeout),
		})
		if err != nil {
			return TaskData{}, fmt.Errorf("WAIT FOR SELECTOR FAILED: %v", err)
		}

		// EXTRACT TEXT FROM ELEMENT
		text, err := page.Locator(selector).TextContent(playwright.LocatorTextContentOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("TEXT EXTRACTION FAILED: %v", err)
		}

		// TRIM TEXT IF NEEDED
		if trim && text != "" {
			text = strings.TrimSpace(text)
		}

		ctx.Logger.Printf("EXTRACTED TEXT: %s", text)

		return TaskData{
			Type:  "string",
			Value: text,
		}, nil
	}
}

// EXTRACT ATTRIBUTE TASK
type ExtractAttributeTask struct{}

func (t *ExtractAttributeTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":    "string",   // REQUIRED
		"selector":  "string",   // REQUIRED
		"attribute": "string",   // REQUIRED
		"multiple":  "boolean?", // OPTIONAL (get attribute from multiple elements)
		"timeout":   "number?",  // OPTIONAL
	}
}

func (t *ExtractAttributeTask) GetOutputSchema() string {
	return "any" // RETURNS STRING OR ARRAY OF STRINGS
}

func (t *ExtractAttributeTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["selector"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["attribute"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ExtractAttributeTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR AND ATTRIBUTE
	selector, _ := config["selector"].(string)
	attribute, _ := config["attribute"].(string)

	// CHECK IF EXTRACTING MULTIPLE
	multiple := false
	if mult, ok := config["multiple"].(bool); ok {
		multiple = mult
	}

	// GET TIMEOUT
	timeout := float64(5000) // DEFAULT 5 SECONDS
	if timeoutVal, ok := config["timeout"].(float64); ok && timeoutVal > 0 {
		timeout = timeoutVal
	}

	if multiple {
		ctx.Logger.Printf("EXTRACTING ATTRIBUTE '%s' FROM MULTIPLE ELEMENTS: %s", attribute, selector)

		// WAIT FOR SELECTOR TO BE PRESENT
		err = page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(timeout),
		})
		if err != nil {
			return TaskData{}, fmt.Errorf("WAIT FOR SELECTOR FAILED: %v", err)
		}

		// EXTRACT ATTRIBUTE FROM ALL MATCHING ELEMENTS
		script := fmt.Sprintf(`(selector) => {
			const elements = Array.from(document.querySelectorAll(selector));
			return elements.map(el => el.getAttribute('%s') || '');
		}`, attribute)

		result, err := page.Evaluate(script, []interface{}{selector})
		if err != nil {
			return TaskData{}, fmt.Errorf("ATTRIBUTE EXTRACTION FAILED: %v", err)
		}

		// CONVERT RESULT TO STRING ARRAY
		attrArray, ok := result.([]interface{})
		if !ok {
			return TaskData{}, fmt.Errorf("UNEXPECTED RESULT TYPE: %T", result)
		}

		ctx.Logger.Printf("EXTRACTED %d ATTRIBUTE VALUES", len(attrArray))

		return TaskData{
			Type:  "array",
			Value: attrArray,
		}, nil
	} else {
		ctx.Logger.Printf("EXTRACTING ATTRIBUTE '%s' FROM ELEMENT: %s", attribute, selector)

		// WAIT FOR SELECTOR TO BE PRESENT
		err = page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(timeout),
		})
		if err != nil {
			return TaskData{}, fmt.Errorf("WAIT FOR SELECTOR FAILED: %v", err)
		}

		// EXTRACT ATTRIBUTE FROM ELEMENT
		attrValue, err := page.Locator(selector).GetAttribute(attribute, playwright.LocatorGetAttributeOptions{})
		if err != nil {
			return TaskData{}, fmt.Errorf("ATTRIBUTE EXTRACTION FAILED: %v", err)
		}

		ctx.Logger.Printf("EXTRACTED ATTRIBUTE VALUE: %s", attrValue)

		return TaskData{
			Type:  "string",
			Value: attrValue,
		}, nil
	}
}

// EXTRACT LINKS TASK
type ExtractLinksTask struct{}

func (t *ExtractLinksTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":        "string",   // REQUIRED
		"selector":      "string?",  // OPTIONAL (defaults to 'a')
		"baseUrl":       "string?",  // OPTIONAL (for resolving relative URLs)
		"normalizeUrls": "boolean?", // OPTIONAL
		"includeText":   "boolean?", // OPTIONAL (include link text)
		"timeout":       "number?",  // OPTIONAL
	}
}

func (t *ExtractLinksTask) GetOutputSchema() string {
	return "array" // RETURNS ARRAY OF LINKS
}

func (t *ExtractLinksTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ExtractLinksTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR (DEFAULT TO ALL LINKS)
	selector := "a"
	if sel, ok := config["selector"].(string); ok && sel != "" {
		selector = sel
	}

	// GET BASE URL FOR RESOLVING RELATIVE LINKS
	baseUrl := ""
	if base, ok := config["baseUrl"].(string); ok && base != "" {
		baseUrl = base
	} else {
		// USE CURRENT PAGE URL AS BASE
		baseUrl = page.URL()
	}

	// CHECK IF SHOULD NORMALIZE URLS
	normalizeUrls := true
	if norm, ok := config["normalizeUrls"].(bool); ok {
		normalizeUrls = norm
	}

	// CHECK IF SHOULD INCLUDE LINK TEXT
	includeText := false
	if inclText, ok := config["includeText"].(bool); ok {
		includeText = inclText
	}

	// GET TIMEOUT
	timeout := float64(5000) // DEFAULT 5 SECONDS
	if timeoutVal, ok := config["timeout"].(float64); ok && timeoutVal > 0 {
		timeout = timeoutVal
	}

	ctx.Logger.Printf("EXTRACTING LINKS FROM ELEMENTS: %s", selector)

	// WAIT FOR SELECTOR TO BE PRESENT
	err = page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return TaskData{}, fmt.Errorf("WAIT FOR SELECTOR FAILED: %v", err)
	}

	// CREATE SCRIPT TO EXTRACT LINKS
	var script string
	if includeText {
		script = `(selector, baseUrl, normalize) => {
			const elements = Array.from(document.querySelectorAll(selector));
			return elements.map(el => {
				const href = el.getAttribute('href') || '';
				const url = href ? (normalize ? new URL(href, baseUrl).href : href) : '';
				return {
					url: url,
					text: el.textContent.trim(),
					title: el.getAttribute('title') || ''
				};
			}).filter(link => link.url);
		}`
	} else {
		script = `(selector, baseUrl, normalize) => {
			const elements = Array.from(document.querySelectorAll(selector));
			return elements.map(el => {
				const href = el.getAttribute('href') || '';
				return href ? (normalize ? new URL(href, baseUrl).href : href) : '';
			}).filter(url => url);
		}`
	}

	// EXECUTE SCRIPT TO EXTRACT LINKS
	result, err := page.Evaluate(script, []interface{}{selector, baseUrl, normalizeUrls})
	if err != nil {
		return TaskData{}, fmt.Errorf("LINK EXTRACTION FAILED: %v", err)
	}

	// PROCESS RESULTS
	links, ok := result.([]interface{})
	if !ok {
		return TaskData{}, fmt.Errorf("UNEXPECTED RESULT TYPE: %T", result)
	}

	ctx.Logger.Printf("EXTRACTED %d LINKS", len(links))

	return TaskData{
		Type:  "array",
		Value: links,
	}, nil
}

// EXTRACT IMAGES TASK
type ExtractImagesTask struct{}

func (t *ExtractImagesTask) GetInputSchema() map[string]string {
	return map[string]string{
		"pageId":          "string",   // REQUIRED
		"selector":        "string?",  // OPTIONAL (defaults to 'img')
		"baseUrl":         "string?",  // OPTIONAL (for resolving relative URLs)
		"normalizeUrls":   "boolean?", // OPTIONAL
		"includeMetadata": "boolean?", // OPTIONAL (include alt, title, dimensions)
		"minWidth":        "number?",  // OPTIONAL (filter by minimum width)
		"minHeight":       "number?",  // OPTIONAL (filter by minimum height)
		"timeout":         "number?",  // OPTIONAL
	}
}

func (t *ExtractImagesTask) GetOutputSchema() string {
	return "array" // RETURNS ARRAY OF IMAGES
}

func (t *ExtractImagesTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["pageId"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ExtractImagesTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET PAGE FROM RESOURCE MANAGER
	page, err := getPage(ctx, config["pageId"])
	if err != nil {
		return TaskData{}, err
	}

	// GET SELECTOR (DEFAULT TO ALL IMAGES)
	selector := "img"
	if sel, ok := config["selector"].(string); ok && sel != "" {
		selector = sel
	}

	// GET BASE URL FOR RESOLVING RELATIVE LINKS
	baseUrl := ""
	if base, ok := config["baseUrl"].(string); ok && base != "" {
		baseUrl = base
	} else {
		// USE CURRENT PAGE URL AS BASE
		baseUrl = page.URL()
	}

	// CHECK IF SHOULD NORMALIZE URLS
	normalizeUrls := true
	if norm, ok := config["normalizeUrls"].(bool); ok {
		normalizeUrls = norm
	}

	// CHECK IF SHOULD INCLUDE METADATA
	includeMetadata := true
	if inclMeta, ok := config["includeMetadata"].(bool); ok {
		includeMetadata = inclMeta
	}

	// GET MIN DIMENSIONS
	minWidth := float64(0)
	if width, ok := config["minWidth"].(float64); ok && width > 0 {
		minWidth = width
	}

	minHeight := float64(0)
	if height, ok := config["minHeight"].(float64); ok && height > 0 {
		minHeight = height
	}

	// GET TIMEOUT
	timeout := float64(5000) // DEFAULT 5 SECONDS
	if timeoutVal, ok := config["timeout"].(float64); ok && timeoutVal > 0 {
		timeout = timeoutVal
	}

	ctx.Logger.Printf("EXTRACTING IMAGES FROM ELEMENTS: %s", selector)

	// WAIT FOR SELECTOR TO BE PRESENT
	err = page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return TaskData{}, fmt.Errorf("WAIT FOR SELECTOR FAILED: %v", err)
	}

	resultStruct := ""
	if includeMetadata {
		resultStruct = `
				result.alt = img.getAttribute('alt') || '';
				result.title = img.getAttribute('title') || '';
				result.width = img.naturalWidth;
				result.height = img.naturalHeight;`
	}

	// CREATE SCRIPT TO EXTRACT IMAGES
	script := fmt.Sprintf(`(selector, baseUrl, normalize, minWidth, minHeight) => {
		const elements = Array.from(document.querySelectorAll(selector));
		return elements
			.filter(img => img.naturalWidth >= minWidth && img.naturalHeight >= minHeight)
			.map(img => {
				const src = img.getAttribute('src') || '';
				const dataSrc = img.getAttribute('data-src') || '';
				const url = dataSrc || src;
				const result = {
					url: url ? (normalize ? new URL(url, baseUrl).href : url) : ''
				};
				
				%s
				
				return result;
			})
			.filter(img => img.url);
	}`, resultStruct)

	// EXECUTE SCRIPT TO EXTRACT IMAGES
	result, err := page.Evaluate(script, []interface{}{selector, baseUrl, normalizeUrls, minWidth, minHeight})
	if err != nil {
		return TaskData{}, fmt.Errorf("IMAGE EXTRACTION FAILED: %v", err)
	}

	// PROCESS RESULTS
	images, ok := result.([]interface{})
	if !ok {
		return TaskData{}, fmt.Errorf("UNEXPECTED RESULT TYPE: %T", result)
	}

	ctx.Logger.Printf("EXTRACTED %d IMAGES", len(images))

	return TaskData{
		Type:  "array",
		Value: images,
	}, nil
}

//
// ASSET TASKS
//

// DOWNLOAD ASSET TASK
type DownloadAssetTask struct{}

func (t *DownloadAssetTask) GetInputSchema() map[string]string {
	return map[string]string{
		"url":      "string",  // REQUIRED
		"folder":   "string?", // OPTIONAL (defaults to 'downloads')
		"filename": "string?", // OPTIONAL (auto-generated if not provided)
		"headers":  "object?", // OPTIONAL (custom headers)
		"timeout":  "number?", // OPTIONAL
	}
}

func (t *DownloadAssetTask) GetOutputSchema() string {
	return "object" // RETURNS DOWNLOAD INFO
}

func (t *DownloadAssetTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["url"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *DownloadAssetTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET URL TO DOWNLOAD
	url, _ := config["url"].(string)

	// GET FOLDER (DEFAULT TO 'downloads')
	folder := "downloads"
	if f, ok := config["folder"].(string); ok && f != "" {
		folder = f
	}

	// ENSURE FOLDER EXISTS
	if err := os.MkdirAll(folder, 0755); err != nil {
		return TaskData{}, fmt.Errorf("FAILED TO CREATE DIRECTORY: %v", err)
	}

	// GET FILENAME (AUTO-GENERATE IF NOT PROVIDED)
	var filename string
	if f, ok := config["filename"].(string); ok && f != "" {
		filename = f
	} else {
		// GENERATE FILENAME BASED ON URL
		filename = utils.GenerateID("asset")

		// TRY TO EXTRACT EXTENSION FROM URL
		urlParts := strings.Split(url, "?")[0] // REMOVE QUERY PARAMETERS
		urlPath := strings.Split(urlParts, "/")
		if len(urlPath) > 0 {
			lastPart := urlPath[len(urlPath)-1]
			if strings.Contains(lastPart, ".") {
				// EXTRACT EXTENSION
				ext := filepath.Ext(lastPart)
				if ext != "" {
					filename += ext
				}
			}
		}

		// ENSURE WE HAVE AN EXTENSION
		if !strings.Contains(filename, ".") {
			filename += ".bin" // DEFAULT EXTENSION
		}
	}

	// COMBINE FOLDER AND FILENAME
	filePath := filepath.Join(folder, filename)

	// GET TIMEOUT
	timeout := float64(60000) // DEFAULT 60 SECONDS
	if timeoutVal, ok := config["timeout"].(float64); ok && timeoutVal > 0 {
		timeout = timeoutVal
	}

	ctx.Logger.Printf("DOWNLOADING ASSET FROM URL: %s TO %s", url, filePath)

	// CREATE HTTP CLIENT WITH TIMEOUT
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}

	// CREATE REQUEST
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TaskData{}, fmt.Errorf("FAILED TO CREATE REQUEST: %v", err)
	}

	// SET DEFAULT HEADERS
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// SET CUSTOM HEADERS IF PROVIDED
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// PERFORM REQUEST
	resp, err := client.Do(req)
	if err != nil {
		return TaskData{}, fmt.Errorf("REQUEST FAILED: %v", err)
	}
	defer resp.Body.Close()

	// CHECK STATUS CODE
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TaskData{}, fmt.Errorf("BAD STATUS CODE: %d", resp.StatusCode)
	}

	// CREATE FILE
	file, err := os.Create(filePath)
	if err != nil {
		return TaskData{}, fmt.Errorf("FAILED TO CREATE FILE: %v", err)
	}
	defer file.Close()

	// COPY RESPONSE BODY TO FILE
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return TaskData{}, fmt.Errorf("FAILED TO DOWNLOAD FILE: %v", err)
	}

	ctx.Logger.Printf("DOWNLOADED %d BYTES TO %s", size, filePath)

	// GET CONTENT TYPE
	contentType := resp.Header.Get("Content-Type")

	// DETECT ASSET TYPE FROM CONTENT TYPE
	assetType := "unknown"
	if strings.Contains(contentType, "image/") {
		assetType = "image"
	} else if strings.Contains(contentType, "video/") {
		assetType = "video"
	} else if strings.Contains(contentType, "audio/") {
		assetType = "audio"
	} else if strings.Contains(contentType, "text/") || strings.Contains(contentType, "application/") {
		assetType = "document"
	}

	// RETURN DOWNLOAD INFO
	return TaskData{
		Type: "object",
		Value: map[string]interface{}{
			"url":         url,
			"filePath":    filePath,
			"size":        size,
			"contentType": contentType,
			"type":        assetType,
			"timestamp":   time.Now().Unix(),
		},
	}, nil
}

// SAVE ASSET TASK
type SaveAssetTask struct{}

func (t *SaveAssetTask) GetInputSchema() map[string]string {
	return map[string]string{
		"jobId":             "string",   // REQUIRED
		"url":               "string",   // REQUIRED
		"title":             "string?",  // OPTIONAL
		"description":       "string?",  // OPTIONAL
		"assetInfo":         "object?",  // OPTIONAL (properties from download task)
		"generateThumbnail": "boolean?", // OPTIONAL
	}
}

func (t *SaveAssetTask) GetOutputSchema() string {
	return "object" // RETURNS ASSET INFO
}

func (t *SaveAssetTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["jobId"]; !ok {
		return ErrMissingRequiredInput
	}
	if _, ok := config["url"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *SaveAssetTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET REQUIRED FIELDS
	jobId, _ := config["jobId"].(string)
	url, _ := config["url"].(string)

	// GET OPTIONAL FIELDS
	title := ""
	if t, ok := config["title"].(string); ok {
		title = t
	}

	description := ""
	if d, ok := config["description"].(string); ok {
		description = d
	}

	// GET ASSET INFO IF PROVIDED
	var assetInfo map[string]interface{}
	if ai, ok := config["assetInfo"].(map[string]interface{}); ok {
		assetInfo = ai
	}

	// GET GENERATE THUMBNAIL FLAG
	generateThumbnail := true
	if gt, ok := config["generateThumbnail"].(bool); ok {
		generateThumbnail = gt
	}

	ctx.Logger.Printf("SAVING ASSET FROM URL: %s", url)

	// CREATE NEW ASSET
	asset := models.Asset{
		ID:          fmt.Sprintf("asset_%s", utils.GenerateID("")),
		JobID:       jobId,
		URL:         url,
		Title:       title,
		Description: description,
		Date:        time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// SET ASSET TYPE AND LOCAL PATH IF AVAILABLE IN ASSET INFO
	if assetInfo != nil {
		if assetType, ok := assetInfo["type"].(string); ok {
			asset.Type = assetType
		}

		if filePath, ok := assetInfo["filePath"].(string); ok {
			asset.LocalPath = filePath
		}

		if size, ok := assetInfo["size"].(int64); ok {
			asset.Size = size
		} else if sizeFloat, ok := assetInfo["size"].(float64); ok {
			asset.Size = int64(sizeFloat)
		}

		// CREATE METADATA
		metadata := models.JSONMap{}
		if contentType, ok := assetInfo["contentType"].(string); ok {
			metadata["contentType"] = contentType
		}
		if timestamp, ok := assetInfo["timestamp"].(int64); ok {
			metadata["timestamp"] = timestamp
		}

		asset.Metadata = metadata
	}

	// GENERATE THUMBNAIL IF REQUESTED
	if generateThumbnail && asset.LocalPath != "" {
		ctx.Logger.Printf("GENERATING THUMBNAIL FOR ASSET")

		// GENERATE THUMBNAIL FILENAME
		thumbnailFilename := fmt.Sprintf("thumb_%s.jpg", asset.ID)
		thumbnailPath := filepath.Join("thumbnails", thumbnailFilename)

		// ENSURE THUMBNAILS DIRECTORY EXISTS
		os.MkdirAll("thumbnails", 0755)

		// GENERATE THUMBNAIL BASED ON ASSET TYPE
		var err error
		switch {
		case strings.HasPrefix(asset.Type, "image"):
			err = utils.GenerateImageThumbnail(asset.LocalPath, thumbnailPath)
		case strings.HasPrefix(asset.Type, "video"):
			err = utils.GenerateVideoThumbnail(asset.LocalPath, thumbnailPath)
		case strings.HasPrefix(asset.Type, "audio"):
			err = utils.GenerateAudioThumbnail(thumbnailPath) // GENERIC AUDIO ICON
		case strings.HasPrefix(asset.Type, "document"):
			err = utils.GenerateDocumentThumbnail(thumbnailPath) // GENERIC DOCUMENT ICON
		default:
			err = utils.GenerateGenericThumbnail(thumbnailPath) // GENERIC ICON
		}

		if err != nil {
			ctx.Logger.Printf("FAILED TO GENERATE THUMBNAIL: %v", err)
		} else {
			asset.ThumbnailPath = thumbnailFilename
			ctx.Logger.Printf("THUMBNAIL GENERATED: %s", thumbnailFilename)
		}
	}

	// SAVE ASSET TO DATABASE
	if err := ctx.Engine.db.Create(&asset).Error; err != nil {
		return TaskData{}, fmt.Errorf("FAILED TO SAVE ASSET TO DATABASE: %v", err)
	}

	ctx.Logger.Printf("ASSET SAVED WITH ID: %s", asset.ID)

	// UPDATE JOB PROGRESS ASSET COUNT
	ctx.Engine.mu.Lock()
	if progress, ok := ctx.Engine.jobProgress[jobId]; ok {
		progress.Assets++
		ctx.Engine.jobProgress[jobId] = progress
	}
	ctx.Engine.mu.Unlock()

	// RETURN ASSET INFO
	return TaskData{
		Type: "object",
		Value: map[string]interface{}{
			"id":            asset.ID,
			"url":           asset.URL,
			"type":          asset.Type,
			"title":         asset.Title,
			"description":   asset.Description,
			"localPath":     asset.LocalPath,
			"thumbnailPath": asset.ThumbnailPath,
			"size":          asset.Size,
		},
	}, nil
}

//
// FLOW CONTROL TASKS
//

// CONDITIONAL TASK
type ConditionalTask struct{}

func (t *ConditionalTask) GetInputSchema() map[string]string {
	return map[string]string{
		"condition": "any",  // REQUIRED
		"ifTrue":    "any?", // OPTIONAL
		"ifFalse":   "any?", // OPTIONAL
	}
}

func (t *ConditionalTask) GetOutputSchema() string {
	return "any" // RETURNS RESULT OF BRANCH EXECUTED
}

func (t *ConditionalTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["condition"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *ConditionalTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET CONDITION VALUE
	condition := false

	// CONDITION CAN BE BOOLEAN OR ANY TYPE THAT CAN BE EVALUATED AS BOOLEAN
	switch c := config["condition"].(type) {
	case bool:
		condition = c
	case string:
		condition = c != "" && c != "false" && c != "0"
	case float64:
		condition = c != 0
	case int:
		condition = c != 0
	case map[string]interface{}:
		condition = len(c) > 0
	case []interface{}:
		condition = len(c) > 0
	case nil:
		condition = false
	default:
		// DEFAULT TO TRUE FOR ANY OTHER NON-EMPTY VALUE
		condition = true
	}

	ctx.Logger.Printf("EVALUATING CONDITION: %v", condition)

	// EXECUTE APPROPRIATE BRANCH
	if condition {
		ctx.Logger.Printf("CONDITION IS TRUE, EXECUTING IF-TRUE BRANCH")

		// GET IF-TRUE VALUE
		if ifTrue, ok := config["ifTrue"]; ok {
			// DETERMINE VALUE TYPE
			var valueType string
			switch ifTrue.(type) {
			case string:
				valueType = "string"
			case float64, int, int64:
				valueType = "number"
			case bool:
				valueType = "boolean"
			case map[string]interface{}:
				valueType = "object"
			case []interface{}:
				valueType = "array"
			case nil:
				valueType = "null"
			default:
				valueType = "any"
			}

			return TaskData{
				Type:  valueType,
				Value: ifTrue,
			}, nil
		}
	} else {
		ctx.Logger.Printf("CONDITION IS FALSE, EXECUTING IF-FALSE BRANCH")

		// GET IF-FALSE VALUE
		if ifFalse, ok := config["ifFalse"]; ok {
			// DETERMINE VALUE TYPE
			var valueType string
			switch ifFalse.(type) {
			case string:
				valueType = "string"
			case float64, int, int64:
				valueType = "number"
			case bool:
				valueType = "boolean"
			case map[string]interface{}:
				valueType = "object"
			case []interface{}:
				valueType = "array"
			case nil:
				valueType = "null"
			default:
				valueType = "any"
			}

			return TaskData{
				Type:  valueType,
				Value: ifFalse,
			}, nil
		}
	}

	// IF NO BRANCH WAS EXECUTED, RETURN THE CONDITION ITSELF
	return TaskData{
		Type:  "boolean",
		Value: condition,
	}, nil
}

// LOOP TASK
type LoopTask struct{}

func (t *LoopTask) GetInputSchema() map[string]string {
	return map[string]string{
		"items":        "array",   // REQUIRED
		"mapFn":        "string?", // OPTIONAL (JavaScript function to apply to each item)
		"filterFn":     "string?", // OPTIONAL (JavaScript function to filter items)
		"reduceFn":     "string?", // OPTIONAL (JavaScript function to reduce items)
		"initialValue": "any?",    // OPTIONAL (initial value for reduce)
	}
}

func (t *LoopTask) GetOutputSchema() string {
	return "array" // RETURNS PROCESSED ARRAY
}

func (t *LoopTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["items"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *LoopTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET ITEMS ARRAY
	itemsAny, ok := config["items"].([]interface{})
	if !ok {
		return TaskData{}, fmt.Errorf("ITEMS MUST BE AN ARRAY")
	}

	items := itemsAny

	ctx.Logger.Printf("PROCESSING %d ITEMS", len(items))

	// APPLY MAP FUNCTION IF PROVIDED
	if mapFn, ok := config["mapFn"].(string); ok && mapFn != "" {
		ctx.Logger.Printf("APPLYING MAP FUNCTION")

		// IN A REAL IMPLEMENTATION, WOULD USE A JS ENGINE (E.G., GOJA)
		// FOR THIS EXAMPLE, WE'LL JUST RETURN THE ORIGINAL ITEMS
		// THIS IS A MOCK IMPLEMENTATION
	}

	// APPLY FILTER FUNCTION IF PROVIDED
	if filterFn, ok := config["filterFn"].(string); ok && filterFn != "" {
		ctx.Logger.Printf("APPLYING FILTER FUNCTION")

		// IN A REAL IMPLEMENTATION, WOULD USE A JS ENGINE (E.G., GOJA)
		// FOR THIS EXAMPLE, WE'LL JUST RETURN THE ORIGINAL ITEMS
		// THIS IS A MOCK IMPLEMENTATION
	}

	// APPLY REDUCE FUNCTION IF PROVIDED
	if reduceFn, ok := config["reduceFn"].(string); ok && reduceFn != "" {
		ctx.Logger.Printf("APPLYING REDUCE FUNCTION")

		// GET INITIAL VALUE
		initialValue := interface{}(nil)
		if iv, ok := config["initialValue"]; ok {
			initialValue = iv
		}

		// IN A REAL IMPLEMENTATION, WOULD USE A JS ENGINE (E.G., GOJA)
		// FOR THIS EXAMPLE, WE'LL JUST RETURN THE ITEMS AS AN ARRAY
		// THIS IS A MOCK IMPLEMENTATION

		if initialValue != nil {
			// IF WE HAD REDUCED TO A SINGLE VALUE, WE'D RETURN IT LIKE THIS
			// BUT FOR THIS MOCK, WE'LL RETURN THE FULL ARRAY
			/*
				valueType := "any"
				switch initialValue.(type) {
				case string:
					valueType = "string"
				case float64, int, int64:
					valueType = "number"
				case bool:
					valueType = "boolean"
				case map[string]interface{}:
					valueType = "object"
				case []interface{}:
					valueType = "array"
				case nil:
					valueType = "null"
				}

				return TaskData{
					Type:  valueType,
					Value: initialValue,
				}, nil
			*/
		}
	}

	ctx.Logger.Printf("LOOP PROCESSING COMPLETE")

	// RETURN PROCESSED ITEMS
	return TaskData{
		Type:  "array",
		Value: items,
	}, nil
}

// WAIT TASK
type WaitTask struct{}

func (t *WaitTask) GetInputSchema() map[string]string {
	return map[string]string{
		"duration": "number", // REQUIRED (milliseconds)
	}
}

func (t *WaitTask) GetOutputSchema() string {
	return "boolean" // RETURNS SUCCESS STATUS
}

func (t *WaitTask) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["duration"]; !ok {
		return ErrMissingRequiredInput
	}
	return nil
}

func (t *WaitTask) Execute(ctx *TaskContext, config map[string]interface{}) (TaskData, error) {
	// GET DURATION
	duration := float64(1000) // DEFAULT 1 SECOND
	if d, ok := config["duration"].(float64); ok && d > 0 {
		duration = d
	}

	ctx.Logger.Printf("WAITING FOR %d MS", int(duration))

	// CREATE TIMER
	timer := time.NewTimer(time.Duration(duration) * time.Millisecond)

	// WAIT UNTIL TIMER EXPIRES OR CONTEXT IS CANCELLED
	select {
	case <-timer.C:
		ctx.Logger.Printf("WAIT COMPLETED")
		return TaskData{
			Type:  "boolean",
			Value: true,
		}, nil

	case <-ctx.Context.Done():
		timer.Stop()
		ctx.Logger.Printf("WAIT CANCELLED")
		return TaskData{}, ctx.Context.Err()
	}
}
