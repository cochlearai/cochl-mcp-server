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

type senseApiClientKey struct{}
type captionApiClientKey struct{}

var ExtractCochlApiClientFromEnv server.StdioContextFunc = func(ctx context.Context) context.Context {
	apiKey := os.Getenv(_cochlSenseProjectKeyEnvVar)
	baseUrl := os.Getenv(_cochlSenseBaseURLEnvVar)
	if baseUrl == "" {
		baseUrl = _defaultBaseURL
	}

	senseClient := client.NewSense(apiKey, baseUrl, Version)
	captionClient := client.NewCaption(apiKey, baseUrl, Version)

	ctx = context.WithValue(ctx, senseApiClientKey{}, senseClient)
	ctx = context.WithValue(ctx, captionApiClientKey{}, captionClient)

	slog.Debug("Cochl api client created", "baseUrl", baseUrl, "version", Version, "api-key-set", apiKey != "")

	return ctx
}

var ExtractCochlApiClientFromHeader server.HTTPContextFunc = func(ctx context.Context, r *http.Request) context.Context {
	apiKey := r.Header.Get(_cochlSenseProjectKeyHeader)
	baseUrl := r.Header.Get(_cochlSenseBaseURLHeader)

	if baseUrl == "" {
		baseUrl = _defaultBaseURL
	}

	senseClient := client.NewSense(apiKey, baseUrl, Version)
	captionClient := client.NewCaption(apiKey, baseUrl, Version)

	ctx = context.WithValue(ctx, senseApiClientKey{}, senseClient)
	ctx = context.WithValue(ctx, captionApiClientKey{}, captionClient)

	slog.Debug("Cochl api client created", "baseUrl", baseUrl, "version", Version, "api-key-set", apiKey != "")

	return ctx
}

var (
	HTTPContextFunc  server.HTTPContextFunc  = ExtractCochlApiClientFromHeader
	StdioContextFunc server.StdioContextFunc = ExtractCochlApiClientFromEnv
)

func CaptionClientFromContext(ctx context.Context) *client.CaptionClient {
	c, ok := ctx.Value(captionApiClientKey{}).(*client.CaptionClient)
	if !ok {
		return nil
	}
	return c
}

func SenseClientFromContext(ctx context.Context) *client.SenseClient {
	c, ok := ctx.Value(senseApiClientKey{}).(*client.SenseClient)
	if !ok {
		return nil
	}
	return c
}
