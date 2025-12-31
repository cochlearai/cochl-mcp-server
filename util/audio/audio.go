package audio

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cochlearai/cochl-mcp-server/util"
)

// AudioInfo contains basic information about an audio file
type AudioInfo struct {
	Duration float64 // Duration in seconds
	Size     int     // Size in bytes
	Format   string  // Audio format (mp3, wav, ogg, etc.)
	FileName string  // Original file name
}

// ChunkInfo represents information about a single audio chunk
type ChunkInfo struct {
	Index    int    // Chunk index (0-based)
	FilePath string // Path to chunk file
}

// FFProbe output structure
type FFProbeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

// audioContentTypeMap maps Content-Type headers to audio formats.
var audioContentTypeMap = map[string]string{
	"audio/wav":       "wav",
	"audio/wave":      "wav",
	"audio/mpeg":      "mp3",
	"audio/mp3":       "mp3",
	"audio/ogg":       "ogg",
	"application/ogg": "ogg",
}

// downloadFromHTTP downloads an audio file from HTTP URL.
func downloadFromHTTP(fileURL string) ([]byte, string, error) {
	result, err := util.DownloadFromHTTP(fileURL, util.DownloadOptions{
		Timeout:        time.Minute,
		ContentTypeMap: audioContentTypeMap,
	})
	if err != nil {
		return nil, "", err
	}
	return result.Data, result.Format, nil
}

// GetAudioInfoAndData returns both audio info and raw data in a single file read.
func GetAudioInfoAndData(fileURL string, isRemote bool) (*AudioInfo, []byte, error) {
	var (
		rawData    []byte
		format     string
		err        error
		probeInput string // Path for ffprobe (either original path or temp file)
		tempFile   *os.File
	)

	var fileName string

	// Check if it's a remote HTTP URL or use the isRemote flag
	if util.IsHTTPURL(fileURL) || isRemote {
		rawData, format, err = downloadFromHTTP(fileURL)
		if err != nil {
			return nil, nil, err
		}

		fileName = fmt.Sprintf("audio-%d.%s", time.Now().UnixNano(), format)

		// Create temporary file for ffprobe
		tempFile, err = os.CreateTemp("", fmt.Sprintf("audio-*.%s", format))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		if _, err = tempFile.Write(rawData); err != nil {
			return nil, nil, fmt.Errorf("failed to write temp file: %w", err)
		}

		probeInput = tempFile.Name()
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
		probeInput = fileURL
	}

	// Get file size
	size := len(rawData)

	duration, err := getAudioDurationWithFFProbe(probeInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get audio duration: %w", err)
	}

	info := &AudioInfo{
		Duration: duration,
		Size:     size,
		Format:   format,
		FileName: fileName,
	}

	return info, rawData, nil
}

func getAudioDurationWithFFProbe(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe command failed: %w", err)
	}

	var probeData FFProbeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(probeData.Format.Duration, "%f", &duration); err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

// SaveRawDataToTempFile saves raw audio data to a temporary file
// If dir is empty, uses system temp directory. Otherwise saves to the specified directory.
// Returns the temp file path. Cleanup is caller's responsibility.
func SaveRawDataToTempFile(rawData []byte, format, dir string) (string, error) {
	tempFile, err := os.CreateTemp(dir, fmt.Sprintf("audio-*.%s", format))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tempFile.Write(rawData); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// SplitAudioIntoChunks splits an audio file into chunks using ffmpeg.
// chunkDuration is in seconds. Returns a sorted slice of output file paths.
func SplitAudioIntoChunks(inputPath, outputDir string, chunkDuration int) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	ext := filepath.Ext(inputPath)
	outputPattern := filepath.Join(outputDir, fmt.Sprintf("chunk_%%03d%s", ext))

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", chunkDuration),
		"-c", "copy",
		"-reset_timestamps", "1",
		outputPattern)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg command failed: %w", err)
	}

	pattern := filepath.Join(outputDir, fmt.Sprintf("chunk_*%s", ext))
	outputFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find output files: %w", err)
	}

	if len(outputFiles) == 0 {
		return nil, fmt.Errorf("no output files were created")
	}

	return outputFiles, nil
}
