package common

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/cochlearai/cochl-mcp-server/client"
)

// Version is set at build time using ldflags
var Version = "0.0.0"

const (
	_cochlSenseProjectKeyHeader = "X-Api-Key"
	_cochlSenseBaseURLHeader    = "X-Base-Url"

	_cochlSenseProjectKeyEnvVar = "COCHL_SENSE_PROJECT_KEY"
	_cochlSenseBaseURLEnvVar    = "COCHL_SENSE_BASE_URL"

	_defaultBaseURL = "https://api.cochl.ai"
)

type cochlSenseClientKey struct{}

var ExtractCochlSenseApiClientFromEnv server.StdioContextFunc = func(ctx context.Context) context.Context {
	apiKey := os.Getenv(_cochlSenseProjectKeyEnvVar)
	baseUrl := os.Getenv(_cochlSenseBaseURLEnvVar)
	if baseUrl == "" {
		baseUrl = _defaultBaseURL
	}
	client := client.NewCochlSense(apiKey, baseUrl, Version)

	slog.Debug("CochlSense client created", "baseUrl", baseUrl, "version", Version, "api-key-set", apiKey != "")
	return context.WithValue(ctx, cochlSenseClientKey{}, client)
}

var ExtractCochlSenseApiClientFromHeader server.SSEContextFunc = func(ctx context.Context, r *http.Request) context.Context {
	apiKey := r.Header.Get(_cochlSenseProjectKeyHeader)
	baseUrl := r.Header.Get(_cochlSenseBaseURLHeader)

	if baseUrl == "" {
		baseUrl = _defaultBaseURL
	}

	client := client.NewCochlSense(apiKey, baseUrl, Version)

	slog.Debug("CochlSense client created", "baseUrl", baseUrl, "version", Version, "api-key-set", apiKey != "")
	return context.WithValue(ctx, cochlSenseClientKey{}, client)
}

var (
	SSEContextFunc   server.SSEContextFunc   = ExtractCochlSenseApiClientFromHeader
	StdioContextFunc server.StdioContextFunc = ExtractCochlSenseApiClientFromEnv
)

func CochlSenseClientFromContext(ctx context.Context) *client.CochlSenseClient {
	c, ok := ctx.Value(cochlSenseClientKey{}).(*client.CochlSenseClient)
	if !ok {
		return nil
	}
	return c
}
