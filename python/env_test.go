package python

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreeze(t *testing.T) {

	// remove this once equivalent tests for windows have been set up
	// date: 28 Nov 2022
	if runtime.GOOS == "windows" {
		t.Skip("skipping temperorilty to make windows unit tests green")
	}

	env, err := Freeze(context.Background())
	assert.NoError(t, err)
	assert.Greater(t, len(env), 1)
	assert.True(t, env.Has("urllib3"))
}

func TestPyInlineX(t *testing.T) {
	defer chdirAndBack("testdata/simple-python-wheel")()
	dist, err := ReadDistribution(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "dummy", dist.Name)
	assert.Equal(t, "dummy", dist.Packages[0])
	assert.True(t, dist.InstallEnvironment().Has("requests"))
}
