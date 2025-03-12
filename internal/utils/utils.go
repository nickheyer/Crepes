package utils

import (
	"crypto/md5"
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

func GenerateHash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])[:12] // TRUNCATE TO 12 CHARS FOR FILENAME USE
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

func NormalizeURL(baseURL, relativeURL string) string {
	// Check if URL is already absolute
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL // Return original if parsing fails
	}

	// Special case for protocol-relative URLs (//example.com/path)
	if strings.HasPrefix(relativeURL, "//") {
		return base.Scheme + ":" + relativeURL
	}

	// Handle absolute paths (/path/to/resource)
	if strings.HasPrefix(relativeURL, "/") {
		return fmt.Sprintf("%s://%s%s", base.Scheme, base.Host, relativeURL)
	}

	// Handle relative paths
	baseDir := "/"
	if strings.Contains(base.Path, "/") && base.Path != "/" {
		baseDir = base.Path[:strings.LastIndex(base.Path, "/")+1]
	}

	// Join paths for relative URLs
	return fmt.Sprintf("%s://%s%s%s", base.Scheme, base.Host, baseDir, relativeURL)
}
