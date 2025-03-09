package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MontFerret/ferret"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

// HTTP RESPONSES
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// GENERATE A UNIQUE ID WITH PREFIX
func GenerateID(prefix string) string {
	uuid := uuid.New().String()
	return fmt.Sprintf("%s_%s", prefix, strings.Replace(uuid, "-", "", -1))
}

// FORMAT FILE SIZE
func FormatFileSize(size uint64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := uint64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// FORMAT DURATION
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// GENERATE A FILENAME FOR DOWNLOADED ASSETS
func GenerateFilename(sourceURL, contentType string) string {
	// EXTRACT FILENAME FROM URL
	parsedURL, err := url.Parse(sourceURL)
	if err == nil {
		path := parsedURL.Path
		if path != "" && path != "/" {
			filename := filepath.Base(path)
			// IF FILENAME HAS EXTENSION, USE IT
			if filepath.Ext(filename) != "" {
				return filename
			}
		}
	}

	// GENERATE RANDOM FILENAME WITH APPROPRIATE EXTENSION
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	// GET
	// GET EXTENSION FROM CONTENT TYPE
	var extension string
	switch {
	case strings.HasPrefix(contentType, "image/jpeg"):
		extension = ".jpg"
	case strings.HasPrefix(contentType, "image/png"):
		extension = ".png"
	case strings.HasPrefix(contentType, "image/gif"):
		extension = ".gif"
	case strings.HasPrefix(contentType, "image/webp"):
		extension = ".webp"
	case strings.HasPrefix(contentType, "video/mp4"):
		extension = ".mp4"
	case strings.HasPrefix(contentType, "video/webm"):
		extension = ".webm"
	case strings.HasPrefix(contentType, "audio/mpeg"):
		extension = ".mp3"
	case strings.HasPrefix(contentType, "audio/wav"):
		extension = ".wav"
	case strings.HasPrefix(contentType, "application/pdf"):
		extension = ".pdf"
	case strings.HasPrefix(contentType, "application/msword"):
		extension = ".doc"
	case strings.HasPrefix(contentType, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"):
		extension = ".docx"
	default:
		extension = ".bin"
	}

	return randomStr + extension
}

// RESOLVE RELATIVE URL TO ABSOLUTE
func ResolveURL(baseURL, relativeURL string) string {
	// PARSE BASE URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL
	}

	// HANDLE CASE WHERE RELATIVE URL IS ALREADY ABSOLUTE
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	// HANDLE ROOT-RELATIVE URLS
	if strings.HasPrefix(relativeURL, "/") {
		base.Path = relativeURL
		return base.String()
	}

	// HANDLE RELATIVE URLS
	rel, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL
	}

	return base.ResolveReference(rel).String()
}

