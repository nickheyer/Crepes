package assets

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

	// CHECK ASSET TYPE
	switch asset.Type {
	case "video":
		// EXTRACT FRAME USING FFMPEG
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-i", filepath.Join(config.AppConfig.StoragePath, asset.LocalPath),
			"-ss", "00:00:05", // TAKE FRAME AT 5 SECONDS
			"-vframes", "1",
			"-vf", "scale=320:-1",
			"-y",
			thumbPath,
		)

		if err := cmd.Run(); err != nil {
			log.Printf("FFMPEG failed for video thumbnail: %v", err)
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

// GENERATEGENERICTHUMBNAIL CREATES A GENERIC THUMBNAIL BASED ON FILE TYPE
func GenerateGenericThumbnail(asset *models.Asset) (string, error) {
	// CREATE GENERIC THUMBNAIL BASED ON FILE TYPE
	thumbnailDir := filepath.Join(config.AppConfig.ThumbnailsPath, filepath.Dir(asset.LocalPath))
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", err
	}

	thumbName := fmt.Sprintf("%s.jpg", asset.ID)
	thumbPath := filepath.Join(thumbnailDir, thumbName)
	relThumbPath := filepath.Join(filepath.Dir(asset.LocalPath), thumbName)

	// COPY GENERIC ICON BASED ON TYPE
	genericPath := fmt.Sprintf("./static/icons/%s.jpg", asset.Type)
	if !FileExists(genericPath) {
		genericPath = "./static/icons/generic.jpg"
	}

	input, err := os.Open(genericPath)
	if err != nil {
		return "", err
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
	// CREATE A SIMPLE 320X320 BLACK IMAGE AS FALLBACK
	img := make([]byte, 320*320*3)
	for i := range img {
		img[i] = 0 // BLACK
	}

	// WRITE AS RAW JPG (NOT IDEAL BUT WORKS AS EMERGENCY FALLBACK)
	os.WriteFile(jpgPath, img, 0644)
}

// FILEEXISTS CHECKS IF A FILE EXISTS
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
