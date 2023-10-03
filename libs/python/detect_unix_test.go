//go:build unix

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

func TestDetectsViaListing(t *testing.T) {
	t.Setenv("PATH", "testdata/other-binaries-filtered")
	ctx := context.Background()
	py, err := DetectExecutable(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "testdata/other-binaries-filtered/python3.10", py)
}

func TestDetectFailsNoInterpreters(t *testing.T) {
	t.Setenv("PATH", "testdata")
	ctx := context.Background()
	_, err := DetectExecutable(ctx)
	assert.Equal(t, ErrNoPythonInterpreters, err)
}

func TestDetectFailsNoMinimalVersion(t *testing.T) {
	t.Setenv("PATH", "testdata/no-python3")
	ctx := context.Background()
	_, err := DetectExecutable(ctx)
	assert.EqualError(t, err, "cannot find Python greater or equal to v3.8.0")
}
