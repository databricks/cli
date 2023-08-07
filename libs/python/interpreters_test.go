package python

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtLeastOnePythonInstalled(t *testing.T) {
	ctx := context.Background()
	all, err := DetectInterpreters(ctx)
	assert.NoError(t, err)
	a := all.Latest()
	t.Logf("latest is: %s", a)
	assert.True(t, len(all) > 0)
}
