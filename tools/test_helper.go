package tools

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func parseParams[T any](t *testing.T, args map[string]any, target T) (T, error) {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return target, err
	}
	err = json.Unmarshal(argsJSON, target)
	return target, err
}

func getAbsPath(t *testing.T, relativePath string) string {
	t.Helper()
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}
	return absPath
}
