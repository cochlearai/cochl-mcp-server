package util

import (
	"path/filepath"
	"runtime"
	"testing"
)

// boolPtr returns a pointer to the given bool value
func boolPtr(b bool) *bool {
	return &b
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       string
		wantErr    bool
		osSpecific *bool // nil = run on all OS, true = Windows only, false = Unix only
	}{
		{
			name:       "Windows drive letter path",
			input:      "C:/Users/test/file.mp3",
			want:       filepath.FromSlash("C:/Users/test/file.mp3"),
			osSpecific: boolPtr(true),
		},
		{
			name:       "Windows path with slash prefix",
			input:      "/c:/Users/test/file.mp3",
			want:       filepath.FromSlash("c:/Users/test/file.mp3"),
			osSpecific: boolPtr(true),
		},
		{
			name:       "Unix absolute path",
			input:      "/home/user/file.mp3",
			want:       "/home/user/file.mp3",
			osSpecific: boolPtr(false),
		},
		{
			name:       "Clean double slashes on Unix",
			input:      "/home//user///file.mp3",
			want:       "/home/user/file.mp3",
			osSpecific: boolPtr(false),
		},
		{
			name:       "Clean double slashes on Windows",
			input:      "C://Users//test///file.mp3",
			want:       filepath.FromSlash("C:/Users/test/file.mp3"),
			osSpecific: boolPtr(true),
		},
		{
			name:       "Relative path should fail",
			input:      "relative/path/file.mp3",
			wantErr:    true,
			osSpecific: nil, // Run on all OS
		},
		{
			name:       "Empty path should fail",
			input:      "",
			wantErr:    true,
			osSpecific: nil, // Run on all OS
		},
		{
			name:       "URL encoded Windows path",
			input:      "/c%3A/Users/test/file%20name.mp3",
			want:       filepath.FromSlash("c:/Users/test/file name.mp3"),
			osSpecific: boolPtr(true),
		},
		{
			name:       "URL encoded Unix path with special characters",
			input:      "/home/user/한글%20파일.mp3",
			want:       "/home/user/한글 파일.mp3",
			osSpecific: boolPtr(false),
		},
		{
			name:       "Invalid URL encoded path should fail",
			input:      "/home/user/%XX",
			wantErr:    true,
			osSpecific: nil, // Run on all OS
		},
	}

	isWindows := runtime.GOOS == "windows"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if it's OS-specific and doesn't match current OS
			if tt.osSpecific != nil {
				if *tt.osSpecific && !isWindows {
					t.Skip("Test is Windows-specific")
					return
				}
				if !*tt.osSpecific && isWindows {
					t.Skip("Test is Unix-specific")
					return
				}
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

			if got.Path != tt.want {
				t.Errorf("got %q, want %q", got.Path, tt.want)
			}
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
			name:  "Google Drive share URL with drive_link parameter",
			input: "https://drive.google.com/file/d/11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h/view?usp=drive_link",
			want:  "https://drive.google.com/uc?export=download&id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h",
		},
		{
			name:  "Google Drive open URL",
			input: "https://drive.google.com/open?id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h",
			want:  "https://drive.google.com/uc?export=download&id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h",
		},
		{
			name:    "Invalid Google Drive URL",
			input:   "https://example.com/file/invalid",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:  "Direct download URL with different parameter order",
			input: "https://drive.google.com/uc?id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h&export=download",
			want:  "https://drive.google.com/uc?id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h&export=download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertGoogleDriveURL(tt.input)
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

func TestIsGoogleDriveURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Google Drive URL",
			input: "https://drive.google.com/file/d/11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h/view?usp=drive_link",
			want:  true,
		},
		{
			name:  "Google Drive URL with drive_link parameter",
			input: "https://drive.google.com/uc?export=download&id=11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h",
			want:  true,
		},
		{
			name:  "Regular HTTP URL",
			input: "https://example.com/file.mp3",
			want:  false,
		},
		{
			name:  "Empty URL",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGoogleDriveURL(tt.input)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
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
			name:  "Standard Dropbox share URL",
			input: "https://www.dropbox.com/scl/fi/ax3y94mznux4cfgkieqs9/Sample-1.mp3?rlkey=r1v54ipn0iiv9waev04t4pmce&st=o023ka80&dl=0",
			want:  "https://www.dropbox.com/scl/fi/ax3y94mznux4cfgkieqs9/Sample-1.mp3?rlkey=r1v54ipn0iiv9waev04t4pmce&st=o023ka80&dl=1",
		},
		{
			name:  "Dropbox URL with dl=1 already",
			input: "https://www.dropbox.com/scl/fi/ax3y94mznux4cfgkieqs9/Sample-1.mp3?rlkey=r1v54ipn0iiv9waev04t4pmce&st=o023ka80&dl=1",
			want:  "https://www.dropbox.com/scl/fi/ax3y94mznux4cfgkieqs9/Sample-1.mp3?rlkey=r1v54ipn0iiv9waev04t4pmce&st=o023ka80&dl=1",
		},
		{
			name:    "Invalid Dropbox URL format",
			input:   "https://example.com/file/invalid",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Dropbox URL without dl parameter",
			input:   "https://www.dropbox.com/s/abcd1234/file.mp3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertDropboxURL(tt.input)
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

func TestIsDropboxURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Dropbox URL",
			input: "https://www.dropbox.com/scl/fi/ax3y94mznux4cfgkieqs9/Sample-1.mp3?rlkey=r1v54ipn0iiv9waev04t4pmce&st=o023ka80&dl=0",
			want:  true,
		},
		{
			name:  "Dropbox direct download URL",
			input: "https://www.dropbox.com/scl/fi/ax3y94mznux4cfgkieqs9/Sample-1.mp3?rlkey=r1v54ipn0iiv9waev04t4pmce&st=o023ka80&dl=1",
			want:  true,
		},
		{
			name:  "Regular HTTP URL",
			input: "https://example.com/file.mp3",
			want:  false,
		},
		{
			name:  "Google Drive URL",
			input: "https://drive.google.com/file/d/11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h/view",
			want:  false,
		},
		{
			name:  "Empty URL",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDropboxURL(tt.input)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
