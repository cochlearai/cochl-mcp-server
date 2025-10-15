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

	_defaultBaseURL = "https://api.cochl.ai"
)

type senseApiClientKey struct{}
type captionApiClientKey struct{}

var ExtractCochlApiClientFromEnv = func(ctx context.Context) context.Context {
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
