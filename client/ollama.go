package client

import (
	"encoding/base64"
	"fmt"

	"resty.dev/v3"

	"github.com/cochlearai/cochl-mcp-server/util/restcli"
)

// Ollama defines the interface for Ollama API client
type Ollama interface {
	// GenerateWithImages sends images to Ollama for analysis
	GenerateWithImages(prompt string, images [][]byte) (*OllamaResponse, error)
	// GetModel returns the configured model name
	GetModel() string
}

// OllamaResponse represents the response from Ollama generate API
type OllamaResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	DoneReason string `json:"done_reason,omitempty"`
}

// OllamaClient implements the Ollama interface
type OllamaClient struct {
	Client *resty.Client
	Model  string
}

// NewOllama creates a new Ollama client
func NewOllama(baseURL, model, version string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llava"
	}

	cli := resty.New().
		SetBaseURL(baseURL).
		SetHeader("User-Agent", "cochl-mcp-server/"+version)

	return &OllamaClient{
		Client: cli,
		Model:  model,
	}
}

// GetModel returns the configured model name
func (c *OllamaClient) GetModel() string {
	return c.Model
}

// GenerateWithImages sends images to Ollama for analysis using the generate API
func (c *OllamaClient) GenerateWithImages(prompt string, images [][]byte) (*OllamaResponse, error) {
	// Convert images to base64
	base64Images := make([]string, len(images))
	for i, img := range images {
		base64Images[i] = base64.StdEncoding.EncodeToString(img)
	}

	// Build request body for Ollama generate API
	body := map[string]any{
		"model":  c.Model,
		"prompt": prompt,
		"images": base64Images,
		"stream": false, // Disable streaming for simpler handling
	}

	param := restcli.Params{
		Body: body,
	}

	var result OllamaResponse
	res, err := restcli.Post(c.Client, "/api/generate", &param, &result)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("ollama returned status %d: %s", res.StatusCode(), res.String())
	}

	return &result, nil
}

// FrameAnalysis represents the analysis result for a single frame
type FrameAnalysis struct {
	Timestamp   float64 `json:"timestamp"`
	Description string  `json:"description"`
}

// VideoAnalysisResult represents the complete video analysis result
type VideoAnalysisResult struct {
	Frames  []FrameAnalysis `json:"frames"`
	Summary string          `json:"summary,omitempty"`
}
