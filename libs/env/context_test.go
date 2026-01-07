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

func TestGetBool(t *testing.T) {
	testutil.CleanupEnvironment(t)
	ctx := context.Background()

	// Test true values
	trueValues := []string{"true", "TRUE", "True", "1", "t", "T", "yes", "YES", "Yes", "on", "ON", "On"}
	for _, v := range trueValues {
		t.Run("true_"+v, func(t *testing.T) {
			ctx := Set(ctx, "TEST_BOOL", v)
			val, ok := GetBool(ctx, "TEST_BOOL")
			assert.True(t, ok, "expected key to be set")
			assert.True(t, val, "expected %q to be true", v)
		})
	}

	// Test false values
	falseValues := []string{"false", "FALSE", "False", "0", "f", "F", "no", "NO", "No", "off", "OFF", "Off", ""}
	for _, v := range falseValues {
		t.Run("false_"+v, func(t *testing.T) {
			ctx := Set(ctx, "TEST_BOOL", v)
			val, ok := GetBool(ctx, "TEST_BOOL")
			assert.True(t, ok, "expected key to be set")
			assert.False(t, val, "expected %q to be false", v)
		})
	}

	// Test invalid/unknown values default to false but ok=true
	invalidValues := []string{"invalid", "random", "2", "maybe"}
	for _, v := range invalidValues {
		t.Run("invalid_"+v, func(t *testing.T) {
			ctx := Set(ctx, "TEST_BOOL", v)
			val, ok := GetBool(ctx, "TEST_BOOL")
			assert.True(t, ok, "expected key to be set")
			assert.False(t, val, "expected %q to be false (invalid)", v)
		})
	}

	// Test missing key returns ok=false
	val, ok := GetBool(ctx, "NON_EXISTENT_KEY")
	assert.False(t, ok, "expected key to not be set")
	assert.False(t, val, "expected value to be false when not set")

	// Test from actual environment variable
	t.Setenv("TEST_ENV_BOOL", "true")
	val, ok = GetBool(context.Background(), "TEST_ENV_BOOL")
	assert.True(t, ok)
	assert.True(t, val)

	t.Setenv("TEST_ENV_BOOL_FALSE", "0")
	val, ok = GetBool(context.Background(), "TEST_ENV_BOOL_FALSE")
	assert.True(t, ok)
	assert.False(t, val)
}
