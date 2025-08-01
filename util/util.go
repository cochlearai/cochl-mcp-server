package util

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

func NormalizePath(path string) (string, error) {
	// URL decode the path first
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		return "", fmt.Errorf("failed to decode path: %v", err)
	}
	path = decodedPath

	if strings.HasPrefix(path, "http") {
		return path, nil
	}

	path = filepath.FromSlash(path)

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(path, `\`) || strings.HasPrefix(path, `/`) {
			path = strings.TrimLeft(path, `\/`)
		}
	}

	path = filepath.Clean(path)

	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute: %s", path)
	}

	return path, nil
}
