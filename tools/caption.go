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
		mcp.WithDescription(
			"Analyze the environmental (background) sounds in an audio file and generate a concise natural language caption. "+
				"This caption infers and summarizes the likely situation or scene. "+
				"This tool does not transcribe speech or summarize the full content, "+
				"but instead focuses on the ambient sounds to describe the environment or context in which the audio was recorded. "+
				"Example: 'A woman speaks while a television plays in the background.'",
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
