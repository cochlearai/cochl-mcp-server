package audio

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"resty.dev/v3"

	"github.com/cochlearai/cochl-mcp-server/util"
	"github.com/cochlearai/cochl-mcp-server/util/restcli"
)

type AudioInfo struct {
	Duration float64
	Size     int
	Format   string
	FileName string
}

// FFProbe output structure
type FFProbeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

// isHTTPURL checks if the given URL is HTTP or HTTPS
func isHTTPURL(fileUrl string) bool {
	return strings.HasPrefix(strings.ToLower(fileUrl), "http://") ||
		strings.HasPrefix(strings.ToLower(fileUrl), "https://")
}

// downloadFromHTTP downloads file from HTTP URL using resty
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
		SetTimeout(1 * time.Minute).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second)

	// Use restcli to download the file
	resp, err := restcli.Get(client, downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %v", err)
	}

	if !resp.IsSuccess() {
		return nil, "", fmt.Errorf("HTTP error: %s", resp.Status())
	}

	// Get response body as bytes
	data := resp.Bytes()

	// Try to get format from Content-Type header
	contentType := resp.Header().Get("Content-Type")
	var format string
	switch contentType {
	case "audio/wav", "audio/wave":
		format = "wav"
	case "audio/mpeg", "audio/mp3":
		format = "mp3"
	case "audio/ogg", "application/ogg":
		format = "ogg"
	default:
		parsedURL, err := url.Parse(fileUrl)
		if err == nil {
			cleanPath := parsedURL.Path
			format = strings.ToLower(filepath.Ext(cleanPath))
			if format != "" {
				format = format[1:]
			}
		} else {
			format = strings.ToLower(filepath.Ext(fileUrl))
			if format != "" {
				format = format[1:]
			}
		}
	}
	return data, format, nil
}

// GetAudioInfoAndData returns both audio info and raw data in a single file read
func GetAudioInfoAndData(fileUrl string, isRemote bool) (*AudioInfo, []byte, error) {
	var (
		rawData    []byte
		format     string
		err        error
		probeInput string // Path for ffprobe (either original path or temp file)
		tempFile   *os.File
	)

	var fileName string

	// Check if it's a remote HTTP URL or use the isRemote flag
	if isHTTPURL(fileUrl) || isRemote {
		rawData, format, err = downloadFromHTTP(fileUrl)
		if err != nil {
			return nil, nil, err
		}

		// no need to get filename from URL
		fileName = fmt.Sprintf("audio-%d.%s", time.Now().UnixNano(), format)

		// Create temporary file for ffprobe
		tempFile, err = os.CreateTemp("", fmt.Sprintf("audio-*.%s", format))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Write downloaded data to temp file
		if _, err = tempFile.Write(rawData); err != nil {
			return nil, nil, fmt.Errorf("failed to write temp file: %v", err)
		}

		// Use temp file for ffprobe
		probeInput = tempFile.Name()
	} else {
		// Read local file
		rawData, err = os.ReadFile(fileUrl)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file: %v", err)
		}

		// Get format from file extension for local files
		format = strings.ToLower(filepath.Ext(fileUrl))
		if format != "" {
			format = format[1:] // Remove the dot
		}

		fileName = filepath.Base(fileUrl)
		probeInput = fileUrl
	}

	// Get file size
	size := len(rawData)

	duration, err := getAudioDurationWithFFProbe(probeInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get audio duration: %v", err)
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
	// Run ffprobe command
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe command failed: %v", err)
	}

	// Parse JSON output
	var probeData FFProbeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	// Parse duration string to float64
	var duration float64
	if _, err := fmt.Sscanf(probeData.Format.Duration, "%f", &duration); err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}

	return duration, nil
}

// SplitAudioIntoChunks splits an audio file into chunks using ffmpeg
// Returns a slice of output file paths
func SplitAudioIntoChunks(inputPath string, outputDir string, chunkDuration int) ([]string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Get the file extension
	ext := filepath.Ext(inputPath)

	// Create output pattern
	outputPattern := filepath.Join(outputDir, fmt.Sprintf("chunk_%%03d%s", ext))

	// Run ffmpeg command to split audio
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", chunkDuration),
		"-c", "copy",
		"-reset_timestamps", "1",
		outputPattern)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg command failed: %v", err)
	}

	// Find all generated chunk files using glob pattern
	pattern := filepath.Join(outputDir, fmt.Sprintf("chunk_*%s", ext))
	outputFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find output files: %v", err)
	}

	if len(outputFiles) == 0 {
		return nil, fmt.Errorf("no output files were created")
	}

	return outputFiles, nil
}
