package installer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
		SchemaVersion: CurrentSchemaVersion,
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

func TestLoadStateCorruptReturnsStateError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, stateFileName), []byte("{not json"), 0o644))

	_, err := LoadState(dir)
	serr, ok := errors.AsType[*StateError](err)
	require.True(t, ok, "want *StateError, got %T", err)
	assert.Equal(t, filepath.Join(dir, stateFileName), serr.Path)
}

func TestLoadStateUnreadableIsNotStateError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions and a non-root user")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, stateFileName)
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o644))
	require.NoError(t, os.Chmod(path, 0o000))
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	// An unreadable file is a permission problem, not invalid content: moving
	// it aside would not help, so it must not look like a StateError.
	_, err := LoadState(dir)
	require.ErrorIs(t, err, fs.ErrPermission)
	_, isState := errors.AsType[*StateError](err)
	assert.False(t, isState)
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
	assert.Equal(t, CurrentSchemaVersion, loaded.SchemaVersion)
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
	assert.Equal(t, CurrentSchemaVersion, state.SchemaVersion)

	migrated := *state
	migrateState(state)
	assert.Equal(t, migrated, *state)
}

func TestSaveAndLoadStateWithPluginAndFileRecords(t *testing.T) {
	dir := t.TempDir()
	original := &InstallState{
		SchemaVersion: CurrentSchemaVersion,
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
		SchemaVersion:       CurrentSchemaVersion,
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

func TestSaveAndLoadStateKeepsMCPSection(t *testing.T) {
	dir := t.TempDir()
	original := &MCPState{Servers: []MCPEntry{{
		Name:           "github-logfood",
		Service:        "system.ai.github",
		Profile:        "logfood",
		Host:           "https://adb-123.test",
		WorkspaceID:    "123",
		SignInRequired: true,
		Command:        "/usr/local/bin/databricks",
		Agents: map[string]MCPAgentRecord{
			"claude-code": {Path: "/home/me/.claude.json", Scope: "user"},
		},
		InstalledAt: time.Date(2026, 11, 12, 10, 12, 0, 0, time.UTC),
		CLIVersion:  "0.300.0",
	}}}
	require.NoError(t, SaveMCPState(dir, original))

	loaded, err := LoadMCPState(dir)
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}

// TestSaveStateKeepsMCPRecordsSavedAfterLoad checks that an installer save writes the MCP section
// on disk, not the copy loaded before an MCP save was made.
func TestSaveStateKeepsMCPRecordsSavedAfterLoad(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, SaveState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion, Skills: map[string]string{"databricks": "1.0.0"}}))

	stale, err := LoadState(dir)
	require.NoError(t, err)
	require.NoError(t, SaveMCPState(dir, &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}))

	stale.Skills["databricks"] = "1.1.0"
	require.NoError(t, SaveState(dir, stale))

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"databricks": "1.1.0"}, loaded.Skills)
	require.NotNil(t, loaded.MCP)
	assert.Equal(t, []MCPEntry{{Name: "github-logfood"}}, loaded.MCP.Servers)
}

// TestSaveMCPStateKeepsSkillRecords checks that an MCP save changes only the MCP section.
func TestSaveMCPStateKeepsSkillRecords(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, SaveState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion, Skills: map[string]string{"databricks": "1.0.0"}}))

	require.NoError(t, SaveMCPState(dir, &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}))
	require.NoError(t, SaveMCPState(dir, &MCPState{}))

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"databricks": "1.0.0"}, loaded.Skills)
	assert.Nil(t, loaded.MCP)
}

