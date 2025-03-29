package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nickheyer/Crepes/internal/config"
	"github.com/nickheyer/Crepes/internal/utils"
)

func GetStorageInfo(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storagePath := cfg.StoragePath
		if !filepath.IsAbs(storagePath) {
			absPath, err := filepath.Abs(storagePath)
			if err == nil {
				storagePath = absPath
			}
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(storagePath, &stat); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get storage info")
			return
		}
		blockSize := uint64(stat.Bsize)
		totalBlocks := stat.Blocks
		freeBlocks := stat.Bfree
		availableBlocks := stat.Bavail
		totalSize := blockSize * totalBlocks
		freeSize := blockSize * freeBlocks
		availableSize := blockSize * availableBlocks
		usedSize := totalSize - freeSize
		assetsSize, err := getDirSize(cfg.StoragePath)
		if err != nil {
			assetsSize = 0
		}
		thumbsSize, err := getDirSize(cfg.ThumbnailsPath)
		if err != nil {
			thumbsSize = 0
		}
		response := map[string]any{
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
		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    response,
		})
	}
}

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
