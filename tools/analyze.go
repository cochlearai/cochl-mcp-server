package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/audio"
)

const (
	// Timing and concurrency
	_inferenceTimeout         = 30 * time.Second
	_pollingInterval          = 2 * time.Second
	_durationForSingleCaption = 10 // 10 seconds
	_maxConcurrentChunks      = 5  // Limit concurrent processing

	// Caption processing
	_captionTempDirPattern = "audio-chunks-*"
	_captionChunkDirName   = "chunks"
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

	// Upload audio data (single upload for now, chunked upload for large files is handled by Caption API)
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

// CaptionChunkResult represents the result of processing a single caption chunk
type CaptionChunkResult struct {
	Index   int    // Chunk index for ordering
	Caption string // Caption result
	Error   error  // Error if processing failed
}

// RefinedCaptionResult represents a caption with time range information
type RefinedCaptionResult struct {
	Caption   string `json:"caption"`
	StartTime int    `json:"start_time"` // Start time in seconds
	EndTime   int    `json:"end_time"`   // End time in seconds
}

// analyzeCaption performs Caption API analysis
// For long audio files, splits into chunks and processes concurrently
// Returns []RefinedCaptionResult for consistency
func analyzeCaption(ctx context.Context, audioInfo *audio.AudioInfo, rawData []byte) (any, error) {
	captionClient := common.CaptionClientFromContext(ctx)
	if captionClient == nil {
		return nil, fmt.Errorf("caption client not found in context")
	}

	// If audio is short enough, process as a single file
	if audioInfo.Duration <= _durationForSingleCaption {
		captionResult, err := captionClient.Inference(audioInfo.Format, audioInfo.FileName, rawData)
		if err != nil {
			return nil, fmt.Errorf("failed to get caption: %w", err)
		}

		// Return as single-element array for consistency
		return []RefinedCaptionResult{
			{
				Caption:   captionResult.Caption,
				StartTime: 0,
				EndTime:   int(audioInfo.Duration),
			},
		}, nil
	}

	// Long audio: split and process concurrently
	return analyzeCaptionWithChunks(ctx, captionClient, audioInfo, rawData)
}

// analyzeCaptionWithChunks splits audio into chunks and processes them concurrently
func analyzeCaptionWithChunks(ctx context.Context, captionClient client.Caption, audioInfo *audio.AudioInfo, rawData []byte) (any, error) {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before processing: %w", err)
	}

	// Prepare chunk files
	chunkFiles, cleanup, err := prepareCaptionChunkFiles(rawData, audioInfo)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Process chunks concurrently
	captions, err := processCaptionChunksConcurrently(ctx, captionClient, audioInfo, chunkFiles)
	if err != nil {
		return nil, err
	}

	// Combine captions from all chunks into structured results
	refinedResults := combineCaptions(captions, audioInfo.Duration)
	return refinedResults, nil
}

// processCaptionChunksConcurrently processes multiple caption chunks concurrently with rate limiting
func processCaptionChunksConcurrently(ctx context.Context, captionClient client.Caption, audioInfo *audio.AudioInfo, chunkFiles []string) ([]string, error) {
	results := make(chan CaptionChunkResult, len(chunkFiles))
	semaphore := make(chan struct{}, _maxConcurrentChunks)
	var wg sync.WaitGroup

	for i, chunkPath := range chunkFiles {
		wg.Add(1)

		// Check context before starting new goroutine
		select {
		case <-ctx.Done():
			wg.Done()
			continue
		case semaphore <- struct{}{}: // Acquire semaphore
		}

		go func(index int, path string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			caption, err := processCaptionChunk(ctx, captionClient, audioInfo, path, index)
			results <- CaptionChunkResult{
				Index:   index,
				Caption: caption,
				Error:   err,
			}
		}(i, chunkPath)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results in order
	return collectCaptionChunkResults(results, len(chunkFiles))
}

// prepareCaptionChunkFiles prepares audio chunk files for caption processing
// Returns chunk file paths and a cleanup function
func prepareCaptionChunkFiles(rawData []byte, audioInfo *audio.AudioInfo) ([]string, func(), error) {
	// Create temporary directory for all temporary files (input file + chunks)
	tempDir, err := os.MkdirTemp("", _captionTempDirPattern)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup := func() { os.RemoveAll(tempDir) }

	// Save raw data to a temporary file for splitting (inside tempDir)
	tempInputFile, err := audio.SaveRawDataToTempFile(rawData, audioInfo.Format, tempDir)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to save temp input file: %w", err)
	}

	// Split audio into chunks using ffmpeg
	chunkOutputDir := filepath.Join(tempDir, _captionChunkDirName)
	chunkFiles, err := audio.SplitAudioIntoChunks(tempInputFile, chunkOutputDir, _durationForSingleCaption)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to split audio: %w", err)
	}

	return chunkFiles, cleanup, nil
}

// collectCaptionChunkResults collects caption results from the results channel and returns captions in order
func collectCaptionChunkResults(results chan CaptionChunkResult, expectedCount int) ([]string, error) {
	captions := make([]string, expectedCount)
	var errors []error

	for result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else {
			captions[result.Index] = result.Caption
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("failed to process %d chunk(s): %v", len(errors), errors)
	}

	return captions, nil
}

// processCaptionChunk processes a single audio chunk and returns its caption
func processCaptionChunk(ctx context.Context, captionClient client.Caption, audioInfo *audio.AudioInfo, chunkPath string, index int) (string, error) {
	// Check context before processing
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("context cancelled during chunk %d processing: %w", index, ctx.Err())
	default:
	}

	// Read chunk file
	chunkData, err := os.ReadFile(chunkPath)
	if err != nil {
		return "", fmt.Errorf("failed to read chunk %d: %w", index, err)
	}

	// Process chunk with Caption API
	chunkFileName := fmt.Sprintf("%s_chunk_%03d", audioInfo.FileName, index)
	captionResult, err := captionClient.Inference(audioInfo.Format, chunkFileName, chunkData)
	if err != nil {
		return "", fmt.Errorf("failed to infer chunk %d: %w", index, err)
	}

	return captionResult.Caption, nil
}

// combineCaptions merges multiple chunk captions into structured results with timing
func combineCaptions(captions []string, totalDuration float64) []RefinedCaptionResult {
	results := make([]RefinedCaptionResult, 0, len(captions))

	for i, caption := range captions {
		startTime := i * _durationForSingleCaption
		endTime := (i + 1) * _durationForSingleCaption

		// For the last chunk, use actual audio duration
		if i == len(captions)-1 {
			endTime = int(totalDuration)
		}

		results = append(results, RefinedCaptionResult{
			Caption:   caption,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return results
}
