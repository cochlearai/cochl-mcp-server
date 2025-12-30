package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cochlearai/cochl-mcp-server/client"
	"github.com/cochlearai/cochl-mcp-server/common"
	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/audio"
	"github.com/cochlearai/cochl-mcp-server/util/video"
)

const (
	_inferenceTimeout = 30 * time.Second
	_pollingInterval  = 2 * time.Second
	_defaultMaxFrames = 8
)

// AnalyzeOutput represents the unified output for both audio and video analysis
type AnalyzeOutput struct {
	MediaType    string `json:"media_type" jsonschema:"Type of media analyzed: 'audio' or 'video'"`
	Sense        any    `json:"sense,omitempty" jsonschema:"Temporal segments with detected sounds/events and probability scores"`
	Caption      any    `json:"caption,omitempty" jsonschema:"Natural language caption summarizing the audio"`
	VideoCaption any    `json:"video_caption,omitempty" jsonschema:"Video frame analysis results from visual AI"`
}

// AnalyzeInput represents the input for media analysis
type AnalyzeInput struct {
	FileUrl     string `json:"file_url" jsonschema:"Media file URL or local path (Audio: MP3/WAV/OGG, Video: MP4/WebM/AVI)"`
	WithCaption bool   `json:"with_caption" jsonschema:"Generate a natural language caption for the audio (default: false)"`
	MaxFrames   int    `json:"max_frames,omitempty" jsonschema:"Maximum number of frames to extract for video analysis, uniformly sampled (default: 8, max: 16)"`
}

func AnalyzeMediaTool() (tool *mcp.Tool, handler mcp.ToolHandlerFor[*AnalyzeInput, *AnalyzeOutput]) {
	tool = &mcp.Tool{
		Name:        "analyze_media",
		Description: _analyzeMediaDesc,
	}

	handler = func(ctx context.Context, req *mcp.CallToolRequest, input *AnalyzeInput) (*mcp.CallToolResult, *AnalyzeOutput, error) {
		// Validate input
		if err := validateInput(input); err != nil {
			return nil, nil, err
		}

		// Determine media type from file extension
		mediaType := detectMediaType(input.FileUrl)

		var result *AnalyzeOutput
		var err error

		switch mediaType {
		case "video":
			result, err = analyzeVideo(ctx, input)
		case "audio":
			result, err = analyzeAudioFile(ctx, input)
		default:
			return nil, nil, fmt.Errorf("unsupported media format")
		}

		if err != nil {
			return nil, nil, err
		}

		return nil, result, nil
	}

	return tool, handler
}

// detectMediaType determines if the file is audio or video based on extension
func detectMediaType(fileUrl string) string {
	ext := strings.ToLower(filepath.Ext(fileUrl))
	if ext != "" {
		ext = ext[1:] // Remove the dot
	}

	if video.IsVideoFormat(ext) {
		return "video"
	}
	return "audio"
}

// analyzeAudioFile handles audio-only analysis
func analyzeAudioFile(ctx context.Context, input *AnalyzeInput) (*AnalyzeOutput, error) {
	// Prepare audio data
	audioInfo, rawData, err := prepareAudioData(input.FileUrl)
	if err != nil {
		return nil, err
	}

	// Run Sense and Caption analysis concurrently
	result, err := runConcurrentAudioAnalysis(ctx, audioInfo, rawData, input.WithCaption)
	if err != nil {
		return nil, err
	}

	result.MediaType = "audio"
	return result, nil
}

// analyzeVideo handles video analysis with audio extraction
func analyzeVideo(ctx context.Context, input *AnalyzeInput) (*AnalyzeOutput, error) {
	// Normalize and load video data
	normalizedPath, err := util.NormalizePath(input.FileUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize file path: %w", err)
	}

	videoInfo, videoData, err := video.GetVideoInfoAndData(normalizedPath.Path, normalizedPath.IsRemote)
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	maxFrames := input.MaxFrames
	if maxFrames <= 0 {
		maxFrames = _defaultMaxFrames
	}

	// Run all analyses concurrently
	result, err := runConcurrentVideoAnalysis(ctx, videoInfo, videoData, input.WithCaption, maxFrames)
	if err != nil {
		return nil, err
	}

	result.MediaType = "video"
	return result, nil
}

