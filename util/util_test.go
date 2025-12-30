package util

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// boolPtr returns a pointer to the given bool value
func boolPtr(b bool) *bool {
	return &b
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		want         string
		wantIsRemote bool
		wantErr      bool
		osSpecific   *bool // nil = run on all OS, true = Windows only, false = Linux only
	}{
		// Remote URL test
		{
			name:         "HTTPS URL should be remote",
			input:        "https://example.com/audio.mp3",
			want:         "https://example.com/audio.mp3",
			wantIsRemote: true,
		},

		// Windows binary tests (runtime.GOOS == "windows")
		{
			name:       "Windows: backslash path",
			input:      `C:\Users\test\file.mp3`,
			want:       filepath.FromSlash("C:/Users/test/file.mp3"),
			osSpecific: boolPtr(true),
		},
		{
			name:       "Windows: forward slash path",
			input:      "C:/Users/test/file.mp3",
			want:       filepath.FromSlash("C:/Users/test/file.mp3"),
			osSpecific: boolPtr(true),
		},

		// Linux Docker tests (runtime.GOOS == "linux" with Windows paths)
		{
			name:       "Linux Docker: Windows path with backslash",
			input:      `C:\Users\black\music\song.mp3`,
			want:       "/C/Users/black/music/song.mp3",
			osSpecific: boolPtr(false),
		},
		{
			name:       "Linux Docker: Windows path with forward slash",
			input:      "C:/Users/black/music/song.mp3",
			want:       "/C/Users/black/music/song.mp3",
			osSpecific: boolPtr(false),
		},
		{
			name:       "Linux Docker: lowercase drive letter",
			input:      `d:\data\audio.wav`,
			want:       "/D/data/audio.wav",
			osSpecific: boolPtr(false),
		},

		// Linux native path tests (runtime.GOOS == "linux")
		{
			name:       "Linux: absolute path",
			input:      "/home/user/file.mp3",
			want:       "/home/user/file.mp3",
			osSpecific: boolPtr(false),
		},

		// Error cases
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
	}

	isWindows := runtime.GOOS == "windows"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if it's OS-specific and doesn't match current OS
			if tt.osSpecific != nil {
				if *tt.osSpecific && !isWindows {
					t.Skip("Test is Windows-specific")
				}
				if !*tt.osSpecific && isWindows {
					t.Skip("Test is Linux-specific")
				}
			}

			got, err := NormalizePath(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got.Path)
			assert.Equal(t, tt.wantIsRemote, got.IsRemote)
		})
	}
}

func TestConvertGoogleDriveURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "Standard Google Drive share URL",
			input: "https://drive.google.com/file/d/11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h/view?usp=sharing",
			want:  "https://drive.google.com/uc?export=download&id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h",
		},
		{
			name:  "Google Drive open URL",
			input: "https://drive.google.com/open?id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h",
			want:  "https://drive.google.com/uc?export=download&id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h",
		},
		{
			name:    "Invalid URL",
			input:   "https://example.com/file/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertGoogleDriveURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsGoogleDriveURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"Google Drive URL", "https://drive.google.com/file/d/11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h/view", true},
		{"Regular HTTP URL", "https://example.com/file.mp3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsGoogleDriveURL(tt.input))
		})
	}
}

func TestConvertDropboxURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "Standard Dropbox URL",
			input: "https://www.dropbox.com/scl/fi/test/file.mp3?dl=0",
			want:  "https://www.dropbox.com/scl/fi/test/file.mp3?dl=1",
		},
		{
			name:    "Invalid URL",
			input:   "https://example.com/file/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertDropboxURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsDropboxURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"Dropbox URL", "https://www.dropbox.com/scl/fi/test/file.mp3?dl=0", true},
		{"Regular HTTP URL", "https://example.com/file.mp3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDropboxURL(tt.input))
		})
	}
}

