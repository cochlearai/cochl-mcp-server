package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"resty.dev/v3"

	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/restcli"
)

// Supported video formats
var SupportedVideoFormats = map[string]bool{
	"mp4":  true,
	"webm": true,
	"avi":  true,
}

// VideoInfo contains metadata about a video file
type VideoInfo struct {
	Duration float64
	Width    int
	Height   int
	Format   string
	FileName string
	Size     int
}

// FrameData contains a frame image and its timestamp
type FrameData struct {
	Timestamp float64
	Data      []byte
}

// FFProbeOutput represents the JSON output from ffprobe
type FFProbeOutput struct {
	Format  FFProbeFormat   `json:"format"`
	Streams []FFProbeStream `json:"streams"`
}

type FFProbeFormat struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
}

type FFProbeStream struct {
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// IsVideoFormat checks if the given format is a supported video format
func IsVideoFormat(format string) bool {
	return SupportedVideoFormats[strings.ToLower(format)]
}

// GetVideoInfoAndData returns video info and raw data
func GetVideoInfoAndData(fileUrl string, isRemote bool) (*VideoInfo, []byte, error) {
	var (
		rawData  []byte
		format   string
		fileName string
		err      error
	)

	// Check if it's a remote HTTP URL
	if isHTTPURL(fileUrl) || isRemote {
		rawData, format, err = downloadFromHTTP(fileUrl)
		if err != nil {
			return nil, nil, err
		}
		fileName = fmt.Sprintf("video-%d.%s", time.Now().UnixNano(), format)
	} else {
		// Read local file
		rawData, err = os.ReadFile(fileUrl)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file: %v", err)
		}
		format = strings.ToLower(filepath.Ext(fileUrl))
		if format != "" {
			format = format[1:] // Remove the dot
		}
		fileName = filepath.Base(fileUrl)
	}

	if !IsVideoFormat(format) {
		return nil, nil, fmt.Errorf("unsupported video format: %s", format)
	}

	// Create temp file for ffprobe
	tmpFile, err := os.CreateTemp("", "video-*."+format)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(rawData); err != nil {
		return nil, nil, fmt.Errorf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Get video info using ffprobe
	info, err := getVideoInfoFromFile(tmpFile.Name())
	if err != nil {
		return nil, nil, err
	}

	info.Format = format
	info.FileName = fileName
	info.Size = len(rawData)

	return info, rawData, nil
}

// getVideoInfoFromFile uses ffprobe to get video metadata
func getVideoInfoFromFile(filePath string) (*VideoInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe failed: %v, stderr: %s", err, stderr.String())
	}

	var probeOutput FFProbeOutput
	if err := json.Unmarshal(stdout.Bytes(), &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	info := &VideoInfo{}

	// Parse duration
	if probeOutput.Format.Duration != "" {
		duration, err := strconv.ParseFloat(probeOutput.Format.Duration, 64)
		if err == nil {
			info.Duration = duration
		}
	}

	// Get video stream dimensions
	for _, stream := range probeOutput.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			break
		}
	}

	return info, nil
}

// ExtractAudio extracts audio track from video file
// Returns audio data in WAV format
func ExtractAudio(videoData []byte, format string) ([]byte, error) {
	// Create temp input file
	tmpInput, err := os.CreateTemp("", "video-input-*."+format)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %v", err)
	}
	defer os.Remove(tmpInput.Name())

	if _, err := tmpInput.Write(videoData); err != nil {
		tmpInput.Close()
		return nil, fmt.Errorf("failed to write temp input file: %v", err)
	}
	tmpInput.Close()

	// Create temp output file
	tmpOutput, err := os.CreateTemp("", "audio-output-*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %v", err)
	}
	tmpOutputName := tmpOutput.Name()
	tmpOutput.Close()
	defer os.Remove(tmpOutputName)

	// Extract audio using ffmpeg
	cmd := exec.Command("ffmpeg",
		"-i", tmpInput.Name(),
		"-vn",                // No video
		"-acodec", "pcm_s16le", // PCM 16-bit little-endian
		"-ar", "44100",       // Sample rate
		"-ac", "2",           // Stereo
		"-y",                 // Overwrite output
		tmpOutputName,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg audio extraction failed: %v, stderr: %s", err, stderr.String())
	}

	// Read the output audio file
	audioData, err := os.ReadFile(tmpOutputName)
	if err != nil {
		return nil, fmt.Errorf("failed to read extracted audio: %v", err)
	}

	return audioData, nil
}

