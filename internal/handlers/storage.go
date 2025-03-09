package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/utils"

	"github.com/gorilla/mux"
)

// REGISTER STORAGE HANDLERS
func RegisterStorageHandlers(router *mux.Router, cfg *config.Config) {
	// GET STORAGE INFO
	router.HandleFunc("/storage/info", func(w http.ResponseWriter, r *http.Request) {
		// GET STORAGE PATH
		storagePath := cfg.StoragePath
		if !filepath.IsAbs(storagePath) {
			absPath, err := filepath.Abs(storagePath)
			if err == nil {
				storagePath = absPath
			}
		}

		// GET DISK USAGE
		var stat syscall.Statfs_t
		if err := syscall.Statfs(storagePath, &stat); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get storage info")
			return
		}

		// CALCULATE VALUES
		blockSize := uint64(stat.Bsize)
		totalBlocks := stat.Blocks
		freeBlocks := stat.Bfree
		availableBlocks := stat.Bavail

		totalSize := blockSize * totalBlocks
		freeSize := blockSize * freeBlocks
		availableSize := blockSize * availableBlocks
		usedSize := totalSize - freeSize

		// GET STORAGE USAGE FOR ASSETS AND THUMBNAILS
		assetsSize, err := getDirSize(cfg.StoragePath)
		if err != nil {
			assetsSize = 0
		}

		thumbsSize, err := getDirSize(cfg.ThumbnailsPath)
		if err != nil {
			thumbsSize = 0
		}

		// FORMAT SIZES
		response := map[string]interface{}{
			"totalSpace":    utils.FormatFileSize(totalSize),
			"usedSpace":     utils.FormatFileSize(usedSize),
			"freeSpace":     utils.FormatFileSize(availableSize),
			"assetsSize":    utils.FormatFileSize(assetsSize),
			"thumbnailSize": utils.FormatFileSize(thumbsSize),
			"raw": map[string]uint64{
				"totalBytes":  totalSize,
				"usedBytes":   usedSize,
				"freeBytes":   availableSize,
				"assetsBytes": assetsSize,
				"thumbsBytes": thumbsSize,
			},
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    response,
		})
	}).Methods("GET")
}

// GET DIRECTORY SIZE RECURSIVELY
func getDirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return nil
	})
	return size, err
}
