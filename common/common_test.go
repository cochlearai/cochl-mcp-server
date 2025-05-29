package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cochlearai/cochl-mcp-server/client"
)

func Test_CaptionClientFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), captionApiClientKey{}, &client.MockCaption{})
	caption := CaptionClientFromContext(ctx)
	assert.NotNil(t, caption)

	nilCtx := context.Background()
	nilCaption := CaptionClientFromContext(nilCtx)
	assert.Nil(t, nilCaption)
}

func Test_SenseClientFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), senseApiClientKey{}, &client.MockSense{})
	sense := SenseClientFromContext(ctx)
	assert.NotNil(t, sense)

	nilCtx := context.Background()
	nilSense := SenseClientFromContext(nilCtx)
	assert.Nil(t, nilSense)
}
