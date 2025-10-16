package audio

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAudioInfoAndData(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		expectedFormat   string
		expectedDuration int
		shouldFail       bool
		skipOnError      bool
	}{
		{
			name:             "WAV file test",
			path:             "../../testdata/wav-test.wav",
			expectedFormat:   "wav",
			expectedDuration: 10,
		},
		{
			name:             "MP3 file test",
			path:             "../../testdata/mp3-test.mp3",
			expectedFormat:   "mp3",
			expectedDuration: 10,
		},
		{
			name:             "OGG file test",
			path:             "../../testdata/ogg-test.ogg",
			expectedFormat:   "ogg",
			expectedDuration: 10,
		},
		{
			name:           "Unsupported format test",
			path:           "testdata/test.xyz",
			expectedFormat: "xyz",
			shouldFail:     true,
		},
		{
			name:       "Non-existent file test",
			path:       "../../testdata/nonexistent.wav",
			shouldFail: true,
		},
		{
			name:             "HTTP URL test",
			path:             "https://freetestdata.com/wp-content/uploads/2021/09/Free_Test_Data_100KB_MP3.mp3",
			expectedFormat:   "mp3",
			expectedDuration: 3,    // Approximately 3 seconds
			skipOnError:      true, // Skip if network is unavailable
		},
	}

	// Create unsupported format test file
	assert.NoError(t, os.WriteFile("../../testdata/test.xyz", []byte{0x00}, 0644))
	defer os.Remove("../../testdata/test.xyz")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, data, err := GetAudioInfoAndData(tt.path, false)

			if tt.shouldFail {
				assert.Error(t, err)
				return
			}

			if tt.skipOnError && err != nil {
				t.Skipf("Skipping test (likely network): %v", err)
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedFormat, info.Format)
			assert.Greater(t, info.Size, 0)
			assert.NotEmpty(t, data)
			assert.Equal(t, info.Size, len(data))

			if tt.expectedDuration > 0 {
				assert.Equal(t, tt.expectedDuration, int(info.Duration))
			}
		})
	}
}

func TestSplitAudioIntoChunks(t *testing.T) {
	tests := []struct {
		name          string
		inputPath     string
		chunkDuration int
		shouldFail    bool
	}{
		{
			name:          "Split 10-second chunks",
			inputPath:     "../../testdata/split-test.mp3",
			chunkDuration: 10,
		},
		{
			name:          "Split 5-second chunks",
			inputPath:     "../../testdata/split-test.mp3",
			chunkDuration: 5,
		},
		{
			name:          "Non-existent file",
			inputPath:     "../../testdata/nonexistent.mp3",
			chunkDuration: 10,
			shouldFail:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := t.TempDir()

			chunks, err := SplitAudioIntoChunks(tt.inputPath, outputDir, tt.chunkDuration)

			if tt.shouldFail {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				t.Skipf("ffmpeg not installed: %v", err)
			}

			assert.NotEmpty(t, chunks)

			// Verify each chunk
			for i, chunkPath := range chunks {
				stat, err := os.Stat(chunkPath)
				assert.NoError(t, err, "Chunk %d should exist", i)
				assert.Greater(t, stat.Size(), int64(0), "Chunk %d should not be empty", i)

				// Check duration for non-last chunks
				if i < len(chunks)-1 {
					chunkDur, err := getAudioDurationWithFFProbe(chunkPath)
					if assert.NoError(t, err) {
						expectedDur := float64(tt.chunkDuration)
						assert.InDelta(t, expectedDur, chunkDur, expectedDur*0.1, "Chunk %d duration", i)
					}
				}
			}
		})
	}
}
