package tools

import (
	"context"

	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/audio"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
		filePath := request.Params.Arguments["file_absolute_path"].(string)

		// normalize path
		normalizedPath, err := util.NormalizePath(filePath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid file path", err), nil
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

		caption, err := captionClient.Inference(audioInfo.Format, normalizedPath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get caption", err), nil
		}

		return mcp.NewToolResultText(caption.Caption), nil
	}

	return tool, handler
}
