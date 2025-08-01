package audio

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AudioInfo struct {
	Duration float64
	Size     int
	Format   string
	FileName string
}

// GetAudioInfoAndData returns both audio info and raw data in a single file read
func GetAudioInfoAndData(filePath string) (*AudioInfo, []byte, error) {
	// Read the entire file once
	rawData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Get file size
	size := len(rawData)

	// Get format from file extension
	format := strings.ToLower(filepath.Ext(filePath))
	if format != "" {
		format = format[1:] // Remove the dot
	}

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
		FileName: filepath.Base(filePath),
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
