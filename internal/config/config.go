package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CONFIG STRUCTURE
type Config struct {
	Port           string `json:"port"`
	StoragePath    string `json:"storagePath"`
	ThumbnailsPath string `json:"thumbnailsPath"`
	DataPath       string `json:"dataPath"`
	MaxConcurrent  int    `json:"maxConcurrent"`
	DefaultTimeout int    `json:"defaultTimeout"` // IN MS
}

// LOAD CONFIG FROM FILE
func LoadConfig(path string) (*Config, error) {
	// READ CONFIG FILE
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// VALIDATE AS RAW JSON
	var raw json.RawMessage
	if err := json.Unmarshal(file, &raw); err != nil {
		return nil, err
	}

	// PARSE CONFIG JSON
	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	// ENSURE ALL PATHS ARE VALID
	config.StoragePath = sanitizePath(config.StoragePath)
	config.ThumbnailsPath = sanitizePath(config.ThumbnailsPath)
	config.DataPath = sanitizePath(config.DataPath)

	return &config, nil
}

// SAVE CONFIG TO FILE
func SaveConfig(config *Config, path string) error {
	// MARSHAL CONFIG TO JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// WRITE CONFIG FILE
	return os.WriteFile(path, data, 0644)
}

// GET DEFAULT CONFIG
func GetDefaultConfig() *Config {
	return &Config{
		Port:           "8080",
		StoragePath:    "./storage",
		ThumbnailsPath: "./thumbnails",
		DataPath:       "./data",
		MaxConcurrent:  5,
		DefaultTimeout: 5 * 60 * 1000, // 5 MINUTES IN MS
	}
}

// SANITIZE PATH TO ENSURE IT'S VALID
func sanitizePath(path string) string {
	// MAKE SURE PATH IS NOT EMPTY
	if path == "" {
		return "."
	}
	// CLEAN PATH
	return filepath.Clean(path)
}
