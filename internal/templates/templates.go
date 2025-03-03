package templates

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed html/*.html
var templateFS embed.FS

//go:embed icons/*.svg
var iconsFS embed.FS

// TEMPLATES HOLDS PARSED TEMPLATES
var Templates *template.Template

// INITTEMPLATES INITIALIZES ALL TEMPLATES
func InitTemplates() error {
	var err error

	// CREATE NECESSARY DIRECTORIES
	if err := os.MkdirAll("web/templates", 0755); err != nil {
		return err
	}

	if err := os.MkdirAll("web/static/icons", 0755); err != nil {
		return err
	}

	// EXTRACT TEMPLATE FILES IF NEEDED
	if err := ExtractTemplates(); err != nil {
		return err
	}

	// EXTRACT ICONS AND CONVERT TO JPG
	if err := ExtractAndConvertIcons(); err != nil {
		return err
	}

	// PARSE ALL TEMPLATES
	Templates, err = template.ParseGlob("web/templates/*.html")
	return err
}

// CREATETEMPLATES IS KEPT FOR BACKWARD COMPATIBILITY
func CreateTemplates() error {
	return InitTemplates()
}

// CREATESTATICFILES IS KEPT FOR BACKWARD COMPATIBILITY
func CreateStaticFiles() error {
	return ExtractAndConvertIcons()
}

// EXTRACTTEMPLATES EXTRACTS HTML TEMPLATES TO THE FILESYSTEM
func ExtractTemplates() error {
	// LIST ALL HTML TEMPLATES
	entries, err := fs.ReadDir(templateFS, "html")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			// READ TEMPLATE CONTENT
			content, err := fs.ReadFile(templateFS, "html/"+entry.Name())
			if err != nil {
				return err
			}

			// WRITE TO DISK
			targetPath := filepath.Join("web/templates", entry.Name())
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// EXTRACTANDCONVERTICONS EXTRACTS SVG ICONS AND CONVERTS THEM TO JPG
func ExtractAndConvertIcons() error {
	// LIST ALL SVG ICONS
	entries, err := fs.ReadDir(iconsFS, "icons")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".svg") {
			// READ SVG CONTENT
			content, err := fs.ReadFile(iconsFS, "icons/"+entry.Name())
			if err != nil {
				return err
			}

			// WRITE SVG TO DISK
			svgPath := filepath.Join("web/static/icons", entry.Name())
			if err := os.WriteFile(svgPath, content, 0644); err != nil {
				return err
			}

			// CONVERT TO JPG
			ConvertSvgToJpg()
		}
	}

	return nil
}

// CONVERTSVGTOJPG CONVERTS SVG FILES TO JPG FORMAT
func ConvertSvgToJpg() error {
	// FIND ALL SVG FILES
	svgFiles, err := filepath.Glob("web/static/icons/*.svg")
	if err != nil {
		return err
	}

	for _, svgFile := range svgFiles {
		// GET OUTPUT FILENAME
		jpgFile := strings.TrimSuffix(svgFile, ".svg") + ".jpg"

		// SKIP IF JPG ALREADY EXISTS
		if _, err := os.Stat(jpgFile); err == nil {
			continue
		}

		// USE IMAGEMAGICK TO CONVERT
		cmd := exec.Command(
			"convert",
			svgFile,
			jpgFile,
		)

		if err := cmd.Run(); err != nil {
			log.Printf("WARNING: COULD NOT CONVERT %s TO JPG: %v", svgFile, err)
			// CREATE A FALLBACK JPG IF CONVERT FAILS
			CreateFallbackJpg(jpgFile)
		}
	}

	return nil
}

// CREATEFALLBACKJPG CREATES A SIMPLE FALLBACK JPG IF CONVERSION FAILS
func CreateFallbackJpg(jpgPath string) {
	// CREATE A SIMPLE 320X320 BLACK IMAGE AS FALLBACK
	img := make([]byte, 320*320*3)
	for i := range img {
		img[i] = 0 // BLACK
	}

	// WRITE AS RAW JPG
	os.WriteFile(jpgPath, img, 0644)
}
