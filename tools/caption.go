package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/audio"
)

func Caption() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("audio_caption",
		mcp.WithDescription(_audioCaptionDesc),
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

		captionClient := common.CaptionClientFromContext(ctx)
		if captionClient == nil {
			return mcp.NewToolResultErrorFromErr("caption client not found", nil), nil
		}

		result, err := captionClient.Inference(audioInfo.Format, normalizedPath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get caption", err), nil
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal inference result", err), nil
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	}

	return tool, handler
}
