package util

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

func NormalizePath(path string) (string, error) {
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
