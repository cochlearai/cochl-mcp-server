package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tcolgate/mp3"
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

func getMP3Duration(file *os.File) (float64, error) {
	var t float64
	d := mp3.NewDecoder(file)
	var f mp3.Frame
	skipped := 0
	for {

		if err := d.Decode(&f, &skipped); err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("failed to decode MP3: %v", err)
		}

		t = t + f.Duration().Seconds()
	}
	return t, nil
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
