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
