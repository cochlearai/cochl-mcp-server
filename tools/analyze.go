package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/audio"
)

type AnalyzeResult struct {
	Sense   any `json:"sense,omitempty" jsonschema:"Temporal segments with detected sounds/events and probability scores"`
	Caption any `json:"caption,omitempty" jsonschema:"Natural language caption summarizing the audio file"`
}

type AnalyzeAudioParams struct {
	FileUrl     string `json:"file_url" jsonschema:"Audio file URL or local path (MP3/WAV/OGG)"`
	WithCaption bool   `json:"with_caption" jsonschema:"Generate a natural language caption for the audio file (default: false)"`
}

func AnalyzeAudioTool() (tool *mcp.Tool, handler mcp.ToolHandlerFor[*AnalyzeAudioParams, *AnalyzeResult]) {
	tool = &mcp.Tool{
		Name:        "analyze_audio",
		Description: _analyzeAudioDescWithCaption,
	}

	handler = func(ctx context.Context, req *mcp.CallToolRequest, input *AnalyzeAudioParams) (*mcp.CallToolResult, *AnalyzeResult, error) {
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
