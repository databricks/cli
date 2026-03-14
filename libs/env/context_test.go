package env_test

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("FOO", "bar")

	ctx0 := t.Context()

	// Get
	assert.Equal(t, "bar", env.Get(ctx0, "FOO"))
	assert.Equal(t, "", env.Get(ctx0, "dontexist"))

	// Lookup
	v, ok := env.Lookup(ctx0, "FOO")
	assert.True(t, ok)
	assert.Equal(t, "bar", v)
	v, ok = env.Lookup(ctx0, "dontexist")
	assert.False(t, ok)
	assert.Equal(t, "", v)

	// Set and get new context.
	// Verify that the previous context remains unchanged.
	ctx1 := env.Set(ctx0, "FOO", "baz")
	assert.Equal(t, "baz", env.Get(ctx1, "FOO"))
	assert.Equal(t, "bar", env.Get(ctx0, "FOO"))

	// Set and get new context.
	// Verify that the previous contexts remains unchanged.
	ctx2 := env.Set(ctx1, "FOO", "qux")
	assert.Equal(t, "qux", env.Get(ctx2, "FOO"))
	assert.Equal(t, "baz", env.Get(ctx1, "FOO"))
	assert.Equal(t, "bar", env.Get(ctx0, "FOO"))

	ctx3 := env.Set(ctx2, "BAR", "x=y")

	all := env.All(ctx3)
	assert.NotNil(t, all)
	assert.Equal(t, "qux", all["FOO"])
	assert.Equal(t, "x=y", all["BAR"])
	assert.NotEmpty(t, all["PATH"])
}

func TestHome(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "...")
	home, err := env.UserHomeDir(ctx)
	assert.Equal(t, "...", home)
	assert.NoError(t, err)
}

func TestGetBool(t *testing.T) {
	testutil.CleanupEnvironment(t)
	ctx := t.Context()

	// Test true values
	trueValues := []string{"true", "TRUE", "True", "1", "t", "T", "yes", "YES", "Yes", "on", "ON", "On"}
	for _, v := range trueValues {
		t.Run("true_"+v, func(t *testing.T) {
			ctx := env.Set(ctx, "TEST_BOOL", v)
			val, ok := env.GetBool(ctx, "TEST_BOOL")
			assert.True(t, ok, "expected key to be set")
			assert.True(t, val, "expected %q to be true", v)
		})
	}

	// Test false values
	falseValues := []string{"false", "FALSE", "False", "0", "f", "F", "no", "NO", "No", "off", "OFF", "Off", ""}
	for _, v := range falseValues {
		t.Run("false_"+v, func(t *testing.T) {
			ctx := env.Set(ctx, "TEST_BOOL", v)
			val, ok := env.GetBool(ctx, "TEST_BOOL")
			assert.True(t, ok, "expected key to be set")
			assert.False(t, val, "expected %q to be false", v)
		})
	}

	// Test invalid/unknown values default to false but ok=true
	invalidValues := []string{"invalid", "random", "2", "maybe"}
	for _, v := range invalidValues {
		t.Run("invalid_"+v, func(t *testing.T) {
			ctx := env.Set(ctx, "TEST_BOOL", v)
			val, ok := env.GetBool(ctx, "TEST_BOOL")
			assert.True(t, ok, "expected key to be set")
			assert.False(t, val, "expected %q to be false (invalid)", v)
		})
	}

	// Test missing key returns ok=false
	val, ok := env.GetBool(ctx, "NON_EXISTENT_KEY")
	assert.False(t, ok, "expected key to not be set")
	assert.False(t, val, "expected value to be false when not set")

	// Test from actual environment variable
	t.Setenv("TEST_ENV_BOOL", "true")
	val, ok = env.GetBool(t.Context(), "TEST_ENV_BOOL")
	assert.True(t, ok)
	assert.True(t, val)

	t.Setenv("TEST_ENV_BOOL_FALSE", "0")
	val, ok = env.GetBool(t.Context(), "TEST_ENV_BOOL_FALSE")
	assert.True(t, ok)
	assert.False(t, val)
}
