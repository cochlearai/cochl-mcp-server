package client

import (
	"encoding/base64"
	"fmt"

	"resty.dev/v3"

	"github.com/cochlearai/cochl-mcp-server/util/restcli"
)

type Sense interface {
	CreateSession(fileName, contentType string, duration float64, fileSize int) (*RespCreateSession, error)
	UploadChunk(sessionID string, chunkSequence int, chunk []byte) (*RespUploadChunk, error)
	GetInferenceResult(sessionID string) (*RespInferenceResult, error)
	DeleteSession(sessionID string) error
}

type RespUploadChunk struct {
	ChunkSequence int    `json:"chunk_sequence"`
	SessionID     string `json:"session_id"`
}

type RespCreateSession struct {
	SessionID     string `json:"session_id"`
	ChunkSequence int    `json:"chunk_sequence"`
	WindowSize    int    `json:"window_size"`
	WindowHop     int    `json:"window_hop"`
}

type InferenceResult struct {
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
	Tags      []Tags `json:"tags"`
}

type Tags struct {
	Probability float64 `json:"probability"`
	Name        string  `json:"name"`
}

type RespInferenceResult struct {
	Data  []InferenceResult `json:"data"`
	State string            `json:"state"`
}

type SenseClient struct {
	Client *resty.Client
}

func NewSense(key string, baseUrl, version string) *SenseClient {
	baseUrl = baseUrl + "/sense/api/v1"
	cli := resty.New().SetBaseURL(baseUrl).
		SetHeader("X-Api-Key", key).
		SetHeader("User-Agent", "cochl-mcp-server/"+version)

	return &SenseClient{
		Client: cli,
	}
}

func (c *SenseClient) CreateSession(fileName, contentType string, duration float64, fileSize int) (*RespCreateSession, error) {
	param := restcli.Params{
		Body: map[string]any{
			"type":         "file",
			"content_type": "audio/" + contentType,
			"total_size":   fileSize,
			"file_name":    fileName,
			"file_length":  duration,
		},
	}

	var result RespCreateSession
	res, err := restcli.Post(c.Client, "/audio_sessions/", &param, &result)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create session: %v", res.String())
	}

	return &result, nil
}

func (c *SenseClient) UploadChunk(sessionID string, chunkSequence int, chunk []byte) (*RespUploadChunk, error) {
	base64Chunk := base64.StdEncoding.EncodeToString(chunk)
	param := restcli.Params{
		Body: map[string]any{
			"data": base64Chunk,
		},
	}

	var result RespUploadChunk
	res, err := restcli.Put(c.Client, fmt.Sprintf("/audio_sessions/%s/chunks/%d", sessionID, chunkSequence), &param, &result)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to upload chunk: %v", res.String())
	}

	return &result, nil
}

func (c *SenseClient) GetInferenceResult(sessionID string) (*RespInferenceResult, error) {
	var result RespInferenceResult
	res, err := restcli.Get(c.Client, fmt.Sprintf("/audio_sessions/%s/results", sessionID), nil, &result)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get inference result: %v", res.String())
	}

	return &result, nil
}

func (c *SenseClient) DeleteSession(sessionID string) error {
	res, err := restcli.Delete(c.Client, fmt.Sprintf("/audio_sessions/%s", sessionID), nil)
	if err != nil {
		return err
	}

	if res.StatusCode() != 200 {
		return fmt.Errorf("failed to delete session: %v", res.String())
	}

	return nil
}
