package scraper

import (
	"fmt"

	"github.com/nickheyer/Crepes/internal/models"
)

// GENERATE FERRET TEMPLATE FROM JOB CONFIG
func GenerateFerretTemplate(job *models.Job) string {
	// BASIC TEMPLATE
	template := `
LET baseUrl = '%s'
LET doc = DOCUMENT(baseUrl, {
    driver: "cdp"
})

LET results = []

// NAVIGATION AND EXTRACTION FUNCTIONS
LET extract = (page, parentUrl) -> (
    LET pageUrl = page.url()
    LET results = []
`
	// ADD SELECTORS FOR ASSETS
	extractCode := ""
	hasAssetSelectors := false
	hasLinkSelectors := false
	hasPaginationSelectors := false

	// PROCESS SELECTORS
	for _, s := range job.Selectors {
		if selector, ok := s.(map[string]interface{}); ok {
			purpose := getStringProp(selector, "purpose", "")
			selectorType := getStringProp(selector, "type", "css")
			value := getStringProp(selector, "value", "")
			attribute := getStringProp(selector, "attribute", "")

			if value != "" {
				switch purpose {
				case "assets":
					hasAssetSelectors = true
					extractCode += generateAssetExtractorCode(selectorType, value, attribute)
				case "links":
					hasLinkSelectors = true
				case "pagination":
					hasPaginationSelectors = true
				case "metadata":
					// METADATA SELECTORS WILL BE INCLUDED WITH ASSETS
				}
			}
		}
	}

	// IF NO ASSET SELECTORS, ADD DEFAULT IMAGE SELECTOR
	if !hasAssetSelectors {
		extractCode += `
    // DEFAULT IMAGE EXTRACTOR
    FOR img IN page.elements("img")
        LET imgSrc = img.attribute("src")
        LET imgAlt = img.attribute("alt")
        
        FILTER imgSrc != NULL
        
        // RESOLVE RELATIVE URL
        LET fullUrl = imgSrc
        IF NOT STARTS_WITH(imgSrc, "http")
            SET fullUrl = CONCAT(parentUrl, imgSrc)
        END
        
        PUSH(results, {
            url: fullUrl,
            title: imgAlt,
            type: "image"
        })
    END
`
	}

	// ADD RETURN STATEMENT FOR EXTRACT LAMBDA
	extractCode += `
    RETURN results
)
`

	// CREATE CRAWL FUNCTION WITH CONFIG FROM JOB
	crawlCode := `
// CRAWL FUNCTION
LET crawl = (startUrl, maxDepth) -> (
    LET visited = {}
    LET queue = [{url: startUrl, depth: 0}]
    LET results = []
    LET maxAssets = %d
    
    WHILE LENGTH(queue) > 0
        LET current = POP(queue)
        LET url = current.url
        LET depth = current.depth
        
        // SKIP IF ALREADY VISITED OR EXCEEDS MAX DEPTH
        CONTINUE_WHEN visited[url] != NULL
        CONTINUE_WHEN depth > maxDepth
        
        // MARK AS VISITED
        SET visited[url] = TRUE
        
        TRY
            // OPEN PAGE
            LET page = DOCUMENT(url, {
                driver: "cdp",
                timeout: %d
            })
            
            // EXTRACT ASSETS FROM PAGE
            LET pageResults = extract(page, url)
            FOR item IN pageResults
                // LIMIT NUMBER OF RESULTS IF MAXASSETS IS SET
                IF maxAssets > 0 AND LENGTH(results) >= maxAssets
                    BREAK
                END
                
                PUSH(results, item)
            END
            
            // EXTRACT LINKS FOR FURTHER CRAWLING
            IF depth < maxDepth
`
	// ADD LINK SELECTORS
	linkCode := ""
	if hasLinkSelectors {
		for _, s := range job.Selectors {
			if selector, ok := s.(map[string]interface{}); ok {
				purpose := getStringProp(selector, "purpose", "")
				if purpose == "links" {
					value := getStringProp(selector, "value", "")
					if value != "" {
						linkCode += generateLinkExtractorCode(value)
					}
				}
			}
		}
	}

	// IF NO LINK SELECTORS, ADD DEFAULT
	if !hasLinkSelectors {
		linkCode += `
                // DEFAULT LINK EXTRACTOR
                FOR link IN page.elements("a")
                    LET href = link.attribute("href")
                    FILTER href != NULL
                    FILTER NOT STARTS_WITH(href, "#")
                    
                    // RESOLVE RELATIVE URL
                    LET nextUrl = href
                    IF NOT STARTS_WITH(href, "http")
                        SET nextUrl = RESOLVE_URL(url, href)
                    END
                    
                    // APPLY URL FILTERS
                    LET shouldFollow = TRUE
`
		// ADD INCLUDE PATTERN
		if includePattern, ok := job.Rules["includeUrlPattern"].(string); ok && includePattern != "" {
			linkCode += fmt.Sprintf(`
                    // INCLUDE PATTERN
                    LET matchesInclude = REGEX_TEST(@'%s', nextUrl)
                    IF NOT matchesInclude
                        SET shouldFollow = FALSE
                    END
`, includePattern)
		}
		// ADD EXCLUDE PATTERN
		if excludePattern, ok := job.Rules["excludeUrlPattern"].(string); ok && excludePattern != "" {
			linkCode += fmt.Sprintf(`
                    // EXCLUDE PATTERN
                    LET matchesExclude = REGEX_TEST(@'%s', nextUrl)
                    IF matchesExclude
                        SET shouldFollow = FALSE
                    END
`, excludePattern)
		}
		// ADD QUEUE INSERTION
		linkCode += `
                    IF shouldFollow
                        PUSH(queue, {
                            url: nextUrl,
                            depth: depth + 1
                        })
                    END
                END
`
	}

	// ADD PAGINATION HANDLING
	paginationCode := ""
	if hasPaginationSelectors {
		for _, s := range job.Selectors {
			if selector, ok := s.(map[string]interface{}); ok {
				purpose := getStringProp(selector, "purpose", "")
				if purpose == "pagination" {
					value := getStringProp(selector, "value", "")
					if value != "" {
						paginationCode += generatePaginationCode(value)
					}
				}
			}
		}
	}

	// CLOSE CRAWL FUNCTION LAMBDA
	crawlCode += linkCode + paginationCode + `
            END
        CATCH
            // LOG ERROR AND CONTINUE
            LET error = CURRENT_EXCEPTION()
            WARN(error)
        END
    END
    
    RETURN results
)
`

	// MAIN EXECUTION CODE
	maxDepth := 2  // DEFAULT
	maxAssets := 0 // NO LIMIT

	// SET MAX DEPTH FROM JOB RULES
	if depth, ok := job.Rules["maxDepth"].(float64); ok {
		maxDepth = int(depth)
	}

	if assets, ok := job.Rules["maxAssets"].(float64); ok {
		maxAssets = int(assets)
	}

	// DEFAULT TIMEOUT
	timeout := 30000 // 30 SECONDS

	// SET TIMEOUT FROM JOB CONFIG
	if t, ok := job.Rules["requestDelay"].(float64); ok {
		timeout = int(t) + 30000 // ADD BASE TIMEOUT
	}

	mainCode := fmt.Sprintf(`
// MAIN EXECUTION
LET crawlResults = crawl(baseUrl, %d)
RETURN crawlResults
`, maxDepth)

	// ASSEMBLE FINAL TEMPLATE
	finalTemplate := fmt.Sprintf(template, job.BaseURL) + extractCode + fmt.Sprintf(crawlCode, maxAssets, timeout) + mainCode

	return finalTemplate
}

