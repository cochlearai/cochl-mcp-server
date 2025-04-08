package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAudioInfo(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		expectedFormat string
		expectedError  bool
	}{
		{
			name:           "WAV file test",
			filePath:       "testdata/wav-test.wav",
			expectedFormat: "wav",
			expectedError:  false,
		},
		{
			name:           "MP3 file test",
			filePath:       "testdata/mp3-test.mp3",
			expectedFormat: "mp3",
			expectedError:  false,
		},
		{
			name:           "OGG file test",
			filePath:       "testdata/ogg-test.ogg",
			expectedFormat: "ogg",
			expectedError:  false,
		},
		{
			name:           "Unsupported format test",
			filePath:       "testdata/test.xyz",
			expectedFormat: "xyz",
			expectedError:  true,
		},
		{
			name:          "Non-existent file test",
			filePath:      "testdata/nonexistent.wav",
			expectedError: true,
		},
	}

	// Create unsupported format test file
	if err := os.WriteFile("testdata/test.xyz", []byte{0x00}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := GetAudioInfo(tt.filePath)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if info.Format != tt.expectedFormat {
				t.Errorf("Expected format %s but got %s", tt.expectedFormat, info.Format)
			}

			if info.Duration <= 0 {
				t.Errorf("Expected duration > 0 but got %f", info.Duration)
			}

			t.Logf("Duration for %s: %f seconds", tt.filePath, info.Duration)

			if info.Size <= 0 {
				t.Errorf("Expected size > 0 but got %d", info.Size)
			}

			expectedFileName := filepath.Base(tt.filePath)
			if info.FileName != expectedFileName {
				t.Errorf("Expected filename %s but got %s", expectedFileName, info.FileName)
			}
		})
	}

	// Cleanup unsupported format test file
	os.Remove("testdata/test.xyz")
}
