package env

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("FOO", "bar")

	ctx0 := context.Background()

	// Get
	assert.Equal(t, "bar", Get(ctx0, "FOO"))
	assert.Equal(t, "", Get(ctx0, "dontexist"))

	// Lookup
	v, ok := Lookup(ctx0, "FOO")
	assert.True(t, ok)
	assert.Equal(t, "bar", v)
	v, ok = Lookup(ctx0, "dontexist")
	assert.False(t, ok)
	assert.Equal(t, "", v)

	// Set and get new context.
	// Verify that the previous context remains unchanged.
	ctx1 := Set(ctx0, "FOO", "baz")
	assert.Equal(t, "baz", Get(ctx1, "FOO"))
	assert.Equal(t, "bar", Get(ctx0, "FOO"))

	// Set and get new context.
	// Verify that the previous contexts remains unchanged.
	ctx2 := Set(ctx1, "FOO", "qux")
	assert.Equal(t, "qux", Get(ctx2, "FOO"))
	assert.Equal(t, "baz", Get(ctx1, "FOO"))
	assert.Equal(t, "bar", Get(ctx0, "FOO"))

	ctx3 := Set(ctx2, "BAR", "x=y")

	all := All(ctx3)
	assert.NotNil(t, all)
	assert.Equal(t, "qux", all["FOO"])
	assert.Equal(t, "x=y", all["BAR"])
	assert.NotEmpty(t, all["PATH"])
}

func TestHome(t *testing.T) {
	ctx := context.Background()
	ctx = WithUserHomeDir(ctx, "...")
	home, err := UserHomeDir(ctx)
	assert.Equal(t, "...", home)
	assert.NoError(t, err)
}
