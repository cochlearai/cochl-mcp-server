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

const (
	_inferenceTimeout = 30 * time.Second
	_pollingInterval  = 2 * time.Second
)

type AnalyzeAudioOutput struct {
	Sense   any `json:"sense,omitempty" jsonschema:"Temporal segments with detected sounds/events and probability scores"`
	Caption any `json:"caption,omitempty" jsonschema:"Natural language caption summarizing the audio file"`
}

type AnalyzeAudioInput struct {
	FileUrl     string `json:"file_url" jsonschema:"Audio file URL or local path (MP3/WAV/OGG)"`
	WithCaption bool   `json:"with_caption" jsonschema:"Generate a natural language caption for the audio file (default: false)"`
}

func AnalyzeAudioTool() (tool *mcp.Tool, handler mcp.ToolHandlerFor[*AnalyzeAudioInput, *AnalyzeAudioOutput]) {
	tool = &mcp.Tool{
		Name:        "analyze_audio",
		Description: _analyzeAudioDescWithCaption,
	}

	handler = func(ctx context.Context, req *mcp.CallToolRequest, input *AnalyzeAudioInput) (*mcp.CallToolResult, *AnalyzeAudioOutput, error) {
		// Validate input
		if err := validateInput(input); err != nil {
			return nil, nil, err
		}

		// Prepare audio data
		audioInfo, rawData, err := prepareAudioData(input.FileUrl)
		if err != nil {
			return nil, nil, err
		}

		// Run Sense and Caption analysis concurrently
		result, err := runConcurrentAnalysis(ctx, audioInfo, rawData, input.WithCaption)
		if err != nil {
			return nil, nil, err
		}

		return nil, result, nil
	}

	return tool, handler
}

// validateInput validates the input parameters
func validateInput(input *AnalyzeAudioInput) error {
	if input == nil {
		return fmt.Errorf("input is required")
	}
	if input.FileUrl == "" {
		return fmt.Errorf("file_url is required")
	}
	return nil
}

// prepareAudioData normalizes the path and loads audio data
func prepareAudioData(fileUrl string) (*audio.AudioInfo, []byte, error) {
	normalizedPath, err := util.NormalizePath(fileUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to normalize file path: %w", err)
	}

	audioInfo, rawData, err := audio.GetAudioInfoAndData(normalizedPath.Path, normalizedPath.IsRemote)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get audio info and data: %w", err)
	}

	return audioInfo, rawData, nil
}

// runConcurrentAnalysis runs Sense and optionally Caption analysis concurrently
func runConcurrentAnalysis(ctx context.Context, audioInfo *audio.AudioInfo, rawData []byte, withCaption bool) (*AnalyzeAudioOutput, error) {
	var (
		wg     sync.WaitGroup
		result AnalyzeAudioOutput
		mu     sync.Mutex // Protect concurrent writes to result
		errors []error
		errMu  sync.Mutex // Protect concurrent writes to errors
	)

	// Run Sense analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		senseData, err := analyzeSense(ctx, audioInfo, rawData)
		if err != nil {
			errMu.Lock()
			errors = append(errors, fmt.Errorf("sense analysis: %w", err))
			errMu.Unlock()
			return
		}
		mu.Lock()
		result.Sense = senseData
		mu.Unlock()
	}()

	// Run Caption analysis if requested
	if withCaption {
		wg.Add(1)
		go func() {
			defer wg.Done()
			caption, err := analyzeCaption(ctx, audioInfo, rawData)
			if err != nil {
				errMu.Lock()
				errors = append(errors, fmt.Errorf("caption analysis: %w", err))
				errMu.Unlock()
				return
			}
			mu.Lock()
			result.Caption = caption
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		return nil, fmt.Errorf("analysis failed with %d error(s): %v", len(errors), errors)
	}

	return &result, nil
}

// analyzeSense performs Sense API analysis
func analyzeSense(ctx context.Context, audioInfo *audio.AudioInfo, rawData []byte) (any, error) {
	senseClient := common.SenseClientFromContext(ctx)
	if senseClient == nil {
		return nil, fmt.Errorf("cochl sense client not found in context")
	}

	// Create session
	session, err := senseClient.CreateSession(
		audioInfo.FileName,
		audioInfo.Format,
		audioInfo.Duration,
		audioInfo.Size,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Upload audio data
	// TODO: if file is too large, upload in chunks
	if _, err := senseClient.UploadChunk(session.SessionID, session.ChunkSequence, rawData); err != nil {
		return nil, fmt.Errorf("failed to upload chunk: %w", err)
	}

	// Wait for inference result
	senseData, err := waitForInferenceResult(senseClient, session.SessionID)
	if err != nil {
		return nil, err
	}

	return senseData, nil
}

// waitForInferenceResult polls for inference result until done or timeout
func waitForInferenceResult(senseClient client.Sense, sessionID string) (any, error) {
	timeout := time.After(_inferenceTimeout)
	ticker := time.NewTicker(_pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for inference result after %v", _inferenceTimeout)
		case <-ticker.C:
			result, err := senseClient.GetInferenceResult(sessionID)
			if err != nil {
				return nil, fmt.Errorf("failed to get inference result: %w", err)
			}

			if result.State == "done" {
				return result.Data, nil
			}
			// Continue polling if not done yet
		}
	}
}

// analyzeCaption performs Caption API analysis
func analyzeCaption(ctx context.Context, audioInfo *audio.AudioInfo, rawData []byte) (any, error) {
	captionClient := common.CaptionClientFromContext(ctx)
	if captionClient == nil {
		return nil, fmt.Errorf("caption client not found in context")
	}

	captionResult, err := captionClient.Inference(audioInfo.Format, audioInfo.FileName, rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to get caption: %w", err)
	}

	return captionResult.Caption, nil
}
