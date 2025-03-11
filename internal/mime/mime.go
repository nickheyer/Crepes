package mime

import (
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// EXTENSIVEMIMETYPEEXTENSIONMAP MAPS MIME TYPES TO FILE EXTENSIONS
var EXTENSIVEMIMETYPEEXTENSIONMAP = map[string]string{
	// IMAGES
	"image/jpeg":               ".jpg",
	"image/jpg":                ".jpg",
	"image/pjpeg":              ".jpg",
	"image/png":                ".png",
	"image/apng":               ".apng",
	"image/gif":                ".gif",
	"image/webp":               ".webp",
	"image/svg+xml":            ".svg",
	"image/tiff":               ".tiff",
	"image/bmp":                ".bmp",
	"image/x-icon":             ".ico",
	"image/vnd.microsoft.icon": ".ico",
	"image/x-ms-bmp":           ".bmp",
	"image/heif":               ".heif",
	"image/heic":               ".heic",
	"image/avif":               ".avif",

	// DOCUMENTS
	"application/pdf":         ".pdf",
	"application/msword":      ".doc",
	"application/vnd.ms-word": ".doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	"application/vnd.ms-excel": ".xls",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
	"application/vnd.ms-powerpoint":                                             ".ppt",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
	"application/rtf":           ".rtf",
	"text/plain":                ".txt",
	"text/csv":                  ".csv",
	"text/tab-separated-values": ".tsv",
	"application/vnd.oasis.opendocument.text":         ".odt",
	"application/vnd.oasis.opendocument.spreadsheet":  ".ods",
	"application/vnd.oasis.opendocument.presentation": ".odp",
	"application/epub+zip":                            ".epub",
	"application/x-ibooks+zip":                        ".ibooks",
	"application/vnd.amazon.ebook":                    ".azw",
	"application/vnd.amazon.mobi8-ebook":              ".azw3",

	// ARCHIVES
	"application/zip":                         ".zip",
	"application/x-rar-compressed":            ".rar",
	"application/x-7z-compressed":             ".7z",
	"application/x-tar":                       ".tar",
	"application/gzip":                        ".gz",
	"application/x-bzip2":                     ".bz2",
	"application/x-xz":                        ".xz",
	"application/java-archive":                ".jar",
	"application/vnd.android.package-archive": ".apk",

	// AUDIO
	"audio/mpeg":     ".mp3",
	"audio/mp4":      ".m4a",
	"audio/ogg":      ".ogg",
	"audio/wav":      ".wav",
	"audio/webm":     ".weba",
	"audio/aac":      ".aac",
	"audio/flac":     ".flac",
	"audio/x-ms-wma": ".wma",
	"audio/midi":     ".mid",
	"audio/x-midi":   ".mid",

	// VIDEO
	"video/mp4":        ".mp4",
	"video/mpeg":       ".mpeg",
	"video/ogg":        ".ogv",
	"video/webm":       ".webm",
	"video/x-msvideo":  ".avi",
	"video/quicktime":  ".mov",
	"video/x-matroska": ".mkv",
	"video/x-flv":      ".flv",
	"video/x-ms-wmv":   ".wmv",
	"video/3gpp":       ".3gp",
	"video/3gpp2":      ".3g2",

	// WEB
	"text/html":                ".html",
	"application/xhtml+xml":    ".xhtml",
	"text/css":                 ".css",
	"text/javascript":          ".js",
	"application/javascript":   ".js",
	"application/x-javascript": ".js",
	"application/json":         ".json",
	"application/ld+json":      ".jsonld",
	"application/xml":          ".xml",
	"text/xml":                 ".xml",
	"application/rss+xml":      ".rss",
	"application/atom+xml":     ".atom",
	"application/wasm":         ".wasm",
	"text/markdown":            ".md",
	"application/graphql":      ".graphql",

	// FONTS
	"font/ttf":                      ".ttf",
	"font/otf":                      ".otf",
	"font/woff":                     ".woff",
	"font/woff2":                    ".woff2",
	"application/vnd.ms-fontobject": ".eot",
	"application/x-font-ttf":        ".ttf",

	// BINARY/EXECUTABLE
	"application/octet-stream":    ".bin",
	"application/x-msdownload":    ".exe",
	"application/x-msdos-program": ".exe",
	"application/x-msi":           ".msi",
	"application/x-deb":           ".deb",
	"application/x-rpm":           ".rpm",
	"application/x-sh":            ".sh",
	"application/x-gtar":          ".gtar",

	// DATA/SCIENTIFIC
	"application/vnd.geo+json":                    ".geojson",
	"application/sql":                             ".sql",
	"application/vnd.sqlite3":                     ".sqlite",
	"application/vnd.tcpdump.pcap":                ".pcap",
	"application/x-hdf5":                          ".h5",
	"application/x-parquet":                       ".parquet",
	"application/vnd.wolfram.mathematica.package": ".m",
	"application/vnd.wolfram.cdf":                 ".cdf",
	"application/mathematica":                     ".nb",
	"application/matlab":                          ".mat",
	"application/x-netcdf":                        ".nc",

	// 3D/CAD/DESIGN
	"model/gltf-binary":            ".glb",
	"model/gltf+json":              ".gltf",
	"model/obj":                    ".obj",
	"model/stl":                    ".stl",
	"model/vnd.collada+xml":        ".dae",
	"application/x-blender":        ".blend",
	"application/x-dxf":            ".dxf",
	"application/acad":             ".dwg",
	"application/vnd.ms-pki.stl":   ".stl",
	"application/vnd.sketchup.skp": ".skp",

	// VECTOR/DESIGN
	"application/illustrator":        ".ai",
	"application/x-photoshop":        ".psd",
	"application/x-indesign":         ".indd",
	"application/vnd.adobe.xd":       ".xd",
	"application/vnd.ms-xpsdocument": ".xps",
	"application/x-coreldraw":        ".cdr",
	"application/x-gimp":             ".xcf",

	// MISC
	"text/calendar":                    ".ics",
	"text/vcard":                       ".vcf",
	"application/pgp-signature":        ".sig",
	"application/pgp-keys":             ".key",
	"application/x-pkcs7-signature":    ".p7s",
	"application/pkcs10":               ".p10",
	"application/x-pkcs12":             ".p12",
	"application/x-pkcs7-certificates": ".p7b",
	"application/x-x509-ca-cert":       ".cer",
}

// GETEXTENSIONFORCONTENTTYPE RETURNS APPROPRIATE FILE EXTENSION FOR A GIVEN CONTENT TYPE
func GetExtensionForContentType(contentType string, fileURL string) string {
	// CLEAN UP CONTENT TYPE
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	// STRIP PARAMETERS
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}

	// CHECK FOR EXACT MATCH IN OUR MAP
	if ext, found := EXTENSIVEMIMETYPEEXTENSIONMAP[contentType]; found {
		return ext
	}

	// TRY STANDARD LIBRARY
	exts, err := mime.ExtensionsByType(contentType)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}

	// TRY TO EXTRACT FROM URL
	if fileURL != "" {
		parsedURL, err := url.Parse(fileURL)
		if err == nil {
			// GET EXTENSION FROM PATH
			urlExt := filepath.Ext(parsedURL.Path)
			if urlExt != "" {
				return urlExt
			}

			// CHECK FOR COMMON FILE INDICATORS IN QUERY PARAMS
			query := parsedURL.Query()
			for key, values := range query {
				if len(values) > 0 {
					// LOOK FOR FILE PARAMETERS
					lowerKey := strings.ToLower(key)
					if lowerKey == "file" || lowerKey == "filename" || lowerKey == "name" ||
						lowerKey == "download" || lowerKey == "attachment" {
						if ext := filepath.Ext(values[0]); ext != "" {
							return ext
						}
					}
				}
			}
		}
	}

	// HANDLE SPECIAL CONTENT-TYPE PATTERNS
	if strings.HasPrefix(contentType, "audio/") {
		return ".audio"
	}
	if strings.HasPrefix(contentType, "video/") {
		return ".video"
	}
	if strings.HasPrefix(contentType, "image/") {
		return ".img"
	}
	if strings.HasPrefix(contentType, "text/") {
		return ".txt"
	}

	// DEFAULT FALLBACK
	return ".bin"
}

