package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"cochl-mcp-server/client"
	"cochl-mcp-server/util/audio"
)

func CochlSenseTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath := request.Params.Arguments["file_absolute_path"].(string)

	audioInfo, err := audio.GetAudioInfo(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get audio info: %v", err)
	}

	// Get raw audio data
	rawData, err := audio.GetRawAudioData(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get raw audio data: %v", err)
	}

	cochlSenseClient := client.CochlSense()

	resp, err := cochlSenseClient.CreateSession(
		audioInfo.FileName,
		audioInfo.Format,
		audioInfo.Duration,
		audioInfo.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	//TODO: if file is too large, upload in chunks
	_, err = cochlSenseClient.UploadChunk(
		resp.SessionID,
		resp.ChunkSequence,
		rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to upload chunk: %v", err)
	}

	var result *client.RespInferenceResult
	//TODO: set timeout
	for {
		time.Sleep(2 * time.Second)
		inferenceResult, err := cochlSenseClient.GetInferenceResult(resp.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get inference result: %v", err)
		}

		if inferenceResult.State == "done" {
			result = inferenceResult
			break
		}
	}
	jsonResult, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inference result: %v", err)
	}

	if err := cochlSenseClient.DeleteSession(resp.SessionID); err != nil {
		return nil, fmt.Errorf("failed to delete session: %v", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}
