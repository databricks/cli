package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadStateNonexistentFile(t *testing.T) {
	state, err := LoadState(t.TempDir())
	assert.NoError(t, err)
	assert.Nil(t, state)
}

func TestSaveAndLoadStateRoundtrip(t *testing.T) {
	dir := t.TempDir()
	original := &InstallState{
		SchemaVersion: 1,
		SkillsRef:     "v0.2.0",
		LastChecked:   "2026-03-22T10:00:00Z",
		Skills: map[string]InstalledSkill{
			"databricks": {
				Version:     "1.0.0",
				InstalledAt: "2026-03-22T09:00:00Z",
			},
		},
	}

	err := SaveState(dir, original)
	require.NoError(t, err)

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}

func TestSaveStateCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "path")
	state := &InstallState{
		SchemaVersion: 1,
		SkillsRef:     "v0.1.0",
		Skills:        map[string]InstalledSkill{},
	}

	err := SaveState(dir, state)
	require.NoError(t, err)

	// Verify file exists.
	_, err = os.Stat(filepath.Join(dir, stateFileName))
	assert.NoError(t, err)
}

func TestLoadStateCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, stateFileName), []byte("{bad json"), 0o644))

	state, err := LoadState(dir)
	assert.Error(t, err)
	assert.Nil(t, state)
	assert.Contains(t, err.Error(), "failed to parse state file")
}

func TestGlobalSkillsDir(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), "/fake/home")
	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/fake/home", ".databricks", "aitools", "skills"), dir)
}

func TestProjectSkillsDirNotImplemented(t *testing.T) {
	dir, err := ProjectSkillsDir(t.Context())
	assert.ErrorIs(t, err, ErrNotImplemented)
	assert.Empty(t, dir)
}
