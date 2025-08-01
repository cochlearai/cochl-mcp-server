package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
)

func Test_Analyze(t *testing.T) {
	tool, handlerFunc := AnalyzeAudio()

	assert.Equal(t, "analyze_audio", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "file_url")
	assert.Contains(t, tool.InputSchema.Properties, "with_caption")

	ctx := common.NewTestContext("analyze_audio")

	testCases := []struct {
		name            string
		args            map[string]any
		expectError     bool
		expectedErrMsg  string
		withCaption     bool
		expectedCaption string

		// Sense error simulation flags
		shouldCreateSessionError      bool
		shouldUploadChunkError        bool
		shouldGetInferenceResultError bool

		// Caption error simulation flag
		shouldCaptionError bool
	}{
		{
			name:           "invalid file_url parameter",
			args:           map[string]any{"file_url": 123},
			expectError:    true,
			expectedErrMsg: "missing or invalid",
		},
		{
			name:           "nonexistent file",
			args:           map[string]any{"file_url": getAbsPath(t, "../testdata/nonexistent.wav")},
			expectError:    true,
			expectedErrMsg: "failed to get audio info",
		},
		{
			name:                     "sense create session error",
			args:                     map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav")},
			expectError:              true,
			expectedErrMsg:           "failed to create session",
			shouldCreateSessionError: true,
		},
		{
			name:                   "sense upload chunk error",
			args:                   map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav")},
			expectError:            true,
			expectedErrMsg:         "failed to upload chunk",
			shouldUploadChunkError: true,
		},
		{
			name:                          "sense get inference result error",
			args:                          map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav")},
			expectError:                   true,
			expectedErrMsg:                "failed to get inference result",
			shouldGetInferenceResultError: true,
		},
		{
			name:        "successful sense only (with_caption default false)",
			args:        map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav")},
			expectError: false,
			withCaption: false,
		},
		{
			name:        "successful sense only (with_caption explicit false)",
			args:        map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav"), "with_caption": false},
			expectError: false,
			withCaption: false,
		},
		{
			name:            "successful sense and caption",
			args:            map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav"), "with_caption": true},
			expectError:     false,
			withCaption:     true,
			expectedCaption: "This is a mock caption.",
		},
		{
			name:               "sense success but caption error",
			args:               map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav"), "with_caption": true},
			expectError:        true,
			expectedErrMsg:     "caption audio failed",
			withCaption:        true,
			shouldCaptionError: true,
		},
		{
			name:                     "sense error and caption success",
			args:                     map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav"), "with_caption": true},
			expectError:              true,
			expectedErrMsg:           "sense audio failed",
			withCaption:              true,
			shouldCreateSessionError: true,
		},
		{
			name:                     "both sense and caption error",
			args:                     map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav"), "with_caption": true},
			expectError:              true,
			expectedErrMsg:           "sense audio failed",
			withCaption:              true,
			shouldCreateSessionError: true,
			shouldCaptionError:       true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Reset all mock errors
			client.ResetMockSenseErrors()
			client.ResetMockCaptionErrors()

			// Set sense errors
			client.SetShouldMockCreateSessionError(tc.shouldCreateSessionError)
			client.SetShouldMockUploadChunkError(tc.shouldUploadChunkError)
			client.SetShouldMockGetInferenceResultError(tc.shouldGetInferenceResultError)

			// Set caption errors
			client.SetShouldMockCaptionError(tc.shouldCaptionError)

			request := createMCPRequest(tc.args)

			result, _ := handlerFunc(ctx, request)

			if tc.expectError {
				assert.True(t, result.IsError)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			require.NotNil(t, result)
			assert.False(t, result.IsError)

			textContent := getTextResult(t, result)
			var analyzeResult AnalyzeResult
			err := json.Unmarshal([]byte(textContent.Text), &analyzeResult)
			require.NoError(t, err)

			// Sense result should always be present
			assert.NotNil(t, analyzeResult.Sense, "Sense result should always be present")

			// Caption result validation
			if tc.withCaption {
				assert.NotNil(t, analyzeResult.Caption, "Caption result should be present when with_caption is true")
			} else {
				// Caption should be nil or omitted when with_caption is false
				assert.Nil(t, analyzeResult.Caption, "Caption result should be nil when with_caption is false")
			}
		})
	}
}
