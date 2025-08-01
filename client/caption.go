package client

import (
	"fmt"

	"resty.dev/v3"

	"github.com/cochlearai/cochl-mcp-server/util/restcli"
)

type Caption interface {
	Inference(contentType, fileName string, audioData []byte) (*RespCaptionInference, error)
}

type RespCaptionInference struct {
	Caption string `json:"caption"`
}

type CaptionClient struct {
	Client *resty.Client
}

func NewCaption(key string, baseUrl, version string) *CaptionClient {
	baseUrl = baseUrl + "/caption/v1"
	cli := resty.New().SetBaseURL(baseUrl).
		SetHeader("X-Api-Key", key).
		SetHeader("User-Agent", "cochl-mcp-server/"+version)

	return &CaptionClient{
		Client: cli,
	}
}

func (c *CaptionClient) Inference(contentType, fileName string, audioData []byte) (*RespCaptionInference, error) {
	param := restcli.Params{
		Formdata: map[string]string{
			"content_type": "audio/" + contentType,
		},
		FileData: map[string]restcli.FileData{
			"file": {
				FileName: fileName,
				Data:     audioData,
			},
		},
	}

	var result RespCaptionInference
	res, err := restcli.Post(c.Client, "/infer", &param, &result)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to infer: %v", res.String())
	}

	return &result, nil
}