// ANALYZEFILETYPE PROVIDES DETAILED FILE TYPE INFORMATION
func AnalyzeFileType(contentType string, fileURL string, headers http.Header) FileTypeInfo {
	info := FileTypeInfo{
		MimeType:  contentType,
		Extension: GetExtensionForContentType(contentType, fileURL),
	}

	// DETERMINE CATEGORY
	if strings.HasPrefix(contentType, "image/") {
		info.Category = "image"
	} else if strings.HasPrefix(contentType, "video/") {
		info.Category = "video"
	} else if strings.HasPrefix(contentType, "audio/") {
		info.Category = "audio"
	} else if strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "xml") {
		info.Category = "text"
	} else if strings.Contains(contentType, "pdf") ||
		strings.Contains(contentType, "word") ||
		strings.Contains(contentType, "excel") ||
		strings.Contains(contentType, "powerpoint") ||
		strings.Contains(contentType, "document") {
		info.Category = "document"
	} else if strings.Contains(contentType, "zip") ||
		strings.Contains(contentType, "compressed") ||
		strings.Contains(contentType, "archive") ||
		strings.Contains(contentType, "tar") ||
		strings.Contains(contentType, "gzip") {
		info.Category = "archive"
	} else if strings.Contains(contentType, "font") {
		info.Category = "font"
	} else {
		info.Category = "binary"
	}

	// CHECK CONTENT DISPOSITION
	if cd := headers.Get("Content-Disposition"); cd != "" {
		if strings.Contains(cd, "attachment") {
			info.IsAttachment = true
		}

		// GET FILENAME FROM CONTENT DISPOSITION
		if idx := strings.Index(cd, "filename="); idx != -1 {
			filename := cd[idx+9:]
			// STRIP QUOTES IF PRESENT
			if strings.HasPrefix(filename, "\"") && strings.HasSuffix(filename, "\"") {
				filename = filename[1 : len(filename)-1]
			}
			info.SuggestedFilename = filename

			// IF NO EXTENSION YET, GET FROM FILENAME
			if info.Extension == ".bin" && filepath.Ext(filename) != "" {
				info.Extension = filepath.Ext(filename)
			}
		}
	}

	return info
}

// FILETYPEINFO HOLDS DETAILED INFORMATION ABOUT A FILE
type FileTypeInfo struct {
	MimeType          string
	Extension         string
	Category          string
	IsAttachment      bool
	SuggestedFilename string
}
