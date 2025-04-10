package common

import (
	"context"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"cochl-mcp-server/client"
)

// Version is set at build time using ldflags
var Version = "0.0.0"

const (
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

func CochlSenseClientFromContext(ctx context.Context) *client.CochlSenseClient {
	c, ok := ctx.Value(cochlSenseClientKey{}).(*client.CochlSenseClient)
	if !ok {
		return nil
	}
	return c
}
