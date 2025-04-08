package client

import (
	"encoding/base64"
	"fmt"
	"sync"

	"resty.dev/v3"

	"cochl-mcp-server/client/msg"
	"cochl-mcp-server/common"
	"cochl-mcp-server/util/restcli"
)

// baseURL can be overridden at build time using ldflags
var baseURL = "https://api.beta.cochl.ai/sense/api/v1"

var (
	cochlSenseClient *CochlSenseClient

	once sync.Once
)

type CochlSenseClient struct {
	Client *resty.Client
}

func newClient(key string) *resty.Client {
	return resty.New().SetBaseURL(baseURL).
		SetHeader("X-Api-Key", key).
		SetHeader("User-Agent", "cochl-mcp-server/"+common.Version)
}

func CochlSense() *CochlSenseClient {
	once.Do(func() {
		cochlSenseClient = &CochlSenseClient{
			Client: newClient(common.GetCochlSenseProjectKey()),
		}
	})
	return cochlSenseClient
}

func (c *CochlSenseClient) CreateSession(fileName, contentType string, duration float64, fileSize int) (*msg.RespCreateSession, error) {
	param := restcli.Params{
		Body: map[string]any{
			"type":         "file",
			"content_type": "audio/" + contentType,
			"total_size":   fileSize,
			"file_name":    fileName,
			"file_length":  duration,
		},
	}

	var result msg.RespCreateSession
	res, err := restcli.Post(c.Client, "/audio_sessions/", &param, &result)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create session: %v", res.String())
	}

	return &result, nil
}

func (c *CochlSenseClient) UploadChunk(sessionID string, chunkSequence int, chunk []byte) (*msg.RespUploadChunk, error) {
	base64Chunk := base64.StdEncoding.EncodeToString(chunk)
	param := restcli.Params{
		Body: map[string]any{
			"data": base64Chunk,
		},
	}

	var result msg.RespUploadChunk
	res, err := restcli.Put(c.Client, fmt.Sprintf("/audio_sessions/%s/chunks/%d", sessionID, chunkSequence), &param, &result)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to upload chunk: %v", res.String())
	}

	return &result, nil
}

func (c *CochlSenseClient) GetInferenceResult(sessionID string) (*msg.RespInferenceResult, error) {
	var result msg.RespInferenceResult
	res, err := restcli.Get(c.Client, fmt.Sprintf("/audio_sessions/%s/results", sessionID), nil, &result)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get inference result: %v", res.String())
	}

	return &result, nil
}

func (c *CochlSenseClient) DeleteSession(sessionID string) error {
	res, err := restcli.Delete(c.Client, fmt.Sprintf("/audio_sessions/%s", sessionID), nil)
	if err != nil {
		return err
	}

	if res.StatusCode() != 200 {
		return fmt.Errorf("failed to delete session: %v", res.String())
	}

	return nil
}
