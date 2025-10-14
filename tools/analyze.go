package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/audio"

	mcpv2 "github.com/modelcontextprotocol/go-sdk/mcp"
)

type AnalyzeResult struct {
	Sense   any `json:"sense,omitempty" jsonschema:"Temporal segments with detected sounds/events and probability scores"`
	Caption any `json:"caption,omitempty" jsonschema:"Natural language caption summarizing the audio file"`
}

type AnalyzeAudioParams struct {
	FileUrl     string `json:"file_url" jsonschema:"Audio file URL or local path (MP3/WAV/OGG)"`
	WithCaption bool   `json:"with_caption" jsonschema:"Generate a natural language caption for the audio file (default: false)"`
}

func AnalyzeAudioToolv2() (tool *mcpv2.Tool, handler mcpv2.ToolHandlerFor[*AnalyzeAudioParams, *AnalyzeResult]) {
	tool = &mcpv2.Tool{
		Name:        "analyze_audio",
		Description: _analyzeAudioDescWithCaption,
	}

	handler = func(ctx context.Context, req *mcpv2.CallToolRequest, input *AnalyzeAudioParams) (*mcpv2.CallToolResult, *AnalyzeResult, error) {
		if input == nil {
			return nil, nil, fmt.Errorf("input is required")
		}

		if input.FileUrl == "" {
			return nil, nil, fmt.Errorf("file_url is required")
		}

		// normalize path
		normalizedPath, err := util.NormalizePath(input.FileUrl)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to normalize file path: %w", err)
		}

		// Get both audio info and raw data in a single file read
		audioInfo, rawData, err := audio.GetAudioInfoAndData(normalizedPath.Path, normalizedPath.IsRemote)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get audio info and data: %w", err)
		}

		var (
			result AnalyzeResult

			wg         sync.WaitGroup
			senseErr   error
			captionErr error
		)

		wg.Add(1)
		go func() {
			defer wg.Done()

			cochlSenseClient := common.SenseClientFromContext(ctx)
			if cochlSenseClient == nil {
				senseErr = fmt.Errorf("cochl sense client not found")
				return
			}

			resp, err := cochlSenseClient.CreateSession(
				audioInfo.FileName,
				audioInfo.Format,
				audioInfo.Duration,
				audioInfo.Size)
			if err != nil {
				senseErr = fmt.Errorf("failed to create session: %w", err)
				return
			}

			//TODO: if file is too large, upload in chunks
			_, err = cochlSenseClient.UploadChunk(
				resp.SessionID,
				resp.ChunkSequence,
				rawData)
			if err != nil {
				senseErr = fmt.Errorf("failed to upload chunk: %w", err)
				return
			}

			var senseResult *client.RespInferenceResult
			// Set 30 second timeout
			timeout := time.After(30 * time.Second)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					senseErr = fmt.Errorf("timeout waiting for inference result after 30 seconds")
					return
				case <-ticker.C:
					inferenceResult, err := cochlSenseClient.GetInferenceResult(resp.SessionID)
					if err != nil {
						senseErr = fmt.Errorf("failed to get inference result: %w", err)
						return
					}

					if inferenceResult.State == "done" {
						senseResult = inferenceResult
						result.Sense = senseResult.Data
						return
					}
				}
			}
		}()

		if input.WithCaption {
			wg.Add(1)
			go func() {
				defer wg.Done()
				captionClient := common.CaptionClientFromContext(ctx)
				if captionClient == nil {
					captionErr = fmt.Errorf("caption client not found")
					return
				}

				captionResult, err := captionClient.Inference(audioInfo.Format, audioInfo.FileName, rawData)
				if err != nil {
					captionErr = fmt.Errorf("failed to get caption: %w", err)
					return
				}
				result.Caption = captionResult.Caption
			}()
		}

		wg.Wait()

		if senseErr != nil || captionErr != nil {
			return nil, nil, fmt.Errorf("sense audio failed: %v, caption audio failed: %v", senseErr, captionErr)
		}
		return nil, &result, nil
	}

	return tool, handler
}

func AnalyzeAudio() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("analyze_audio",
		mcp.WithDescription(_analyzeAudioDescWithCaption),
		mcp.WithString(
			"file_url",
			mcp.Required(),
			mcp.Description(_fileUrlDesc),
		),
		mcp.WithBoolean(
			"with_caption",
			mcp.Description(_withCaptionDesc),
			mcp.DefaultBool(false),
		),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileUrl, err := request.RequireString("file_url")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("missing or invalid 'file_url' parameter", err), nil
		}

		// normalize path
		normalizedPath, err := util.NormalizePath(fileUrl)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to normalize file path", err), nil
		}

		// Get both audio info and raw data in a single file read
		audioInfo, rawData, err := audio.GetAudioInfoAndData(normalizedPath.Path, normalizedPath.IsRemote)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get audio info and data", err), nil
		}

		withCaption, _ := request.RequireBool("with_caption")

		var (
			result AnalyzeResult

			wg         sync.WaitGroup
			senseErr   error
			captionErr error
		)

		wg.Add(1)
		go func() {
			defer wg.Done()

			cochlSenseClient := common.SenseClientFromContext(ctx)
			if cochlSenseClient == nil {
				senseErr = fmt.Errorf("cochl sense client not found")
				return
			}

			resp, err := cochlSenseClient.CreateSession(
				audioInfo.FileName,
				audioInfo.Format,
				audioInfo.Duration,
				audioInfo.Size)
			if err != nil {
				senseErr = fmt.Errorf("failed to create session: %w", err)
				return
			}

			//TODO: if file is too large, upload in chunks
			_, err = cochlSenseClient.UploadChunk(
				resp.SessionID,
				resp.ChunkSequence,
				rawData)
			if err != nil {
				senseErr = fmt.Errorf("failed to upload chunk: %w", err)
				return
			}

			var senseResult *client.RespInferenceResult
			// Set 30 second timeout
			timeout := time.After(30 * time.Second)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					senseErr = fmt.Errorf("timeout waiting for inference result after 30 seconds")
					return
				case <-ticker.C:
					inferenceResult, err := cochlSenseClient.GetInferenceResult(resp.SessionID)
					if err != nil {
						senseErr = fmt.Errorf("failed to get inference result: %w", err)
						return
					}

					if inferenceResult.State == "done" {
						senseResult = inferenceResult
						result.Sense = senseResult.Data
						return
					}
				}
			}
		}()

		if withCaption {
			wg.Add(1)
			go func() {
				defer wg.Done()
				captionClient := common.CaptionClientFromContext(ctx)
				if captionClient == nil {
					captionErr = fmt.Errorf("caption client not found")
					return
				}

				captionResult, err := captionClient.Inference(audioInfo.Format, audioInfo.FileName, rawData)
				if err != nil {
					captionErr = fmt.Errorf("failed to get caption: %w", err)
					return
				}
				result.Caption = captionResult.Caption
			}()
		}

		wg.Wait()

		if senseErr != nil || captionErr != nil {
			return mcp.NewToolResultErrorf("sense audio failed: %v, caption audio failed: %v", senseErr, captionErr), nil
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal analyze result", err), nil
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	}

	return tool, handler
}
