package common

import (
	"context"
	"log/slog"
	"os"

	"github.com/cochlearai/cochl-mcp-server/client"
)

// Version is set at build time using ldflags
var Version = "HEAD"

const (
	_cochlSenseProjectKeyEnvVar = "COCHL_SENSE_PROJECT_KEY"
	_cochlSenseBaseURLEnvVar    = "COCHL_SENSE_BASE_URL"

	_ollamaBaseURLEnvVar = "OLLAMA_BASE_URL"
	_ollamaModelEnvVar   = "OLLAMA_MODEL"

	_defaultBaseURL       = "https://api.cochl.ai"
	_defaultOllamaBaseURL = "http://localhost:11434"
	_defaultOllamaModel   = "llava"
)

type senseApiClientKey struct{}
type captionApiClientKey struct{}
type ollamaClientKey struct{}

var ExtractCochlApiClientFromEnv = func(ctx context.Context) context.Context {
	apiKey := os.Getenv(_cochlSenseProjectKeyEnvVar)
	baseURL := os.Getenv(_cochlSenseBaseURLEnvVar)
	if baseURL == "" {
		baseURL = _defaultBaseURL
	}

	senseClient := client.NewSense(apiKey, baseURL, Version)
	captionClient := client.NewCaption(apiKey, baseURL, Version)

	ctx = context.WithValue(ctx, senseApiClientKey{}, senseClient)
	ctx = context.WithValue(ctx, captionApiClientKey{}, captionClient)

	slog.Debug("Cochl api client created", "baseURL", baseURL, "version", Version, "api-key-set", apiKey != "")

	ollamaBaseURL := os.Getenv(_ollamaBaseURLEnvVar)
	if ollamaBaseURL == "" {
		ollamaBaseURL = _defaultOllamaBaseURL
	}
	ollamaModel := os.Getenv(_ollamaModelEnvVar)
	if ollamaModel == "" {
		ollamaModel = _defaultOllamaModel
	}

	ollamaClient := client.NewOllama(ollamaBaseURL, ollamaModel, Version)
	ctx = context.WithValue(ctx, ollamaClientKey{}, ollamaClient)

	slog.Debug("Ollama client created", "baseURL", ollamaBaseURL, "model", ollamaModel)

	return ctx
}

func CaptionClientFromContext(ctx context.Context) client.Caption {
	c, ok := ctx.Value(captionApiClientKey{}).(client.Caption)
	if !ok {
		return nil
	}
	return c
}

func SenseClientFromContext(ctx context.Context) client.Sense {
	c, ok := ctx.Value(senseApiClientKey{}).(client.Sense)
	if !ok {
		return nil
	}
	return c
}

func OllamaClientFromContext(ctx context.Context) client.Ollama {
	c, ok := ctx.Value(ollamaClientKey{}).(client.Ollama)
	if !ok {
		return nil
	}
	return c
}
