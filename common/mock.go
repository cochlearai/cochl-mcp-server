package common

import (
	"context"

	"github.com/cochlearai/cochl-mcp-server/client"
)

func NewTestContext(testType string) context.Context {
	switch testType {
	case "analyze_audio", "analyze_media":
		ctx := context.WithValue(context.Background(), senseApiClientKey{}, client.NewMockSense())
		ctx = context.WithValue(ctx, captionApiClientKey{}, client.NewMockCaption())
		ctx = context.WithValue(ctx, ollamaClientKey{}, client.NewMockOllama())
		return ctx
	}
	return context.Background()
}