// TestSaveMCPStateRemovesFileWhenNothingLeft checks that the file goes once neither skills, plugins
// nor MCP servers remain, and that it is not created for an empty MCP section.
func TestSaveMCPStateRemovesFileWhenNothingLeft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, stateFileName)

	require.NoError(t, SaveMCPState(dir, &MCPState{}))
	_, err := os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)

	require.NoError(t, SaveMCPState(dir, &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}))
	require.NoError(t, SaveMCPState(dir, &MCPState{}))
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// TestSaveMCPStateKeepsInstallerLoadAsNoInstall checks that a file holding only MCP records loads as no
// install, so the installer treats it as it treats a missing file, and that a later skills write keeps
// the MCP records.
func TestSaveMCPStateKeepsInstallerLoadAsNoInstall(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, SaveMCPState(dir, &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}))

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Nil(t, loaded)

	require.NoError(t, SaveState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion, Skills: map[string]string{"databricks": "1.0.0"}}))
	mcp, err := LoadMCPState(dir)
	require.NoError(t, err)
	assert.Equal(t, []MCPEntry{{Name: "github-logfood"}}, mcp.Servers)
}

// TestLoadStateHidesOnlyMCPOnlyFiles checks which files load as no install. A file with no MCP section
// loads as it always did, even when empty, since the installer wrote such files on main. Only a file
// holding MCP records and nothing the installer wrote loads as nil.
func TestLoadStateHidesOnlyMCPOnlyFiles(t *testing.T) {
	mcp := &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}
	tests := []struct {
		name  string
		state *InstallState
		want  bool
	}{
		{name: "mcp only", state: &InstallState{MCP: mcp}},
		{name: "empty without mcp", state: &InstallState{SchemaVersion: 2}, want: true},
		{name: "release without skills, with mcp", state: &InstallState{Release: "v0.1.5", MCP: mcp}, want: true},
		{name: "files without skills", state: &InstallState{Files: map[string]FileRecord{"databricks/SKILL.md": {SHA256: "abc"}}}, want: true},
		{name: "repo dirs without skills", state: &InstallState{RepoDirs: map[string]string{"databricks": "skills"}}, want: true},
		{name: "skills", state: &InstallState{Skills: map[string]string{"databricks": "1.0.0"}, MCP: mcp}, want: true},
		{name: "plugins", state: &InstallState{Plugins: map[string]PluginRecord{"claude-code": {Plugin: "databricks"}}, MCP: mcp}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, writeState(dir, tt.state))

			loaded, err := LoadState(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want, loaded != nil)
		})
	}
}

// TestSaveStateWritesEmptyState checks that SaveState writes the state it is given even when nothing
// is left in it, as the installer did before MCP records shared the file.
func TestSaveStateWritesEmptyState(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, SaveState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion, Skills: map[string]string{"databricks": "1.0.0"}}))

	require.NoError(t, SaveState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion}))
	loaded, err := LoadState(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Empty(t, loaded.Skills)
}

// TestSaveStateKeepsMCPRecordsWhenNothingElseRemains checks that SaveState, once the skills are gone,
// keeps the MCP servers recorded on disk.
func TestSaveStateKeepsMCPRecordsWhenNothingElseRemains(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, SaveState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion, Skills: map[string]string{"databricks": "1.0.0"}}))
	require.NoError(t, SaveMCPState(dir, &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}))

	require.NoError(t, SaveState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion}))
	mcp, err := LoadMCPState(dir)
	require.NoError(t, err)
	assert.Equal(t, []MCPEntry{{Name: "github-logfood"}}, mcp.Servers)
}

func TestLoadStateMigratesV2ToV3AndSavesV3(t *testing.T) {
	dir := t.TempDir()
	v2 := `{"schema_version":2,"release":"v0.2.0","last_updated":"2026-03-22T10:00:00Z","skills":{"databricks":"1.0.0"}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, stateFileName), []byte(v2), 0o644))

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, 3, loaded.SchemaVersion)
	assert.Nil(t, loaded.MCP)

	require.NoError(t, SaveState(dir, loaded))
	data, err := os.ReadFile(filepath.Join(dir, stateFileName))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"schema_version": 3`)
	assert.NotContains(t, string(data), `"mcp"`)
}
