//go:build unix

package python

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectsViaPathLookup(t *testing.T) {
	ctx := t.Context()
	py, err := DetectExecutable(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, py)
}

func TestDetectFailsNoInterpreters(t *testing.T) {
	t.Setenv("PATH", "testdata")
	ctx := t.Context()
	_, err := DetectExecutable(ctx)
	assert.Error(t, err)
}

func TestDetectFailsNoMinimalVersion(t *testing.T) {
	t.Setenv("PATH", "testdata/no-python3")
	ctx := t.Context()
	_, err := DetectExecutable(ctx)
	assert.Error(t, err)
}
