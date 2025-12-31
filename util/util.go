package util

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"resty.dev/v3"

	"github.com/cochlearai/cochl-mcp-server/util/restcli"
)

type FilePath struct {
	Path     string
	IsRemote bool
}

func NormalizePath(path string) (*FilePath, error) {
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		return nil, fmt.Errorf("failed to decode path: %w", err)
	}
	path = decodedPath

	if strings.HasPrefix(path, "http") {
		return &FilePath{
			Path:     path,
			IsRemote: true,
		}, nil
	}

	// Check if input is a Windows-style path (C:\ or C:/)
	isWindowsPath := regexp.MustCompile(`^[A-Za-z]:[/\\]`).MatchString(path)

	if runtime.GOOS == "windows" {
		// Running on Windows (native binary)
		path = filepath.FromSlash(path)

		if strings.HasPrefix(path, `\`) || strings.HasPrefix(path, `/`) {
			path = strings.TrimLeft(path, `\/`)
		}
	} else if runtime.GOOS == "linux" && isWindowsPath {
		// Running on Linux (Docker) with Windows path
		// Convert C:\path\to\file -> /C/path/to/file
		driveLetter := strings.ToUpper(string(path[0]))
		pathWithoutDrive := path[2:] // Skip "C:"
		pathWithoutDrive = strings.ReplaceAll(pathWithoutDrive, `\`, `/`)
		path = "/" + driveLetter + pathWithoutDrive
	} else {
		// Linux path on Linux
		path = filepath.FromSlash(path)
	}

	path = filepath.Clean(path)

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	return &FilePath{
		Path:     path,
		IsRemote: false,
	}, nil
}

// ConvertGoogleDriveURL converts a Google Drive share URL to a direct download URL
func ConvertGoogleDriveURL(shareURL string) (string, error) {
	// Check if it's already a direct download URL
	if strings.Contains(shareURL, "uc?export=download&id=") ||
		(strings.Contains(shareURL, "uc?id=") && strings.Contains(shareURL, "&export=download")) {
		return shareURL, nil
	}

	// Regular expression to match Google Drive file ID from various URL formats
	patterns := []string{
		`drive\.google\.com/file/d/([a-zA-Z0-9_-]+)`,
		`drive\.google\.com/open\?id=([a-zA-Z0-9_-]+)`,
		`drive\.google\.com/uc\?.*id=([a-zA-Z0-9_-]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(shareURL)
		if len(matches) > 1 {
			fileID := matches[1]
			// Convert to direct download URL
			return fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", fileID), nil
		}
	}

	return "", fmt.Errorf("invalid Google Drive URL format: %s", shareURL)
}

// IsGoogleDriveURL checks if the URL is a Google Drive URL
func IsGoogleDriveURL(url string) bool {
	return strings.Contains(url, "drive.google.com")
}

// ConvertDropboxURL converts a Dropbox share URL to a direct download URL
func ConvertDropboxURL(shareURL string) (string, error) {
	// Check if it's already a direct download URL
	if strings.Contains(shareURL, "dl=1") {
		return shareURL, nil
	}

	// Check if it's a Dropbox URL with dl=0 parameter
	if strings.Contains(shareURL, "dropbox.com") && strings.Contains(shareURL, "dl=0") {
		// Replace dl=0 with dl=1 to make it a direct download URL
		return strings.Replace(shareURL, "dl=0", "dl=1", 1), nil
	}

	return "", fmt.Errorf("invalid Dropbox URL format: %s", shareURL)
}

// IsDropboxURL checks if the URL is a Dropbox URL.
func IsDropboxURL(url string) bool {
	return strings.Contains(url, "dropbox.com")
}

// IsHTTPURL checks if the given URL is HTTP or HTTPS.
func IsHTTPURL(fileURL string) bool {
	lower := strings.ToLower(fileURL)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// DownloadOptions configures the HTTP download behavior.
type DownloadOptions struct {
	Timeout        time.Duration
	ContentTypeMap map[string]string // Content-Type to format mapping
}

// DownloadResult contains the result of a file download.
type DownloadResult struct {
	Data   []byte
	Format string
}

// DownloadFromHTTP downloads a file from HTTP URL with configurable options.
func DownloadFromHTTP(fileURL string, opts DownloadOptions) (*DownloadResult, error) {
	downloadURL := fileURL
	if IsGoogleDriveURL(fileURL) {
		converted, err := ConvertGoogleDriveURL(fileURL)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Google Drive URL: %w", err)
		}
		downloadURL = converted
	} else if IsDropboxURL(fileURL) {
		converted, err := ConvertDropboxURL(fileURL)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Dropbox URL: %w", err)
		}
		downloadURL = converted
	}

	if opts.Timeout == 0 {
		opts.Timeout = time.Minute
	}

	client := resty.New().
		SetTimeout(opts.Timeout).
		SetRetryCount(2).
		SetRetryWaitTime(time.Second)

	resp, err := restcli.Get(client, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status())
	}

	data := resp.Bytes()
	contentType := resp.Header().Get("Content-Type")

	var format string
	if opts.ContentTypeMap != nil {
		format = opts.ContentTypeMap[contentType]
	}

	// Fallback to URL extension if Content-Type mapping failed
	if format == "" {
		format = ExtractExtensionFromURL(fileURL)
	}

	return &DownloadResult{
		Data:   data,
		Format: format,
	}, nil
}

// ExtractExtensionFromURL extracts the file extension from a URL or file path.
func ExtractExtensionFromURL(fileURL string) string {
	if parsedURL, err := url.Parse(fileURL); err == nil && parsedURL.Path != "" {
		ext := strings.ToLower(filepath.Ext(parsedURL.Path))
		if ext != "" {
			return ext[1:] // Remove the dot
		}
	}

	ext := strings.ToLower(filepath.Ext(fileURL))
	if ext != "" {
		return ext[1:]
	}

	return ""
}
