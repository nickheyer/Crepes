package assets

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/models"
)

// GENERATETHUMBNAIL GENERATES A THUMBNAIL FOR AN ASSET
func GenerateThumbnail(asset *models.Asset) (string, error) {
	// CREATE THUMBNAIL DIRECTORY
	thumbnailDir := filepath.Join(config.AppConfig.ThumbnailsPath, filepath.Dir(asset.LocalPath))
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", err
	}

	thumbName := fmt.Sprintf("%s.jpg", asset.ID)
	thumbPath := filepath.Join(thumbnailDir, thumbName)
	relThumbPath := filepath.Join(filepath.Dir(asset.LocalPath), thumbName)

	// ENSURE STATIC DIRECTORY EXISTS
	staticIconsDir := "./static/icons"
	if err := os.MkdirAll(staticIconsDir, 0755); err != nil {
		log.Printf("Error creating static icons directory: %v", err)
	}

	// CREATE DEFAULT ICONS IF THEY DON'T EXIST
	iconTypes := []string{"video", "audio", "image", "document", "generic"}
	for _, iconType := range iconTypes {
		iconPath := filepath.Join(staticIconsDir, iconType+".jpg")
		if !FileExists(iconPath) {
			// CREATE A SIMPLE COLORED SQUARE AS ICON
			CreateFallbackJpg(iconPath)
		}
	}

	// CHECK ASSET TYPE
	switch asset.Type {
	case "video":
		// EXTRACT FRAME USING FFMPEG
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// CHECK IF FILE EXISTS AND HAS SIZE
		sourceFile := filepath.Join(config.AppConfig.StoragePath, asset.LocalPath)
		info, err := os.Stat(sourceFile)
		if err != nil {
			log.Printf("Source file error: %v", err)
			return GenerateGenericThumbnail(asset)
		}

		if info.Size() < 10000 {
			log.Printf("Video file too small for ffmpeg: %d bytes", info.Size())
			return GenerateGenericThumbnail(asset)
		}

		cmd := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-i", sourceFile,
			"-ss", "00:00:01", // TAKE FRAME AT 1 SECOND (MORE LIKELY TO WORK)
			"-vframes", "1",
			"-vf", "scale=320:-1",
			"-y",
			thumbPath,
		)

		// CAPTURE ERROR OUTPUT FOR DEBUGGING
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			log.Printf("FFMPEG failed for video thumbnail: %v", err)
			log.Printf("FFMPEG error output: %s", stderr.String())
			return GenerateGenericThumbnail(asset)
		}

	case "image":
		// RESIZE IMAGE USING FFMPEG
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-i", filepath.Join(config.AppConfig.StoragePath, asset.LocalPath),
			"-vf", "scale=320:-1",
			"-y",
			thumbPath,
		)

		if err := cmd.Run(); err != nil {
			log.Printf("FFMPEG failed for image thumbnail: %v", err)
			return GenerateGenericThumbnail(asset)
		}

	default:
		return GenerateGenericThumbnail(asset)
	}

	return relThumbPath, nil
}

func GenerateGenericThumbnail(asset *models.Asset) (string, error) {
	// CREATE GENERIC THUMBNAIL BASED ON FILE TYPE
	thumbnailDir := filepath.Join(config.AppConfig.ThumbnailsPath, filepath.Dir(asset.LocalPath))
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", err
	}

	thumbName := fmt.Sprintf("%s.jpg", asset.ID)
	thumbPath := filepath.Join(thumbnailDir, thumbName)
	relThumbPath := filepath.Join(filepath.Dir(asset.LocalPath), thumbName)

	// ENSURE STATIC DIRECTORY EXISTS
	staticIconsDir := "./static/icons"
	if err := os.MkdirAll(staticIconsDir, 0755); err != nil {
		log.Printf("Error creating static icons directory: %v", err)
		// CREATE A FALLBACK THUMBNAIL DIRECTLY
		CreateFallbackJpg(thumbPath)
		return relThumbPath, nil
	}

	// COPY GENERIC ICON BASED ON TYPE
	genericPath := fmt.Sprintf("./static/icons/%s.jpg", asset.Type)
	if !FileExists(genericPath) {
		genericPath = "./static/icons/generic.jpg"

		// CREATE GENERIC ICON IF IT DOESN'T EXIST
		if !FileExists(genericPath) {
			CreateFallbackJpg(genericPath)
		}
	}

	input, err := os.Open(genericPath)
	if err != nil {
		// FALLBACK TO DIRECT CREATION
		CreateFallbackJpg(thumbPath)
		return relThumbPath, nil
	}
	defer input.Close()

	output, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	if err != nil {
		return "", err
	}

	return relThumbPath, nil
}

// CREATEFALLBACKJPG CREATES A SIMPLE FALLBACK JPG IF IMAGEMAGICK IS NOT AVAILABLE
func CreateFallbackJpg(jpgPath string) {
	// CREATE A SIMPLE 320X320 COLORED IMAGE AS FALLBACK
	width, height := 320, 320
	img := make([]byte, width*height*3)

	// CHOOSE COLOR BASED ON FILENAME
	var r, g, b byte = 0, 0, 0

	filename := filepath.Base(jpgPath)
	switch {
	case strings.Contains(filename, "video"):
		r, g, b = 25, 25, 100 // DARK BLUE
	case strings.Contains(filename, "audio"):
		r, g, b = 25, 100, 25 // DARK GREEN
	case strings.Contains(filename, "image"):
		r, g, b = 100, 25, 25 // DARK RED
	case strings.Contains(filename, "document"):
		r, g, b = 100, 100, 25 // YELLOW
	default:
		r, g, b = 50, 50, 50 // GRAY
	}

	// FILL IMAGE WITH COLOR
	for i := 0; i < width*height; i++ {
		img[i*3] = r   // R
		img[i*3+1] = g // G
		img[i*3+2] = b // B
	}

	// CREATE THE FILE
	if err := os.MkdirAll(filepath.Dir(jpgPath), 0755); err != nil {
		log.Printf("Error creating directory for fallback JPG: %v", err)
		return
	}

	// WRITE TO JPG USING FFMPEG
	tempRaw := jpgPath + ".raw"
	if err := os.WriteFile(tempRaw, img, 0644); err != nil {
		log.Printf("Error writing raw image data: %v", err)
		return
	}

	// TRY TO CONVERT WITH FFMPEG
	cmd := exec.Command(
		"ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "rgb24",
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-i", tempRaw,
		"-y",
		jpgPath,
	)

	if err := cmd.Run(); err != nil {
		log.Printf("FFMPEG failed to convert raw image: %v", err)

		// FALLBACK TO WRITING DIRECTLY
		f, err := os.Create(jpgPath)
		if err != nil {
			log.Printf("Error creating fallback JPG: %v", err)
			return
		}
		defer f.Close()

		// WRITE SIMPLE JPG HEADER
		header := []byte{
			0xFF, 0xD8, // SOI
			0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // APP0
			0xFF, 0xDB, 0x00, 0x43, 0x00, // DQT
		}

		// SIMPLE QUANTIZATION TABLE (ALL 1'S)
		quantTable := make([]byte, 64)
		for i := range quantTable {
			quantTable[i] = 1
		}

		f.Write(header)
		f.Write(quantTable)
		f.Write(img) // NOT A VALID JPG, BUT WILL DISPLAY AS SOMETHING
	}

	// CLEAN UP TEMP FILE
	os.Remove(tempRaw)
}

// FILEEXISTS CHECKS IF A FILE EXISTS
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