// ExtractFrames extracts a fixed number of frames uniformly distributed across the video
func ExtractFrames(videoData []byte, format string, duration float64, maxFrames int) ([]FrameData, error) {
	if maxFrames <= 0 {
		maxFrames = 8 // Default 8 frames
	}
	if maxFrames > 16 {
		maxFrames = 16 // Cap at 16 frames to avoid token limits
	}

	// Create temp input file
	tmpInput, err := os.CreateTemp("", "video-input-*."+format)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %v", err)
	}
	defer os.Remove(tmpInput.Name())

	if _, err := tmpInput.Write(videoData); err != nil {
		tmpInput.Close()
		return nil, fmt.Errorf("failed to write temp input file: %v", err)
	}
	tmpInput.Close()

	// Calculate timestamps for uniform sampling
	timestamps := calculateUniformTimestamps(duration, maxFrames)

	// Create temp directory for frames
	tmpDir, err := os.MkdirTemp("", "frames-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract frames at specific timestamps
	var frames []FrameData
	for i, ts := range timestamps {
		outputPath := filepath.Join(tmpDir, fmt.Sprintf("frame-%04d.jpg", i+1))

		cmd := exec.Command("ffmpeg",
			"-ss", fmt.Sprintf("%.3f", ts), // Seek to timestamp
			"-i", tmpInput.Name(),
			"-vframes", "1", // Extract 1 frame
			"-q:v", "2",     // High quality JPEG
			"-y",
			outputPath,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// Skip frames that fail to extract (e.g., timestamp beyond video)
			continue
		}

		frameData, err := os.ReadFile(outputPath)
		if err != nil {
			continue
		}

		frames = append(frames, FrameData{
			Timestamp: ts,
			Data:      frameData,
		})
	}

	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames extracted from video")
	}

	return frames, nil
}

// calculateUniformTimestamps returns evenly distributed timestamps across the video duration
func calculateUniformTimestamps(duration float64, count int) []float64 {
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return []float64{0}
	}

	timestamps := make([]float64, count)
	interval := duration / float64(count-1)

	for i := 0; i < count; i++ {
		timestamps[i] = float64(i) * interval
		// Ensure we don't exceed duration
		if timestamps[i] > duration {
			timestamps[i] = duration
		}
	}

	return timestamps
}

// isHTTPURL checks if the given URL is HTTP or HTTPS
func isHTTPURL(fileUrl string) bool {
	return strings.HasPrefix(strings.ToLower(fileUrl), "http://") ||
		strings.HasPrefix(strings.ToLower(fileUrl), "https://")
}

// downloadFromHTTP downloads file from HTTP URL
func downloadFromHTTP(fileUrl string) ([]byte, string, error) {
	// Check if it's a Google Drive URL and convert it
	downloadURL := fileUrl
	if util.IsGoogleDriveURL(fileUrl) {
		convertedURL, err := util.ConvertGoogleDriveURL(fileUrl)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert Google Drive URL: %v", err)
		}
		downloadURL = convertedURL
	} else if util.IsDropboxURL(fileUrl) {
		convertedURL, err := util.ConvertDropboxURL(fileUrl)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert Dropbox URL: %v", err)
		}
		downloadURL = convertedURL
	}

	// Create resty client with timeout
	client := resty.New().
		SetTimeout(5 * time.Minute). // Longer timeout for video files
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second)

	resp, err := restcli.Get(client, downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %v", err)
	}

	if !resp.IsSuccess() {
		return nil, "", fmt.Errorf("HTTP error: %s", resp.Status())
	}

	data := resp.Bytes()

	// Get format from Content-Type or URL
	contentType := resp.Header().Get("Content-Type")
	var format string
	switch contentType {
	case "video/mp4":
		format = "mp4"
	case "video/webm":
		format = "webm"
	case "video/x-msvideo", "video/avi":
		format = "avi"
	default:
		// Try to get from URL extension
		format = strings.ToLower(filepath.Ext(fileUrl))
		if format != "" {
			format = format[1:]
		}
	}

	return data, format, nil
}
