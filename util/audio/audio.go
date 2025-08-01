package audio

import (
	"encoding/binary"
	"fmt"
	"net/url"
	"os"
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
		rawData []byte
		format  string
		err     error
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
	}

	// Get file size
	size := len(rawData)

	var duration float64

	// Process based on file format using raw data
	switch format {
	case "wav":
		duration, err = getWAVDurationFromData(rawData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get WAV duration: %v", err)
		}
	case "mp3":
		duration, err = getMP3DurationFromData(rawData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get MP3 duration: %v", err)
		}
	case "ogg":
		duration, err = getOggDurationFromData(rawData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get OGG duration: %v", err)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported audio format: %s", format)
	}

	info := &AudioInfo{
		Duration: duration,
		Size:     size,
		Format:   format,
		FileName: fileName,
	}

	return info, rawData, nil
}

func getWAVDurationFromData(data []byte) (float64, error) {
	if len(data) < 44 {
		return 0, fmt.Errorf("file too small to be a valid WAV")
	}

	// Check WAV signature
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return 0, fmt.Errorf("invalid WAV file")
	}

	// Get audio format data
	chunkSize := binary.LittleEndian.Uint32(data[4:8])
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])
	numChannels := binary.LittleEndian.Uint16(data[22:24])

	// Calculate duration
	bytesPerSample := bitsPerSample / 8
	duration := float64(chunkSize) / float64(sampleRate*uint32(numChannels)*uint32(bytesPerSample))

	return duration, nil
}

var mp3BitRates = map[int]int{
	1:  32,
	2:  40,
	3:  48,
	4:  56,
	5:  64,
	6:  80,
	7:  96,
	8:  112,
	9:  128,
	10: 160,
	11: 192,
	12: 224,
	13: 256,
	14: 320,
}

var mp3SampleRates = map[int]int{
	0: 44100,
	1: 48000,
	2: 32000,
}

func getMP3DurationFromData(data []byte) (float64, error) {
	size := int64(len(data))
	if size <= 128 {
		return 0, fmt.Errorf("file too small to be a valid MP3")
	}

	var offset int64 = 0
	if len(data) >= 10 && string(data[0:3]) == "ID3" {
		tagSize := int64(data[6]&0x7f)<<21 |
			int64(data[7]&0x7f)<<14 |
			int64(data[8]&0x7f)<<7 |
			int64(data[9]&0x7f)
		offset = tagSize + 10
	}

	var bitRate, sampleRate, frameSize int
	var totalFrames int64
	var isVBR bool
	var prevBitRate int

	for offset < size-4 {
		frameHeader := data[offset : offset+4]

		if frameHeader[0] == 0xff && (frameHeader[1]&0xe0) == 0xe0 {
			version := (frameHeader[1] >> 3) & 0x03
			layer := (frameHeader[1] >> 1) & 0x03
			bitrateIndex := int((frameHeader[2] >> 4) & 0x0f)
			samplerateIndex := int((frameHeader[2] >> 2) & 0x03)
			padding := (frameHeader[2] >> 1) & 0x01

			if version == 3 && layer == 1 {
				bitRate = mp3BitRates[bitrateIndex] * 1000
				sampleRate = mp3SampleRates[samplerateIndex]

				if prevBitRate != 0 && prevBitRate != bitRate {
					isVBR = true
				}
				prevBitRate = bitRate

				frameSize = ((144 * bitRate) / sampleRate) + int(padding)
				totalFrames++
				offset += int64(frameSize)
			} else {
				offset++
			}
		} else {
			offset++
		}
	}

	if totalFrames == 0 {
		return 0, fmt.Errorf("no valid MP3 frames found")
	}

	if isVBR {
		duration := float64(size) / (float64(bitRate) / 8.0)
		return duration, nil
	}

	samplesPerFrame := 1152.0 // MPEG1 Layer3
	duration := float64(totalFrames) * samplesPerFrame / float64(sampleRate)
	return duration, nil
}

func getOggDurationFromData(data []byte) (float64, error) {
	// Search for the "OggS" signature and calculate the length
	var length int64
	for i := len(data) - 14; i >= 0 && length == 0; i-- {
		if data[i] == 'O' && data[i+1] == 'g' && data[i+2] == 'g' && data[i+3] == 'S' {
			length = int64(binary.LittleEndian.Uint64(data[i+6 : i+14]))
		}
	}

	// Search for the "vorbis" signature and calculate the rate
	var rate int64
	for i := 0; i < len(data)-14 && rate == 0; i++ {
		if data[i] == 'v' && data[i+1] == 'o' && data[i+2] == 'r' && data[i+3] == 'b' && data[i+4] == 'i' && data[i+5] == 's' {
			rate = int64(binary.LittleEndian.Uint32(data[i+11 : i+15]))
		}
	}

	if length == 0 || rate == 0 {
		return 0, fmt.Errorf("could not find necessary information in Ogg file")
	}

	duration := float64(length) / float64(rate)
	return duration, nil
}
