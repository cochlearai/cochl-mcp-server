package util

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type FilePath struct {
	Path     string
	IsRemote bool
}

func NormalizePath(path string) (*FilePath, error) {
	// URL decode the path first
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		return nil, fmt.Errorf("failed to decode path: %v", err)
	}
	path = decodedPath

	if strings.HasPrefix(path, "http") {
		return &FilePath{
			Path:     path,
			IsRemote: true,
		}, nil
	}

	path = filepath.FromSlash(path)

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(path, `\`) || strings.HasPrefix(path, `/`) {
			path = strings.TrimLeft(path, `\/`)
		}
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

// IsDropboxURL checks if the URL is a Dropbox URL
func IsDropboxURL(url string) bool {
	return strings.Contains(url, "dropbox.com")
}
