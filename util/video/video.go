package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
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
	FPS      float64
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
	CodecType    string `json:"codec_type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
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

	// Get video stream dimensions and FPS
	for _, stream := range probeOutput.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height

			// Parse FPS
			if stream.AvgFrameRate != "" {
				parts := strings.Split(stream.AvgFrameRate, "/")
				if len(parts) == 2 {
					num, err1 := strconv.ParseFloat(parts[0], 64)
					den, err2 := strconv.ParseFloat(parts[1], 64)
					if err1 == nil && err2 == nil && den != 0 {
						info.FPS = num / den
					}
				} else {
					// Fallback for simple number
					fps, err := strconv.ParseFloat(stream.AvgFrameRate, 64)
					if err == nil {
						info.FPS = fps
					}
				}
			}
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

// ExtractFrames extracts frames using MiniCPM-V 2.6/4.5 compatible sampling logic.
// It calculates the number of frames based on duration and preferred FPS,
// respecting the model's packing limits.
// If maxFramesLimit is > 0, it overrides the default maximum frame limit (180).
func ExtractFrames(videoData []byte, format string, duration float64, videoFPS float64, maxFramesLimit int) ([]FrameData, error) {
	const (
		defaultMaxNumFrames = 180 // MiniCPM-V constant
		maxNumPacking       = 3   // MiniCPM-V constant
		chooseFPS           = 5.0 // Preferred sampling FPS
	)

	maxNumFrames := defaultMaxNumFrames
	if maxFramesLimit > 0 {
		maxNumFrames = maxFramesLimit
	}

	// Logic from MiniCPM-V: encode_video
	var targetCount int
	// packingNums is calculated but not strictly needed for frame extraction count,
	// unless we want to report it. For now we use it to determine targetCount.
	// var packingNums int

	// Calculate target count
	if chooseFPS*duration <= float64(maxNumFrames) {
		// packingNums = 1
		fpsToUse := min(chooseFPS, videoFPS)
		if fpsToUse <= 0 {
			fpsToUse = chooseFPS // Fallback if videoFPS is invalid
		}
		targetCount = int(math.Round(fpsToUse * min(float64(maxNumFrames), duration)))
	} else {
		packingNums := int(math.Ceil(duration * chooseFPS / float64(maxNumFrames)))
		if packingNums <= maxNumPacking {
			targetCount = int(math.Round(duration * chooseFPS))
		} else {
			targetCount = maxNumFrames * maxNumPacking
			// packingNums = maxNumPacking
		}
	}

	// Safety check: ensure at least 1 frame
	if targetCount < 1 {
		targetCount = 1
	}

	// Debug log or similar could go here to show packingNums if needed,
	// but we just need the frames.

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

// extractUniformFrames extracts frames uniformly from the video
func extractUniformFrames(inputFile string, duration float64, count int, outputDir string) ([]detectedFrame, error) {
	if count <= 0 {
		count = 1
	}

	// Calculate interval
	interval := duration / float64(count)

	// FFmpeg command to extract frames at intervals
	// -vf "fps=1/interval"
	// Note: using fps filter is generally more reliable than seek for uniform sampling
	fps := 1.0 / interval
	if fps > 100 {
		fps = 30 // Cap at reasonable fps if duration is very short
	}

	// Output pattern
	outputPattern := filepath.Join(outputDir, "frame-%03d.jpg")

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputFile,
		"-vf", fmt.Sprintf("fps=%f", fps),
		"-q:v", "2", // High quality JPEG
		"-frames:v", fmt.Sprintf("%d", count),
		outputPattern,
	)

	// Capture output for debugging (optional)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg uniform sampling failed: %s", string(output))
	}

	// Collect generated files
	matches, err := filepath.Glob(filepath.Join(outputDir, "frame-*.jpg"))
	if err != nil {
		return nil, err
	}

	sort.Strings(matches)

	// Limit to requested count
	if len(matches) > count {
		matches = matches[:count]
	}

	var frames []detectedFrame
	for i, match := range matches {
		// Calculate timestamp based on index and interval
		// Center the timestamp in the interval: (i + 0.5) * interval
		ts := (float64(i) + 0.5) * interval
		if ts > duration {
			ts = duration
		}

		frames = append(frames, detectedFrame{
			FrameData: FrameData{Timestamp: ts},
			filePath:  match,
		})
	}

	return frames, nil
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
