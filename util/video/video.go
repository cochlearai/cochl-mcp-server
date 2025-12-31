package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cochlearai/cochl-mcp-server/util"
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

// GetVideoInfoAndData returns video info and raw data.
func GetVideoInfoAndData(fileURL string, isRemote bool) (*VideoInfo, []byte, error) {
	var (
		rawData  []byte
		format   string
		fileName string
		err      error
	)

	if util.IsHTTPURL(fileURL) || isRemote {
		rawData, format, err = downloadFromHTTP(fileURL)
		if err != nil {
			return nil, nil, err
		}
		fileName = fmt.Sprintf("video-%d.%s", time.Now().UnixNano(), format)
	} else {
		rawData, err = os.ReadFile(fileURL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file: %w", err)
		}
		format = strings.ToLower(filepath.Ext(fileURL))
		if format != "" {
			format = format[1:] // Remove the dot
		}
		fileName = filepath.Base(fileURL)
	}

	if !IsVideoFormat(format) {
		return nil, nil, fmt.Errorf("unsupported video format: %s", format)
	}

	tmpFile, err := os.CreateTemp("", "video-*."+format)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(rawData); err != nil {
		return nil, nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	info, err := getVideoInfoFromFile(tmpFile.Name())
	if err != nil {
		return nil, nil, err
	}

	info.Format = format
	info.FileName = fileName
	info.Size = len(rawData)

	return info, rawData, nil
}

// getVideoInfoFromFile uses ffprobe to get video metadata.
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
		return nil, fmt.Errorf("ffprobe failed: %w, stderr: %s", err, stderr.String())
	}

	var probeOutput FFProbeOutput
	if err := json.Unmarshal(stdout.Bytes(), &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
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

// ExtractAudio extracts audio track from video file.
// Returns audio data in WAV format.
func ExtractAudio(videoData []byte, format string) ([]byte, error) {
	tmpInput, err := os.CreateTemp("", "video-input-*."+format)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(tmpInput.Name())

	if _, err := tmpInput.Write(videoData); err != nil {
		tmpInput.Close()
		return nil, fmt.Errorf("failed to write temp input file: %w", err)
	}
	tmpInput.Close()

	tmpOutput, err := os.CreateTemp("", "audio-output-*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	tmpOutputName := tmpOutput.Name()
	tmpOutput.Close()
	defer os.Remove(tmpOutputName)

	cmd := exec.Command("ffmpeg",
		"-i", tmpInput.Name(),
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "44100",
		"-ac", "2",
		"-y",
		tmpOutputName,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg audio extraction failed: %w, stderr: %s", err, stderr.String())
	}

	audioData, err := os.ReadFile(tmpOutputName)
	if err != nil {
		return nil, fmt.Errorf("failed to read extracted audio: %w", err)
	}

	return audioData, nil
}

// ExtractFrames extracts frames using uniform sampling.
// It aims for 1 frame every 1.5 seconds, with a minimum of 8 frames.
// Videos longer than 60 seconds are rejected.
func ExtractFrames(videoData []byte, format string, duration float64, maxFrames int) ([]FrameData, error) {
	const (
		minFrames   = 8
		interval    = 1.5 // Seconds per frame
		maxDuration = 60.0
	)

	if duration > maxDuration {
		return nil, fmt.Errorf("video duration %.2fs exceeds the limit of %.0fs", duration, maxDuration)
	}

	// Calculate target frame count based on duration and interval
	targetCount := max(int(math.Ceil(duration / interval)), minFrames)

	// Note: We ignore the maxFrames parameter to enforce the 1.5s interval policy
	// and removed the maxAllowed cap as requested.

	tmpInput, err := os.CreateTemp("", "video-input-*."+format)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(tmpInput.Name())

	if _, err := tmpInput.Write(videoData); err != nil {
		tmpInput.Close()
		return nil, fmt.Errorf("failed to write temp input file: %w", err)
	}
	tmpInput.Close()

	tmpDir, err := os.MkdirTemp("", "frames-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use Uniform Sampling
	detectedFrames, err := extractUniformFrames(tmpInput.Name(), duration, targetCount, tmpDir)
	if err != nil {
		return nil, err
	}

	// Read actual image data for the selected frames
	var resultFrames []FrameData
	for _, f := range detectedFrames {
		data, err := os.ReadFile(f.filePath)
		if err != nil {
			continue // Skip if read failed
		}
		resultFrames = append(resultFrames, FrameData{
			Timestamp: f.Timestamp,
			Data:      data,
		})
	}

	if len(resultFrames) == 0 {
		return nil, fmt.Errorf("no frames extracted")
	}

	// Sort by timestamp just in case
	sort.Slice(resultFrames, func(i, j int) bool {
		return resultFrames[i].Timestamp < resultFrames[j].Timestamp
	})

	return resultFrames, nil
}

// temp internal struct for processing
type detectedFrame struct {
	FrameData
	filePath string
}

func parseFFmpegTimestamps(logs string, dir string, prefix string) []detectedFrame {
	var frames []detectedFrame

	// 1. Parse timestamps from logs
	re := regexp.MustCompile(`n:\s*(\d+).*?pts_time:\s*([0-9\.]+)`)
	matches := re.FindAllStringSubmatch(logs, -1)

	// 2. Get actual files from directory
	pattern := filepath.Join(dir, fmt.Sprintf("%s-*.jpg", prefix))
	files, _ := filepath.Glob(pattern)
	sort.Strings(files) // Ensure order: frame-001, frame-002 ...

	// 3. Map logs to files
	// If counts match, map 1:1.
	// If mismatch, prioritize files (since we need image data).
	count := len(files)
	if count == 0 {
		return nil
	}

	for i, file := range files {
		var ts float64
		// If we have a matching log entry, use its timestamp
		if i < len(matches) {
			// matches[i][2] is pts_time
			parsedTs, err := strconv.ParseFloat(matches[i][2], 64)
			if err == nil {
				ts = parsedTs
			}
		}

		frames = append(frames, detectedFrame{
			FrameData: FrameData{Timestamp: ts},
			filePath:  file,
		})
	}

	return frames
}

func extractUniformFrames(inputPath string, duration float64, count int, tmpDir string) ([]detectedFrame, error) {
	if count <= 0 {
		return nil, fmt.Errorf("invalid frame count: %d", count)
	}

	fps := float64(count) / duration
	outputPattern := filepath.Join(tmpDir, "uniform-%03d.jpg")

	// Use fps filter to extract uniformly
	// Note: fps filter might not produce exactly 'count' frames due to rounding/duration issues,
	// but it's much faster than loop seeking.
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", fmt.Sprintf("fps=%.4f,showinfo", fps),
		"-q:v", "2",
		"-y",
		outputPattern,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg uniform extraction failed: %w", err)
	}

	// 1. Try parsing timestamps to know exact times
	frames := parseFFmpegTimestamps(stderr.String(), tmpDir, "uniform")

	// 2. SAFETY NET: If parsing failed (empty) but files exist, use them
	// This handles cases where showinfo format differs or parsing fails
	if len(frames) == 0 {
		// Glob for generated files
		files, _ := filepath.Glob(filepath.Join(tmpDir, "uniform-*.jpg"))
		if len(files) > 0 {
			sort.Strings(files) // uniform-001, uniform-002...

			// Reconstruct approx timestamps
			actualCount := len(files)
			interval := duration / float64(actualCount) // approx interval

			for i, file := range files {
				ts := float64(i) * interval
				frames = append(frames, detectedFrame{
					FrameData: FrameData{Timestamp: ts},
					filePath:  file,
				})
			}
		}
	}

	// If fps filter produced too many/few, we might need to slice or pad?
	// Usually for summary, approximate count is fine.
	// But let's limit to count if it exceeded slightly
	if len(frames) > count {
		step := float64(len(frames)-1) / float64(count-1)
		var downsampled []detectedFrame
		for i := 0; i < count; i++ {
			idx := int(math.Round(float64(i) * step))
			if idx >= len(frames) {
				idx = len(frames) - 1
			}
			downsampled = append(downsampled, frames[idx])
		}
		frames = downsampled
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

// videoContentTypeMap maps Content-Type headers to video formats.
var videoContentTypeMap = map[string]string{
	"video/mp4":       "mp4",
	"video/webm":      "webm",
	"video/x-msvideo": "avi",
	"video/avi":       "avi",
}

// downloadFromHTTP downloads a video file from HTTP URL.
func downloadFromHTTP(fileURL string) ([]byte, string, error) {
	result, err := util.DownloadFromHTTP(fileURL, util.DownloadOptions{
		Timeout:        5 * time.Minute, // Longer timeout for video files
		ContentTypeMap: videoContentTypeMap,
	})
	if err != nil {
		return nil, "", err
	}
	return result.Data, result.Format, nil
}