// runConcurrentVideoAnalysis runs audio and video analysis concurrently
func runConcurrentVideoAnalysis(ctx context.Context, videoInfo *video.VideoInfo, videoData []byte, withCaption bool, maxFrames int) (*AnalyzeOutput, error) {
	var (
		wg     sync.WaitGroup
		result AnalyzeOutput
		mu     sync.Mutex
		errors []error
		errMu  sync.Mutex
	)

	// Extract audio from video
	audioData, err := video.ExtractAudio(videoData, videoInfo.Format)
	if err != nil {
		// Audio extraction might fail if video has no audio track - continue with video analysis
		audioData = nil
	}

	// Extract frames from video (uniformly sampled)
	frames, err := video.ExtractFrames(videoData, videoInfo.Format, videoInfo.Duration, maxFrames)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frames: %w", err)
	}

	// Run Sense analysis on extracted audio (if available)
	if audioData != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			audioInfo := &audio.AudioInfo{
				Format:   "wav",
				FileName: videoInfo.FileName + ".wav",
				Size:     len(audioData),
				Duration: videoInfo.Duration,
			}
			senseData, err := analyzeSense(ctx, audioInfo, audioData)
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

		// Run Caption analysis on audio if requested
		if withCaption {
			wg.Add(1)
			go func() {
				defer wg.Done()
				audioInfo := &audio.AudioInfo{
					Format:   "wav",
					FileName: videoInfo.FileName + ".wav",
					Size:     len(audioData),
					Duration: videoInfo.Duration,
				}
				caption, err := analyzeCaption(ctx, audioInfo, audioData)
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
	}

	// Run video frame analysis with Ollama
	wg.Add(1)
	go func() {
		defer wg.Done()
		videoCaption, err := analyzeVideoFrames(ctx, frames)
		if err != nil {
			errMu.Lock()
			errors = append(errors, fmt.Errorf("video analysis: %w", err))
			errMu.Unlock()
			return
		}
		mu.Lock()
		result.VideoCaption = videoCaption
		mu.Unlock()
	}()

	wg.Wait()

	if len(errors) > 0 {
		return nil, fmt.Errorf("analysis failed with %d error(s): %v", len(errors), errors)
	}

	return &result, nil
}

// analyzeVideoFrames analyzes video frames using Ollama/LLaVA in a single batch request
func analyzeVideoFrames(ctx context.Context, frames []video.FrameData) (*client.VideoAnalysisResult, error) {
	ollamaClient := common.OllamaClientFromContext(ctx)
	if ollamaClient == nil {
		return nil, fmt.Errorf("ollama client not found in context")
	}

	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames to analyze")
	}

	// Collect all frame images for batch processing
	images := make([][]byte, len(frames))
	for i, frame := range frames {
		images[i] = frame.Data
	}

	// Build timestamp info for the prompt
	var timestampInfo strings.Builder
	timestampInfo.WriteString("Frame timestamps: ")
	for i, frame := range frames {
		if i > 0 {
			timestampInfo.WriteString(", ")
		}
		timestampInfo.WriteString(fmt.Sprintf("Frame %d at %.1fs", i+1, frame.Timestamp))
	}

	// Single batch request with all frames
	prompt := fmt.Sprintf(`These are %d frames extracted from a video at different timestamps.
%s

For each frame, provide a brief description (1-2 sentences) of what you see.
Then provide an overall summary of what happens in the video.

Format your response as:
Frame 1: [description]
Frame 2: [description]
...
Summary: [overall summary of the video]`, len(frames), timestampInfo.String())

	resp, err := ollamaClient.GenerateWithImages(prompt, images)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze video frames: %w", err)
	}

	// Parse the response to extract frame descriptions and summary
	frameAnalyses, summary := parseVideoAnalysisResponse(resp.Response, frames)

	return &client.VideoAnalysisResult{
		Frames:  frameAnalyses,
		Summary: summary,
	}, nil
}

// parseVideoAnalysisResponse parses the LLM response into frame descriptions and summary
func parseVideoAnalysisResponse(response string, frames []video.FrameData) ([]client.FrameAnalysis, string) {
	frameAnalyses := make([]client.FrameAnalysis, 0, len(frames))
	lines := strings.Split(response, "\n")

	var summary string
	frameDescriptions := make(map[int]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for frame descriptions (Frame 1:, Frame 2:, etc.)
		for i := range frames {
			prefix := fmt.Sprintf("Frame %d:", i+1)
			if strings.HasPrefix(line, prefix) {
				desc := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				frameDescriptions[i] = desc
				break
			}
		}

		// Check for summary
		if strings.HasPrefix(line, "Summary:") {
			summary = strings.TrimSpace(strings.TrimPrefix(line, "Summary:"))
		}
	}

	// Build frame analyses with timestamps
	for i, frame := range frames {
		desc := frameDescriptions[i]
		if desc == "" {
			desc = "Frame description not available"
		}
		frameAnalyses = append(frameAnalyses, client.FrameAnalysis{
			Timestamp:   frame.Timestamp,
			Description: desc,
		})
	}

	// If no summary was found, use the entire response as fallback
	if summary == "" && len(frameDescriptions) == 0 {
		summary = response
	}

	return frameAnalyses, summary
}

// validateInput validates the input parameters
func validateInput(input *AnalyzeInput) error {
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

// runConcurrentAudioAnalysis runs Sense and optionally Caption analysis concurrently
func runConcurrentAudioAnalysis(ctx context.Context, audioInfo *audio.AudioInfo, rawData []byte, withCaption bool) (*AnalyzeOutput, error) {
	var (
		wg     sync.WaitGroup
		result AnalyzeOutput
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