// THUMBNAIL GENERATION
func generatePlaceholderThumbnail(thumbnailPath string, bgColor color.Color) error {
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
	f, err := os.Create(thumbnailPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
}

func GenerateImageThumbnail(sourcePath, thumbnailPath string) error {
	src, err := imaging.Open(sourcePath)
	if err != nil {
		return err
	}
	thumbnail := imaging.Resize(src, 300, 0, imaging.Lanczos)
	return imaging.Save(thumbnail, thumbnailPath)
}

func GenerateVideoThumbnail(sourcePath, thumbnailPath string) error {
	bgColor := color.RGBA{0, 0, 128, 255}
	return generatePlaceholderThumbnail(thumbnailPath, bgColor)
}

func GenerateAudioThumbnail(thumbnailPath string) error {
	bgColor := color.RGBA{0, 128, 0, 255}
	return generatePlaceholderThumbnail(thumbnailPath, bgColor)
}

func GenerateDocumentThumbnail(thumbnailPath string) error {
	bgColor := color.RGBA{128, 0, 0, 255}
	return generatePlaceholderThumbnail(thumbnailPath, bgColor)
}

func GenerateGenericThumbnail(thumbnailPath string) error {
	bgColor := color.RGBA{128, 128, 128, 255}
	return generatePlaceholderThumbnail(thumbnailPath, bgColor)
}

func ValidateFerretTemplate(template string) (bool, []string) {
	if template == "" {
		return false, []string{"Template is empty"}
	}

	compiler := ferret.New()
	_, err := compiler.Compile(template)
	if err != nil {
		return false, []string{err.Error()}
	}
	return true, nil
}

// GET BASIC FERRET EXAMPLE
func GetBasicFerretExample() string {
	return `
LET baseUrl = "https://example.com"
LET doc = DOCUMENT(baseUrl, { driver: "cdp" })

// Basic extraction lambda
LET extract = (page, parentUrl) -> (
    LET results = []
    FOR img IN page.elements("img")
        LET imgSrc = img.attribute("src")
        LET imgAlt = img.attribute("alt")
        FILTER imgSrc != NULL
        LET fullUrl = IF STARTS_WITH(imgSrc, "http") THEN imgSrc ELSE RESOLVE_URL(parentUrl, imgSrc) END
        PUSH(results, { url: fullUrl, title: imgAlt, type: "image" })
    END
    RETURN results
)

// Crawl lambda
LET crawl = (startUrl, maxDepth) -> (
    LET visited = {}
    LET queue = [{ url: startUrl, depth: 0 }]
    LET results = []
    WHILE LENGTH(queue) > 0
        LET current = POP(queue)
        LET url = current.url
        LET depth = current.depth
        IF visited[url] != NULL OR depth > maxDepth THEN CONTINUE END
        SET visited[url] = TRUE
        TRY
            LET page = DOCUMENT(url, { driver: "cdp", timeout: 30000 })
            LET pageResults = extract(page, url)
            FOR item IN pageResults
                PUSH(results, item)
            END
            IF depth < maxDepth THEN
                FOR link IN page.elements("a")
                    LET href = link.attribute("href")
                    FILTER href != NULL
                    FILTER NOT STARTS_WITH(href, "#")
                    LET nextUrl = IF STARTS_WITH(href, "http") THEN href ELSE RESOLVE_URL(url, href) END
                    PUSH(queue, { url: nextUrl, depth: depth + 1 })
                END
            END
        CATCH
            WARN(CURRENT_EXCEPTION())
        END
    END
    RETURN results
)

LET crawlResults = crawl(baseUrl, 2)
RETURN crawlResults
`
}

// GET IMAGE FERRET EXAMPLE
func GetImageFerretExample() string {
	return `
LET baseUrl = "https://example.com"
LET doc = DOCUMENT(baseUrl, { driver: "cdp" })

// Lambda to extract images
LET extractImages = (page, parentUrl) -> (
    LET results = []
    FOR img IN page.elements("img")
        LET imgSrc = img.attribute("src")
        LET imgSrcset = img.attribute("srcset")
        LET imgDataSrc = img.attribute("data-src")
        LET imgLazySrc = img.attribute("data-lazy-src")
        LET imgAlt = img.attribute("alt")
        LET imgTitle = img.attribute("title")
        LET bestSrc = IF imgSrc != NULL THEN imgSrc
                      ELSE IF imgDataSrc != NULL THEN imgDataSrc
                      ELSE IF imgLazySrc != NULL THEN imgLazySrc
                      ELSE IF imgSrcset != NULL THEN SPLIT(imgSrcset, " ")[0]
                      ELSE NULL
                      END
        FILTER bestSrc != NULL
        LET fullUrl = IF STARTS_WITH(bestSrc, "http") THEN bestSrc ELSE RESOLVE_URL(parentUrl, bestSrc) END
        CONTINUE_WHEN STARTS_WITH(fullUrl, "data:")
        LET title = IF imgAlt != NULL AND imgAlt != "" THEN imgAlt ELSE IF imgTitle != NULL THEN imgTitle ELSE "" END
        PUSH(results, { url: fullUrl, title: title, type: "image", width: img.property("naturalWidth"), height: img.property("naturalHeight") })
    END
    RETURN results
)

// Crawl lambda for images
LET crawl = (startUrl, maxDepth) -> (
    LET visited = {}
    LET queue = [{ url: startUrl, depth: 0 }]
    LET results = []
    WHILE LENGTH(queue) > 0
        LET current = POP(queue)
        LET url = current.url
        LET depth = current.depth
        IF visited[url] != NULL OR depth > maxDepth THEN CONTINUE END
        SET visited[url] = TRUE
        TRY
            LET page = DOCUMENT(url, { driver: "cdp", timeout: 30000 })
            LET pageResults = extractImages(page, url)
            FOR item IN pageResults
                PUSH(results, item)
            END
            IF depth < maxDepth THEN
                FOR link IN page.elements("a")
                    LET href = link.attribute("href")
                    FILTER href != NULL
                    FILTER NOT STARTS_WITH(href, "#")
                    LET nextUrl = IF STARTS_WITH(href, "http") THEN href ELSE RESOLVE_URL(url, href) END
                    LET currentDomain = URL_PARSE(url).hostname
                    LET nextDomain = URL_PARSE(nextUrl).hostname
                    IF currentDomain == nextDomain THEN
                        PUSH(queue, { url: nextUrl, depth: depth + 1 })
                    END
                END
            END
        CATCH
            WARN(CURRENT_EXCEPTION())
        END
    END
    RETURN results
)

LET crawlResults = crawl(baseUrl, 2)
RETURN crawlResults
`
}

// GET PAGINATION FERRET EXAMPLE
func GetPaginationFerretExample() string {
	return `
LET baseUrl = "https://example.com"
LET doc = DOCUMENT(baseUrl, { driver: "cdp" })

// Lambda to extract content with pagination
LET extractWithPagination = (baseUrl, maxPages) -> (
    LET results = []
    LET currentPage = 1
    LET currentUrl = baseUrl
    LET hasNextPage = TRUE
    WHILE hasNextPage AND currentPage <= maxPages
        LOG("Processing page: ", currentPage, ", URL: ", currentUrl)
        LET page = DOCUMENT(currentUrl, { driver: "cdp", timeout: 30000 })
        FOR item IN page.elements(".item-selector")
            LET title = item.text()
            LET link = item.element("a").attribute("href")
            LET fullUrl = IF STARTS_WITH(link, "http") THEN link ELSE RESOLVE_URL(currentUrl, link) END
            PUSH(results, { url: fullUrl, title: title, page: currentPage })
        END
        LET nextLink = NULL
        TRY
            SET nextLink = page.element("a.next") || page.element('a[rel="next"]')
            IF nextLink == NULL THEN
                LET pageLinks = page.elements(".pagination a")
                LET nextPageNum = currentPage + 1
                FOR link IN pageLinks
                    IF TO_INT(link.text()) == nextPageNum THEN
                        SET nextLink = link
                        BREAK
                    END
                END
            END
        CATCH
            SET nextLink = NULL
        END
        IF nextLink != NULL THEN
            LET nextHref = nextLink.attribute("href")
            SET currentUrl = IF STARTS_WITH(nextHref, "http") THEN nextHref ELSE RESOLVE_URL(baseUrl, nextHref) END
            SET currentPage = currentPage + 1
        ELSE
            SET hasNextPage = FALSE
        END
    END
    RETURN results
)

LET results = extractWithPagination(baseUrl, 5)
RETURN results
`
}

// GET ARTICLE FERRET EXAMPLE
func GetArticleFerretExample() string {
	return `
LET baseUrl = "https://example.com/blog"
LET doc = DOCUMENT(baseUrl, { driver: "cdp" })

// Lambda to extract articles from a page
LET extractArticles = (page, parentUrl) -> (
    LET results = []
    FOR article IN page.elements("article")
        LET titleElem = article.element("h2") || article.element("h1")
        CONTINUE_WHEN titleElem == NULL
        LET linkElem = titleElem ? titleElem.element("a") : NULL
        LET title = titleElem.text()
        LET link = linkElem ? linkElem.attribute("href") : NULL
        LET dateElem = article.element("time") || article.element(".date") || article.element(".published")
        LET pubDate = dateElem ? dateElem.text() : ""
        LET excerptElem = article.element("p.excerpt") || article.element(".summary") || article.element("p")
        LET excerpt = excerptElem ? excerptElem.text() : ""
        LET imgElem = article.element("img")
        LET imgSrc = imgElem ? imgElem.attribute("src") : NULL
        LET fullUrl = IF link != NULL AND NOT STARTS_WITH(link, "http") THEN RESOLVE_URL(parentUrl, link) ELSE link END
        LET fullImgUrl = IF imgSrc != NULL AND NOT STARTS_WITH(imgSrc, "http") THEN RESOLVE_URL(parentUrl, imgSrc) ELSE imgSrc END
        LET result = { url: fullUrl, title: title, type: "article", date: pubDate, description: excerpt }
        IF fullImgUrl != NULL THEN SET result.image = fullImgUrl END
        PUSH(results, result)
    END
    RETURN results
)

// Lambda to get article content
LET getArticleContent = (url) -> (
    TRY
        LET page = DOCUMENT(url, { driver: "cdp", timeout: 30000 })
        LET title = page.element("h1").text()
        LET contentElem = page.element("article") || page.element(".content") || page.element(".post-content")
        LET content = contentElem ? contentElem.innerHTML() : ""
        LET images = []
        FOR img IN page.elements("article img")
            LET imgSrc = img.attribute("src")
            IF imgSrc != NULL THEN
                LET fullImgUrl = IF NOT STARTS_WITH(imgSrc, "http") THEN RESOLVE_URL(url, imgSrc) ELSE imgSrc END
                PUSH(images, fullImgUrl)
            END
        END
        RETURN { url: url, title: title, content: content, images: images }
    CATCH
        RETURN NULL
    END
)

// Lambda to crawl the blog with pagination.
LET crawlBlog = (startUrl, maxPages) -> (
    LET results = []
    LET currentPage = 1
    LET currentUrl = startUrl
    LET hasNextPage = TRUE
    WHILE hasNextPage AND currentPage <= maxPages
        LET page = DOCUMENT(currentUrl, { driver: "cdp", timeout: 30000 })
        LET pageResults = extractArticles(page, currentUrl)
        FOR item IN pageResults
            PUSH(results, item)
        END
        LET nextLink = NULL
        TRY
            SET nextLink = page.element("a.next") || page.element('a[rel="next"]')
            IF nextLink == NULL THEN
                LET nextPageNum = currentPage + 1
                FOR link IN page.elements(".pagination a")
                    IF TO_INT(link.text()) == nextPageNum THEN
                        SET nextLink = link
                        BREAK
                    END
                END
            END
        CATCH
            SET nextLink = NULL
        END
        IF nextLink != NULL THEN
            LET nextHref = nextLink.attribute("href")
            SET currentUrl = IF STARTS_WITH(nextHref, "http") THEN nextHref ELSE RESOLVE_URL(startUrl, nextHref) END
            SET currentPage = currentPage + 1
        ELSE
            SET hasNextPage = FALSE
        END
    END
    RETURN results
)

LET blogResults = crawlBlog(baseUrl, 3)
RETURN blogResults
`
}

// GET ECOMMERCE FERRET EXAMPLE
func GetEcommerceFerretExample() string {
	return `
LET baseUrl = "https://example.com/shop"
LET doc = DOCUMENT(baseUrl, { driver: "cdp" })

// Lambda to extract products from a listing page
LET extractProducts = (page, parentUrl) -> (
    LET results = []
    FOR product IN page.elements(".product-item")
        LET title = product.element(".product-title").text()
        LET link = product.element("a").attribute("href")
        LET priceElem = product.element(".price") || product.element(".product-price")
        LET price = priceElem ? priceElem.text() : ""
        LET imgElem = product.element("img")
        LET imgSrc = imgElem ? imgElem.attribute("src") : NULL
        LET fullUrl = IF NOT STARTS_WITH(link, "http") THEN RESOLVE_URL(parentUrl, link) ELSE link END
        LET fullImgUrl = IF imgSrc != NULL AND NOT STARTS_WITH(imgSrc, "http") THEN RESOLVE_URL(parentUrl, imgSrc) ELSE imgSrc END
        LET result = { url: fullUrl, title: title, type: "product", price: price }
        IF fullImgUrl != NULL THEN SET result.image = fullImgUrl END
        PUSH(results, result)
    END
    RETURN results
)

// Lambda to get detailed product info
LET getProductDetails = (url) -> (
    TRY
        LET page = DOCUMENT(url, { driver: "cdp", timeout: 30000 })
        LET title = page.element("h1").text()
        LET priceElem = page.element(".price") || page.element("#product-price")
        LET price = priceElem ? priceElem.text() : ""
        LET descElem = page.element(".product-description") || page.element("#description")
        LET description = descElem ? descElem.text() : ""
        LET images = []
        FOR img IN page.elements(".product-gallery img")
            LET imgSrc = img.attribute("src") || img.attribute("data-src")
            IF imgSrc != NULL THEN
                LET fullImgUrl = IF NOT STARTS_WITH(imgSrc, "http") THEN RESOLVE_URL(url, imgSrc) ELSE imgSrc END
                PUSH(images, fullImgUrl)
            END
        END
        LET specs = {}
        FOR row IN page.elements(".product-specs tr")
            LET label = row.element("th").text()
            LET value = row.element("td").text()
            SET specs[label] = value
        END
        RETURN { url: url, title: title, price: price, description: description, images: images, specifications: specs }
    CATCH
        RETURN NULL
    END
)

// Lambda to crawl the shop with pagination and categories.
LET crawlShop = (startUrl, maxPages) -> (
    LET results = []
    LET visited = {}
    LET queue = [{ url: startUrl, depth: 0, type: "listing" }]
    WHILE LENGTH(queue) > 0
        LET current = POP(queue)
        LET url = current.url
        LET type = current.type
        IF visited[url] != NULL THEN CONTINUE END
        SET visited[url] = TRUE
        TRY
            IF type == "listing" THEN
                LET page = DOCUMENT(url, { driver: "cdp", timeout: 30000 })
                LET productResults = extractProducts(page, url)
                FOR product IN productResults
                    PUSH(results, product)
                    PUSH(queue, { url: product.url, depth: current.depth + 1, type: "product" })
                END
                FOR categoryLink IN page.elements(".category-menu a")
                    LET catHref = categoryLink.attribute("href")
                    LET fullCatUrl = IF NOT STARTS_WITH(catHref, "http") THEN RESOLVE_URL(url, catHref) ELSE catHref END
                    PUSH(queue, { url: fullCatUrl, depth: current.depth, type: "listing" })
                END
                LET nextLink = page.element("a.next") || page.element(".pagination a[rel="next"]")
                IF nextLink != NULL THEN
                    LET nextUrl = nextLink.attribute("href")
                    SET nextUrl = IF NOT STARTS_WITH(nextUrl, "http") THEN RESOLVE_URL(url, nextUrl) ELSE nextUrl END
                    PUSH(queue, { url: nextUrl, depth: current.depth, type: "listing" })
                END
            ELSE IF type == "product" THEN
                LET productDetails = getProductDetails(url)
                IF productDetails != NULL THEN
                    LET found = FALSE
                    FOR i IN RANGE(0, LENGTH(results) - 1)
                        IF results[i].url == url THEN
                            SET results[i] = productDetails
                            SET found = TRUE
                            BREAK
                        END
                    END
                    IF NOT found THEN PUSH(results, productDetails) END
                END
            END
        CATCH
            WARN("Error processing ", url, ": ", CURRENT_EXCEPTION())
        END
    END
    RETURN results
)

LET shopResults = crawlShop(baseUrl, 3)
RETURN shopResults
`
}