// GENERATE ASSET EXTRACTOR CODE
func generateAssetExtractorCode(selectorType, value, attribute string) string {
	if attribute == "" {
		attribute = "src"
	}

	return fmt.Sprintf(`
    // ASSET EXTRACTOR
    FOR elem IN page.elements('%s')
        LET elemSrc = elem.attribute('%s')
        LET elemTitle = elem.attribute('title') || elem.attribute('alt') || ''
        
        FILTER elemSrc != NULL
        
        // RESOLVE RELATIVE URL
        LET fullUrl = elemSrc
        IF NOT STARTS_WITH(elemSrc, "http")
            SET fullUrl = RESOLVE_URL(parentUrl, elemSrc)
        END
        
        PUSH(results, {
            url: fullUrl,
            title: elemTitle,
            type: "%s"
        })
    END
`, value, attribute, selectorType)
}

// GENERATE LINK EXTRACTOR CODE
func generateLinkExtractorCode(value string) string {
	return fmt.Sprintf(`
                // LINK EXTRACTOR
                FOR link IN page.elements('%s')
                    LET href = link.attribute('href')
                    FILTER href != NULL
                    FILTER NOT STARTS_WITH(href, "#")
                    
                    // RESOLVE RELATIVE URL
                    LET nextUrl = href
                    IF NOT STARTS_WITH(href, "http")
                        SET nextUrl = RESOLVE_URL(url, href)
                    END
                    
                    PUSH(queue, {
                        url: nextUrl,
                        depth: depth + 1
                    })
                END
`, value)
}

// GENERATE PAGINATION CODE
func generatePaginationCode(value string) string {
	return fmt.Sprintf(`
                // PAGINATION HANDLER
                FOR nextLink IN page.elements('%s')
                    LET nextHref = nextLink.attribute('href')
                    FILTER nextHref != NULL
                    
                    // RESOLVE RELATIVE URL
                    LET nextPageUrl = nextHref
                    IF NOT STARTS_WITH(nextHref, "http")
                        SET nextPageUrl = RESOLVE_URL(url, nextHref)
                    END
                    
                    // ADD TO QUEUE WITH SAME DEPTH TO PRIORITIZE
                    PUSH(queue, {
                        url: nextPageUrl,
                        depth: depth
                    })
                    
                    // ONLY USE FIRST PAGINATION LINK
                    BREAK
                END
`, value)
}

// HELPER TO GET STRING PROPERTY SAFELY
func getStringProp(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}
