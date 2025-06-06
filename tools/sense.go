package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/audio"
)

func Sense() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("analyze_audio",
		mcp.WithDescription(_analyzeAudioDesc),
		mcp.WithString(
			"file_absolute_path",
			mcp.Required(),
			mcp.Description(_fileAbsolutePathDesc),
		),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_absolute_path")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("missing or invalid 'file_absolute_path' parameter", err), nil
		}

		// normalize path
		normalizedPath, err := util.NormalizePath(filePath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to normalize file path", err), nil
		}

		audioInfo, err := audio.GetAudioInfo(normalizedPath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get audio info", err), nil
		}

		//TODO: check if duration is too long, if so, return error

		// Get raw audio data
		rawData, err := audio.GetRawAudioData(normalizedPath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get raw audio data", err), nil
		}

		cochlSenseClient := common.SenseClientFromContext(ctx)
		if cochlSenseClient == nil {
			return mcp.NewToolResultErrorFromErr("cochl sense client not found", nil), nil
		}

		resp, err := cochlSenseClient.CreateSession(
			audioInfo.FileName,
			audioInfo.Format,
			audioInfo.Duration,
			audioInfo.Size)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create session", err), nil
		}

		//TODO: if file is too large, upload in chunks
		_, err = cochlSenseClient.UploadChunk(
			resp.SessionID,
			resp.ChunkSequence,
			rawData)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to upload chunk", err), nil
		}

		var result *client.RespInferenceResult
		//TODO: set timeout
		for {
			time.Sleep(2 * time.Second)
			inferenceResult, err := cochlSenseClient.GetInferenceResult(resp.SessionID)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("failed to get inference result", err), nil
			}

			if inferenceResult.State == "done" {
				result = inferenceResult
				break
			}
		}
		jsonResult, err := json.Marshal(result.Data)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal inference result", err), nil
		}

		if err := cochlSenseClient.DeleteSession(resp.SessionID); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to delete session", err), nil
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	}

	return tool, handler
}
