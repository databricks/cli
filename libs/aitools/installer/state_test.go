package installer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
		SchemaVersion: schemaVersionV2,
		Release:       "v0.2.0",
		LastUpdated:   time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		Skills: map[string]string{
			"databricks": "1.0.0",
		},
		RepoDirs: map[string]string{
			"databricks": stableSkillsRepoPath,
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
		Release:       "v0.1.0",
		LastUpdated:   time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC),
		Skills:        map[string]string{},
	}

	err := SaveState(dir, state)
	require.NoError(t, err)

	// Verify file exists.
	_, err = os.Stat(filepath.Join(dir, stateFileName))
	assert.NoError(t, err)
}

func TestSaveStateTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	state := &InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		LastUpdated:   time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC),
		Skills:        map[string]string{},
	}

	err := SaveState(dir, state)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, stateFileName))
	require.NoError(t, err)
	assert.Equal(t, byte('\n'), data[len(data)-1])
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

func TestProjectSkillsDirReturnsCwdBased(t *testing.T) {
	dir, err := ProjectSkillsDir(t.Context())
	require.NoError(t, err)
	cwd, _ := os.Getwd()
	assert.Equal(t, filepath.Join(cwd, ".databricks", "aitools", "skills"), dir)
}

func TestLoadStateMigratesV1ToV2(t *testing.T) {
	dir := t.TempDir()
	// A v1 state on disk has no plugins/files keys.
	v1 := `{"schema_version":1,"release":"v0.2.0","last_updated":"2026-03-22T10:00:00Z","skills":{"databricks":"1.0.0"},"repo_dirs":{"databricks":"skills"}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, stateFileName), []byte(v1), 0o644))

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, schemaVersionV2, loaded.SchemaVersion)
	// Migration is additive: existing data is untouched and the new maps stay nil.
	assert.Equal(t, map[string]string{"databricks": "1.0.0"}, loaded.Skills)
	assert.Nil(t, loaded.Plugins)
	assert.Nil(t, loaded.Files)
}

func TestMigrateStateIsIdempotent(t *testing.T) {
	// Start at v1 so this exercises the real v1 -> v2 migration, then confirm a
	// second migrateState is a no-op.
	state := &InstallState{SchemaVersion: 1, Skills: map[string]string{"databricks": "1.0.0"}}
	migrateState(state)
	assert.Equal(t, schemaVersionV2, state.SchemaVersion)

	migrated := *state
	migrateState(state)
	assert.Equal(t, migrated, *state)
}

func TestSaveAndLoadStateWithPluginAndFileRecords(t *testing.T) {
	dir := t.TempDir()
	original := &InstallState{
		SchemaVersion: schemaVersionV2,
		Release:       "v0.2.6",
		LastUpdated:   time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
		Skills:        map[string]string{"databricks": "1.0.0"},
		RepoDirs:      map[string]string{"databricks": stableSkillsRepoPath},
		Plugins: map[string]PluginRecord{
			"claude-code": {
				Marketplace:          "databricks-agent-skills",
				Plugin:               "databricks",
				Scope:                "user",
				Version:              "0.2.6",
				InstalledMarketplace: true,
			},
		},
		Files: map[string]FileRecord{
			"databricks/SKILL.md": {SHA256: "abc123", Origin: "v0.2.6"},
		},
	}

	require.NoError(t, SaveState(dir, original))

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}

func TestSaveAndLoadStateWithOptionalFields(t *testing.T) {
	dir := t.TempDir()
	original := &InstallState{
		SchemaVersion:       schemaVersionV2,
		IncludeExperimental: true,
		Release:             "v0.3.0",
		LastUpdated:         time.Date(2026, 3, 22, 12, 30, 0, 0, time.UTC),
		Skills: map[string]string{
			"databricks": "1.0.0",
			"sql-tools":  "0.2.0",
		},
		RepoDirs: map[string]string{
			"databricks": stableSkillsRepoPath,
			"sql-tools":  experimentalRepoPath,
		},
		Scope: "project",
	}

	err := SaveState(dir, original)
	require.NoError(t, err)

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}
