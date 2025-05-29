package util

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		windows bool
	}{
		{
			name:    "Windows drive letter path",
			input:   "C:/Users/test/file.mp3",
			want:    filepath.FromSlash("C:/Users/test/file.mp3"),
			windows: true,
		},
		{
			name:    "Windows path with slash prefix",
			input:   "/c:/Users/test/file.mp3",
			want:    filepath.FromSlash("c:/Users/test/file.mp3"),
			windows: true,
		},
		{
			name:    "Unix absolute path",
			input:   "/home/user/file.mp3",
			want:    "/home/user/file.mp3",
			windows: false,
		},
		{
			name:  "Clean double slashes",
			input: "/home//user///file.mp3",
			want:  "/home/user/file.mp3",
		},
		{
			name:    "Relative path should fail",
			input:   "relative/path/file.mp3",
			wantErr: true,
		},
		{
			name:    "Empty path should fail",
			input:   "",
			wantErr: true,
		},
		{
			name:    "URL encoded Windows path",
			input:   "/c%3A/Users/test/file%20name.mp3",
			want:    filepath.FromSlash("c:/Users/test/file name.mp3"),
			windows: true,
		},
		{
			name:    "URL encoded Unix path with special characters",
			input:   "/home/user/한글%20파일.mp3",
			want:    "/home/user/한글 파일.mp3",
			windows: false,
		},
		{
			name:    "Invalid URL encoded path should fail",
			input:   "/home/user/%XX",
			wantErr: true,
		},
	}

	isWindows := runtime.GOOS == "windows"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.windows && !isWindows) || (tt.windows == false && isWindows) {
				t.Skip("Test is specific to different OS")
			}

			got, err := NormalizePath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
