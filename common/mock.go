package common

import (
	"context"

	"github.com/cochlearai/cochl-mcp-server/client"
)

func NewTestContext(clientType string) context.Context {
	switch clientType {
	case "caption":
		return context.WithValue(context.Background(), captionApiClientKey{}, client.NewMockCaption())
	case "sense":
		return context.WithValue(context.Background(), senseApiClientKey{}, client.NewMockSense())
	}
	return context.Background()
}
