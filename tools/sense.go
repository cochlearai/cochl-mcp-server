package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"cochl-mcp-server/client"
	"cochl-mcp-server/common"
	"cochl-mcp-server/util"
	"cochl-mcp-server/util/audio"
)

func Sense() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("analyze_audio",
		mcp.WithDescription(
			"Analyze an audio file and return detected sounds, events, and their probabilities. "+
				"The analysis includes:\n"+
				"  - Temporal segments with start and end times\n"+
				"  - Tags for each segment indicating the detected sounds/events\n"+
				"  - Probability scores for each detected tag",
		),
		mcp.WithString(
			"file_absolute_path",
			mcp.Required(),
			mcp.Description(
				"Please provide the absolute path to the file.\n"+
					"Avoid using URL-encoded characters.",
			),
		),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath := request.Params.Arguments["file_absolute_path"].(string)

		// normalize path
		normalizedPath, err := util.NormalizePath(filePath)
		if err != nil {
			return nil, fmt.Errorf("invalid file path: %v", err)
		}
		filePath = normalizedPath

		audioInfo, err := audio.GetAudioInfo(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get audio info: %v", err)
		}

		// Get raw audio data
		rawData, err := audio.GetRawAudioData(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get raw audio data: %v", err)
		}

		cochlSenseClient := common.CochlSenseClientFromContext(ctx)
		if cochlSenseClient == nil {
			return nil, fmt.Errorf("cochl sense client not found")
		}

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
		jsonResult, err := json.Marshal(result.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal inference result: %v", err)
		}

		if err := cochlSenseClient.DeleteSession(resp.SessionID); err != nil {
			return nil, fmt.Errorf("failed to delete session: %v", err)
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	}

	return tool, handler
}
