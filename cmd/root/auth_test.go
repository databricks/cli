package root

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyHttpRequest(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	req := emptyHttpRequest(ctx)
	assert.Equal(t, req.Context(), ctx)
}
