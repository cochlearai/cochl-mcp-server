package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
)

func Test_Analyze(t *testing.T) {
	tool, handlerFunc := AnalyzeAudioTool()

	assert.Equal(t, "analyze_audio", tool.Name)
	assert.NotEmpty(t, tool.Description)

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
			expectedErrMsg: "cannot unmarshal number",
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
			expectedErrMsg:     "caption analysis",
			withCaption:        true,
			shouldCaptionError: true,
		},
		{
			name:                     "sense error and caption success",
			args:                     map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav"), "with_caption": true},
			expectError:              true,
			expectedErrMsg:           "sense analysis",
			withCaption:              true,
			shouldCreateSessionError: true,
		},
		{
			name:                     "both sense and caption error",
			args:                     map[string]any{"file_url": getAbsPath(t, "../testdata/wav-test.wav"), "with_caption": true},
			expectError:              true,
			expectedErrMsg:           "analysis failed with 2 error(s)",
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

			params, parseErr := parseParams(t, tc.args, &AnalyzeAudioInput{})

			// If parameter parsing fails, check if we expected an error
			if parseErr != nil {
				if tc.expectError {
					assert.Contains(t, parseErr.Error(), tc.expectedErrMsg)
					return
				}
				t.Fatalf("unexpected parse error: %v", parseErr)
			}

			_, resultData, err := handlerFunc(ctx, nil, params)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resultData)

			// Sense result should always be present
			assert.NotNil(t, resultData.Senses, "Sense result should always be present")

			// Caption result validation
			if tc.withCaption {
				assert.NotNil(t, resultData.Captions, "Caption result should be present when with_caption is true")
			} else {
				// Caption should be nil or omitted when with_caption is false
				assert.Nil(t, resultData.Captions, "Caption result should be nil when with_caption is false")
			}
		})
	}
}
