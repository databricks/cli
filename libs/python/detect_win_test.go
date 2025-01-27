//go:build windows

package python

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectsViaPathLookup(t *testing.T) {
	ctx := context.Background()
	py, err := DetectExecutable(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, py)
}

func TestDetectFailsNoInterpreters(t *testing.T) {
	t.Setenv("PATH", "testdata")
	ctx := context.Background()
	_, err := DetectExecutable(ctx)
	assert.Error(t, err)
}
