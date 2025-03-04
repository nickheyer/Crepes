package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed build/* build/**/*
var distDir embed.FS

// GetFileSystem returns an http.FileSystem for the embedded dist files
func GetFileSystem() http.FileSystem {
	// Get the sub filesystem from the embedded files
	distFS, err := fs.Sub(distDir, "build")
	if err != nil {
		panic(err)
	}
	return http.FS(distFS)
}
