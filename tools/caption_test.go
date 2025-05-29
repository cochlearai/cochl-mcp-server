package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
)

func Test_Caption(t *testing.T) {
	tool, handlerFunc := Caption()

	assert.Equal(t, "audio_caption", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "file_absolute_path")

	ctx := common.NewTestContext("caption")

	testCases := []struct {
		name            string
		args            map[string]any
		expectError     bool
		expectedErrMsg  string
		expectedCaption string
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
			name:            "successful caption",
			args:            map[string]any{"file_absolute_path": getAbsPath("../util/audio/testdata/wav-test.wav")},
			expectError:     false,
			expectedCaption: "This is a mock caption.",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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
			var content client.RespCaptionInference
			err := json.Unmarshal([]byte(textContent.Text), &content)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedCaption, content.Caption)
		})
	}
}
