package sqlcli

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFormat_Defaults(t *testing.T) {
	ctx := t.Context()
	got, err := ResolveFormat(ctx, "text", false, true)
	require.NoError(t, err)
	assert.Equal(t, OutputText, got)
}

func TestResolveFormat_TextOnPipeFallsBackToJSON(t *testing.T) {
	ctx := t.Context()
	got, err := ResolveFormat(ctx, "text", false, false)
	require.NoError(t, err)
	assert.Equal(t, OutputJSON, got)
}

func TestResolveFormat_ExplicitTextOnPipeIsHonoured(t *testing.T) {
	ctx := t.Context()
	got, err := ResolveFormat(ctx, "text", true, false)
	require.NoError(t, err)
	assert.Equal(t, OutputText, got)
}

func TestResolveFormat_EnvVarTextOnPipeIsHonoured(t *testing.T) {
	ctx := env.Set(t.Context(), EnvOutputFormat, "text")
	got, err := ResolveFormat(ctx, "text", false, false)
	require.NoError(t, err)
	assert.Equal(t, OutputText, got)
}

func TestResolveFormat_EnvVarCSVOnPipe(t *testing.T) {
	ctx := env.Set(t.Context(), EnvOutputFormat, "csv")
	got, err := ResolveFormat(ctx, "text", false, false)
	require.NoError(t, err)
	assert.Equal(t, OutputCSV, got)
}

func TestResolveFormat_ExplicitJSON(t *testing.T) {
	ctx := t.Context()
	got, err := ResolveFormat(ctx, "json", true, true)
	require.NoError(t, err)
	assert.Equal(t, OutputJSON, got)
}

func TestResolveFormat_ExplicitCSV(t *testing.T) {
	ctx := t.Context()
	got, err := ResolveFormat(ctx, "csv", true, true)
	require.NoError(t, err)
	assert.Equal(t, OutputCSV, got)
}

func TestResolveFormat_EnvVarHonoredWhenFlagNotSet(t *testing.T) {
	ctx := env.Set(t.Context(), EnvOutputFormat, "csv")
	got, err := ResolveFormat(ctx, "text", false, true)
	require.NoError(t, err)
	assert.Equal(t, OutputCSV, got)
}

func TestResolveFormat_FlagOverridesEnvVar(t *testing.T) {
	ctx := env.Set(t.Context(), EnvOutputFormat, "csv")
	got, err := ResolveFormat(ctx, "json", true, true)
	require.NoError(t, err)
	assert.Equal(t, OutputJSON, got)
}

func TestResolveFormat_InvalidEnvVarIgnored(t *testing.T) {
	ctx := env.Set(t.Context(), EnvOutputFormat, "yaml")
	got, err := ResolveFormat(ctx, "text", false, true)
	require.NoError(t, err)
	assert.Equal(t, OutputText, got)
}

func TestResolveFormat_InvalidFlagErrors(t *testing.T) {
	ctx := t.Context()
	_, err := ResolveFormat(ctx, "yaml", true, true)
	assert.ErrorContains(t, err, "unsupported output format")
}

func TestResolveFormat_CaseInsensitive(t *testing.T) {
	ctx := t.Context()
	got, err := ResolveFormat(ctx, "JSON", true, true)
	require.NoError(t, err)
	assert.Equal(t, OutputJSON, got)
}

func TestIsKnown(t *testing.T) {
	assert.True(t, IsKnown(OutputText))
	assert.True(t, IsKnown(OutputJSON))
	assert.True(t, IsKnown(OutputCSV))
	assert.False(t, IsKnown(Format("yaml")))
	assert.False(t, IsKnown(Format("")))
}
