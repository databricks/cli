package autoinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFileIsZeroState(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{}, s)
}

func TestLoadStateWithoutLastAttempt(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	// The format of CLI versions before last_attempt was added.
	require.NoError(t, os.WriteFile(path, []byte(`{"last_version": "1.27.0", "attempts": 3, "id": "machine"}`), 0o600))

	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{LastVersion: "1.27.0", Attempts: 3, ID: "machine"}, s)
}

func TestLoadCorruptFileFails(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, err := Load(ctx)
	assert.ErrorContains(t, err, "parse ")
}

func TestLoadNegativeAttemptsFails(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"last_version": "1.30.0", "attempts": -1}`), 0o600))

	_, err := Load(ctx)
	assert.ErrorContains(t, err, "negative attempts -1")
}

func TestSetOptedOutKeepsOtherFields(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	require.NoError(t, save(ctx, State{LastVersion: "1.2.3", Attempts: 2}))

	require.NoError(t, SetOptedOut(ctx, true))
	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{LastVersion: "1.2.3", Attempts: 2, OptedOut: true}, s)

	require.NoError(t, SetOptedOut(ctx, false))
	s, err = Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{LastVersion: "1.2.3", Attempts: 2}, s)
}

func TestSetOptedOutReplacesCorruptFile(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	require.NoError(t, SetOptedOut(ctx, true))
	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{OptedOut: true}, s)
}

func TestEnvOptedOut(t *testing.T) {
	ctx := t.Context()
	assert.False(t, EnvOptedOut(ctx), "unset")

	for value, want := range map[string]bool{
		"true":    false,
		"1":       false,
		"yes":     false,
		"on":      false,
		"false":   true,
		"0":       true,
		"no":      true,
		"off":     true,
		"":        true,
		"enabled": true,
	} {
		t.Run(value, func(t *testing.T) {
			assert.Equal(t, want, EnvOptedOut(env.Set(ctx, EnvVar, value)))
		})
	}
}
