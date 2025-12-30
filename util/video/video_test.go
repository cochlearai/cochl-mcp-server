package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsVideoFormat(t *testing.T) {
	testCases := []struct {
		format   string
		expected bool
	}{
		{"mp4", true},
		{"MP4", true},
		{"webm", true},
		{"avi", true},
		{"AVI", true},
		{"mp3", false},
		{"wav", false},
		{"ogg", false},
		{"", false},
		{"mkv", false},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			result := IsVideoFormat(tc.format)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCalculateUniformTimestamps(t *testing.T) {
	testCases := []struct {
		name     string
		duration float64
		count    int
		expected []float64
	}{
		{
			name:     "10 second video with 8 frames",
			duration: 10.0,
			count:    8,
			expected: []float64{0, 1.4285714285714286, 2.857142857142857, 4.285714285714286, 5.714285714285714, 7.142857142857143, 8.571428571428571, 10.0},
		},
		{
			name:     "60 second video with 8 frames",
			duration: 60.0,
			count:    8,
			expected: []float64{0, 8.571428571428571, 17.142857142857142, 25.714285714285715, 34.285714285714285, 42.857142857142854, 51.42857142857143, 60.0},
		},
		{
			name:     "single frame",
			duration: 10.0,
			count:    1,
			expected: []float64{0},
		},
		{
			name:     "two frames",
			duration: 10.0,
			count:    2,
			expected: []float64{0, 10.0},
		},
		{
			name:     "zero count",
			duration: 10.0,
			count:    0,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateUniformTimestamps(tc.duration, tc.count)
			assert.Equal(t, len(tc.expected), len(result))
			for i := range result {
				assert.InDelta(t, tc.expected[i], result[i], 0.001)
			}
		})
	}
}

func TestDetectMediaType(t *testing.T) {
	testCases := []struct {
		fileUrl  string
		expected string
	}{
		{"/path/to/video.mp4", "video"},
		{"/path/to/video.MP4", "video"},
		{"/path/to/video.webm", "video"},
		{"/path/to/video.avi", "video"},
		{"/path/to/audio.mp3", "audio"},
		{"/path/to/audio.wav", "audio"},
		{"/path/to/audio.ogg", "audio"},
		{"https://example.com/file.mp4", "video"},
		{"https://example.com/file.mp3", "audio"},
	}

	for _, tc := range testCases {
		t.Run(tc.fileUrl, func(t *testing.T) {
			result := detectMediaType(tc.fileUrl)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// detectMediaType helper for testing (same logic as in tools/analyze.go)
func detectMediaType(fileUrl string) string {
	ext := ""
	for i := len(fileUrl) - 1; i >= 0; i-- {
		if fileUrl[i] == '.' {
			ext = fileUrl[i+1:]
			break
		}
	}

	// Convert to lowercase for comparison
	for i := 0; i < len(ext); i++ {
		if ext[i] >= 'A' && ext[i] <= 'Z' {
			ext = string(append([]byte(ext[:i]), byte(ext[i]+32))) + ext[i+1:]
		}
	}

	if IsVideoFormat(ext) {
		return "video"
	}
	return "audio"
}
