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

func GetRawAudioData(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func GetAudioInfo(filePath string) (*AudioInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}
	size := int(fileInfo.Size())

	// Get format from file extension
	format := strings.ToLower(filepath.Ext(filePath))
	if format != "" {
		format = format[1:] // Remove the dot
	}

	var duration float64

	// Process based on file format
	switch format {
	case "wav":
		duration, err = getWAVDuration(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get WAV duration: %v", err)
		}
	case "mp3":
		duration, err = getMP3Duration(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get MP3 duration: %v", err)
		}
	case "ogg":
		duration, err = getOggDuration(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get OGG duration: %v", err)
		}
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", format)
	}

	return &AudioInfo{
		Duration: duration,
		Size:     size,
		Format:   format,
		FileName: filepath.Base(filePath),
	}, nil
}

func getWAVDuration(file *os.File) (float64, error) {
	// Read WAV header
	header := make([]byte, 44)
	if _, err := file.Read(header); err != nil {
		return 0, fmt.Errorf("failed to read WAV header: %v", err)
	}

	// Check WAV signature
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return 0, fmt.Errorf("invalid WAV file")
	}

	// Get audio format data
	chunkSize := binary.LittleEndian.Uint32(header[4:8])
	sampleRate := binary.LittleEndian.Uint32(header[24:28])
	bitsPerSample := binary.LittleEndian.Uint16(header[34:36])
	numChannels := binary.LittleEndian.Uint16(header[22:24])

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

func getMP3Duration(file *os.File) (float64, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %v", err)
	}

	size := fileInfo.Size()
	if size <= 128 {
		return 0, fmt.Errorf("file too small to be a valid MP3")
	}

	header := make([]byte, 10)
	_, err = file.ReadAt(header, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to read header: %v", err)
	}

	var offset int64 = 0
	if string(header[0:3]) == "ID3" {
		tagSize := int64(header[6]&0x7f)<<21 |
			int64(header[7]&0x7f)<<14 |
			int64(header[8]&0x7f)<<7 |
			int64(header[9]&0x7f)
		offset = tagSize + 10
	}

	frameHeader := make([]byte, 4)
	var bitRate, sampleRate, frameSize int
	var totalFrames int64
	var isVBR bool
	var prevBitRate int

	for offset < size-4 {
		_, err = file.ReadAt(frameHeader, offset)
		if err != nil {
			return 0, fmt.Errorf("failed to read frame header: %v", err)
		}

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

func getOggDuration(file *os.File) (float64, error) {
	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("error getting file info: %w", err)
	}

	// Read the entire file
	data := make([]byte, fileInfo.Size())
	_, err = file.Read(data)
	if err != nil {
		return 0, fmt.Errorf("error reading Ogg file: %w", err)
	}

	// Reset file position for future reads
	_, err = file.Seek(0, 0)
	if err != nil {
		return 0, fmt.Errorf("error resetting file position: %w", err)
	}

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
