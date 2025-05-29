package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
)

func Test_Sense(t *testing.T) {
	tool, handlerFunc := Sense()

	assert.Equal(t, "analyze_audio", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "file_absolute_path")

	ctx := common.NewTestContext("sense")

	testCases := []struct {
		name           string
		args           map[string]any
		expectError    bool
		expectedErrMsg string

		shouldCreateSessionError      bool
		shouldUploadChunkError        bool
		shouldGetInferenceResultError bool
	}{
		{
			name:           "invalid file path",
			args:           map[string]any{"file_absolute_path": 123},
			expectError:    true,
			expectedErrMsg: "missing or invalid",
		},
		{
			name:           "no such file",
			args:           map[string]any{"file_absolute_path": getAbsPath("../util/audio/testdata/txt-test.txt")},
			expectError:    true,
			expectedErrMsg: "failed to get audio info",
		},
		{
			name:           "create session error",
			args:           map[string]any{"file_absolute_path": getAbsPath("../util/audio/testdata/wav-test.wav")},
			expectError:    true,
			expectedErrMsg: "failed to create session",

			shouldCreateSessionError: true,
		},
		{
			name:           "upload chunk error",
			args:           map[string]any{"file_absolute_path": getAbsPath("../util/audio/testdata/wav-test.wav")},
			expectError:    true,
			expectedErrMsg: "failed to upload chunk",

			shouldUploadChunkError: true,
		},
		{
			name:           "get inference result error",
			args:           map[string]any{"file_absolute_path": getAbsPath("../util/audio/testdata/wav-test.wav")},
			expectError:    true,
			expectedErrMsg: "failed to get inference result",

			shouldGetInferenceResultError: true,
		},
		{
			name:        "successful inference",
			args:        map[string]any{"file_absolute_path": getAbsPath("../util/audio/testdata/wav-test.wav")},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			client.ResetMockSenseErrors()
			client.SetShouldMockCreateSessionError(tc.shouldCreateSessionError)
			client.SetShouldMockUploadChunkError(tc.shouldUploadChunkError)
			client.SetShouldMockGetInferenceResultError(tc.shouldGetInferenceResultError)

			request := createMCPRequest(tc.args)

			result, _ := handlerFunc(ctx, request)

			if tc.expectError {
				assert.True(t, result.IsError)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			require.NotNil(t, result)

			textContent := getTextResult(t, result)
			var content []client.InferenceResult
			err := json.Unmarshal([]byte(textContent.Text), &content)
			require.NoError(t, err)

			assert.Greater(t, len(content), 0)
			for _, seg := range content {
				assert.Less(t, seg.StartTime, seg.EndTime)
				assert.Greater(t, len(seg.Tags), 0)
				for _, tag := range seg.Tags {
					assert.NotEmpty(t, tag.Name)
					assert.GreaterOrEqual(t, tag.Probability, 0.0)
					assert.LessOrEqual(t, tag.Probability, 1.0)
				}
			}
		})
	}
}
