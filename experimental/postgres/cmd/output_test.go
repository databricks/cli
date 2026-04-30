package postgrescmd

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOutputFormat_Defaults(t *testing.T) {
	ctx := t.Context()

	got, err := resolveOutputFormat(ctx, "text", false, true)
	require.NoError(t, err)
	assert.Equal(t, outputText, got)
}

func TestResolveOutputFormat_TextOnPipeFallsBackToJSON(t *testing.T) {
	ctx := t.Context()
	got, err := resolveOutputFormat(ctx, "text", false, false)
	require.NoError(t, err)
	assert.Equal(t, outputJSON, got)
}

func TestResolveOutputFormat_ExplicitTextOnPipeIsHonoured(t *testing.T) {
	ctx := t.Context()
	got, err := resolveOutputFormat(ctx, "text", true, false)
	require.NoError(t, err)
	assert.Equal(t, outputText, got)
}

func TestResolveOutputFormat_EnvVarTextOnPipeIsHonoured(t *testing.T) {
	ctx := env.Set(t.Context(), envOutputFormat, "text")
	got, err := resolveOutputFormat(ctx, "text", false, false)
	require.NoError(t, err)
	assert.Equal(t, outputText, got)
}

func TestResolveOutputFormat_EnvVarCSVOnPipe(t *testing.T) {
	ctx := env.Set(t.Context(), envOutputFormat, "csv")
	got, err := resolveOutputFormat(ctx, "text", false, false)
	require.NoError(t, err)
	assert.Equal(t, outputCSV, got)
}

func TestResolveOutputFormat_ExplicitJSON(t *testing.T) {
	ctx := t.Context()
	got, err := resolveOutputFormat(ctx, "json", true, true)
	require.NoError(t, err)
	assert.Equal(t, outputJSON, got)
}

func TestResolveOutputFormat_ExplicitCSV(t *testing.T) {
	ctx := t.Context()
	got, err := resolveOutputFormat(ctx, "csv", true, true)
	require.NoError(t, err)
	assert.Equal(t, outputCSV, got)
}

func TestResolveOutputFormat_EnvVarHonoredWhenFlagNotSet(t *testing.T) {
	ctx := env.Set(t.Context(), envOutputFormat, "csv")
	got, err := resolveOutputFormat(ctx, "text", false, true)
	require.NoError(t, err)
	assert.Equal(t, outputCSV, got)
}

func TestResolveOutputFormat_FlagOverridesEnvVar(t *testing.T) {
	ctx := env.Set(t.Context(), envOutputFormat, "csv")
	got, err := resolveOutputFormat(ctx, "json", true, true)
	require.NoError(t, err)
	assert.Equal(t, outputJSON, got)
}

func TestResolveOutputFormat_InvalidEnvVarIgnored(t *testing.T) {
	ctx := env.Set(t.Context(), envOutputFormat, "yaml")
	got, err := resolveOutputFormat(ctx, "text", false, true)
	require.NoError(t, err)
	assert.Equal(t, outputText, got)
}

func TestResolveOutputFormat_InvalidFlagErrors(t *testing.T) {
	ctx := t.Context()
	_, err := resolveOutputFormat(ctx, "yaml", true, true)
	assert.ErrorContains(t, err, "unsupported output format")
}

func TestResolveOutputFormat_CaseInsensitive(t *testing.T) {
	ctx := t.Context()
	got, err := resolveOutputFormat(ctx, "JSON", true, true)
	require.NoError(t, err)
	assert.Equal(t, outputJSON, got)
}
