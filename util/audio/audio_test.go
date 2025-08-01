package audio

import (
	"os"
	"testing"
)

func TestGetAudioInfoAndData(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		expectedFormat   string
		expectedDuration int
		expectedError    bool
		skipOnError      bool // Skip test if error occurs (for network-dependent tests)
	}{
		{
			name:             "WAV file test",
			path:             "../../testdata/wav-test.wav",
			expectedFormat:   "wav",
			expectedDuration: 10,
			expectedError:    false,
		},
		{
			name:             "MP3 file test",
			path:             "../../testdata/mp3-test.mp3",
			expectedFormat:   "mp3",
			expectedDuration: 10,
			expectedError:    false,
		},
		{
			name:             "OGG file test",
			path:             "../../testdata/ogg-test.ogg",
			expectedFormat:   "ogg",
			expectedDuration: 10,
			expectedError:    false,
		},
		{
			name:           "Unsupported format test",
			path:           "testdata/test.xyz",
			expectedFormat: "xyz",
			expectedError:  true,
		},
		{
			name:          "Non-existent file test",
			path:          "../../testdata/nonexistent.wav",
			expectedError: true,
		},
		{
			name:             "HTTP URL test (optional)",
			path:             "https://drive.google.com/file/d/11NG67v0jlYqh8T2oCh0nlL8z5WKNT26h/view?usp=drive_link",
			expectedFormat:   "mp3",
			expectedDuration: 0, // Duration may vary for online files
			expectedError:    false,
			skipOnError:      true, // Skip if network error
		},
	}

	// Create unsupported format test file
	if err := os.WriteFile("../../testdata/test.xyz", []byte{0x00}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, data, err := GetAudioInfoAndData(tt.path, false)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				if tt.skipOnError {
					t.Skipf("Skipping test due to error (likely network): %v", err)
					return
				}
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if info.Format != tt.expectedFormat {
				t.Errorf("Expected format %s but got %s", tt.expectedFormat, info.Format)
			}

			// Skip duration check for HTTP tests (duration may vary)
			if tt.expectedDuration > 0 && int(info.Duration) != tt.expectedDuration {
				t.Errorf("Expected duration %d but got %f", tt.expectedDuration, info.Duration)
			}

			t.Logf("Path: %s, Duration: %f seconds, Size: %d bytes, Format: %s, FileName: %s",
				tt.path, info.Duration, info.Size, info.Format, info.FileName)

			if info.Size <= 0 {
				t.Errorf("Expected size > 0 but got %d", info.Size)
			}

			// Verify that data size matches info size
			if len(data) != info.Size {
				t.Errorf("Expected data size %d but got %d", info.Size, len(data))
			}

			// Verify data is not empty
			if len(data) == 0 {
				t.Errorf("Expected data to be non-empty")
			}
		})
	}

	// Cleanup unsupported format test file
	os.Remove("../../testdata/test.xyz")
}
