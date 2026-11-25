package installer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// saveGlobalState writes state as the global install state of the test home.
func saveGlobalState(t *testing.T, state *InstallState) {
	t.Helper()
	dir, err := GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	state.SchemaVersion = CurrentSchemaVersion
	require.NoError(t, SaveState(dir, state))
}

// codexPlugin returns the plugin records of a state that records the Codex
// plugin at version.
func codexPlugin(version string) map[string]PluginRecord {
	return map[string]PluginRecord{agents.NameCodex: {Plugin: "databricks", Version: version}}
}

// writeCodexCLIPlugin writes the records `codex plugin add` leaves for the
// databricks plugin, which the install state never sees.
func writeCodexCLIPlugin(t *testing.T, home string) {
	t.Helper()
	codexHome := filepath.Join(home, ".codex")
	require.NoError(t, os.MkdirAll(filepath.Join(codexHome, "plugins", "cache", "databricks-agent-skills", "databricks", "0.2.50"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte("[plugins.\"databricks@databricks-agent-skills\"]\nenabled = true\n"), 0o600))
}

func TestOutdatedTooling(t *testing.T) {
	skills := map[string]string{"databricks-core": "0.1.0"}
	tests := []struct {
		name string
		// state is the global install state; nil leaves the home without one.
		state *InstallState
		// codexCLIPlugin gives Codex a plugin installed with its own CLI.
		codexCLIPlugin bool
		// pinnedRef, when set, pins the release `aitools update` installs
		// (DATABRICKS_SKILLS_REF). Otherwise the compatibility manifest tracks
		// the latest release, and its lookup answers v0.2.47 or latestErr.
		pinnedRef     string
		latestErr     error
		agent         string
		wantInstalled string
		wantAvailable string
		// wantLookups is the number of latest-release lookups made.
		wantLookups int
	}{
		{
			name:  "no install state",
			agent: agents.NameCursor,
		},
		{
			name:          "skills older than the target",
			state:         &InstallState{Release: "v0.2.10", Skills: skills},
			agent:         agents.NameCursor,
			wantInstalled: "0.2.10",
			wantAvailable: "0.2.47",
			wantLookups:   1,
		},
		{
			name:        "skills at the target",
			state:       &InstallState{Release: "v0.2.47", Skills: skills},
			agent:       agents.NameCursor,
			wantLookups: 1,
		},
		{
			// `aitools update` refuses the downgrade, so there is nothing to run.
			name:        "skills newer than the target",
			state:       &InstallState{Release: "v0.2.50", Skills: skills},
			agent:       agents.NameCursor,
			wantLookups: 1,
		},
		{
			// The agent loads its plugin, so its record decides, not the raw
			// skills next to it.
			name:          "plugin record older than the target, skills current",
			state:         &InstallState{Release: "v0.2.47", Skills: skills, Plugins: codexPlugin("0.2.30")},
			agent:         agents.NameCodex,
			wantInstalled: "0.2.30",
			wantAvailable: "0.2.47",
			wantLookups:   1,
		},
		{
			name:        "plugin record at the target, skills older",
			state:       &InstallState{Release: "v0.2.10", Skills: skills, Plugins: codexPlugin("0.2.47")},
			agent:       agents.NameCodex,
			wantLookups: 1,
		},
		{
			name:          "agent without a plugin record uses the skills release",
			state:         &InstallState{Release: "v0.2.10", Skills: skills, Plugins: codexPlugin("0.2.47")},
			agent:         agents.NameCursor,
			wantInstalled: "0.2.10",
			wantAvailable: "0.2.47",
			wantLookups:   1,
		},
		{
			// Its last update ran from a marketplace copy that could not be
			// refreshed, so what it holds is not known.
			name:  "plugin record without a version",
			state: &InstallState{Release: "v0.2.10", Skills: skills, Plugins: codexPlugin("")},
			agent: agents.NameCodex,
		},
		{
			// Codex loads that plugin, of a release the CLI does not know.
			name:           "plugin installed with the agent's own CLI, skills older",
			state:          &InstallState{Release: "v0.2.10", Skills: skills},
			codexCLIPlugin: true,
			agent:          agents.NameCodex,
		},
		{
			name:           "another agent's own plugin does not hide the skills release",
			state:          &InstallState{Release: "v0.2.10", Skills: skills},
			codexCLIPlugin: true,
			agent:          agents.NameCursor,
			wantInstalled:  "0.2.10",
			wantAvailable:  "0.2.47",
			wantLookups:    1,
		},
		{
			name:  "no raw skills and no plugin record for the agent",
			state: &InstallState{Release: "v0.2.10", Skills: map[string]string{}, Plugins: codexPlugin("0.2.10")},
			agent: agents.NameCursor,
		},
		{
			name:  "skills installed from a branch",
			state: &InstallState{Release: "main", Skills: skills},
			agent: agents.NameCursor,
		},
		{
			name:      "target is a branch",
			state:     &InstallState{Release: "v0.2.10", Skills: skills},
			pinnedRef: "main",
			agent:     agents.NameCursor,
		},
		{
			name:        "failed release lookup",
			state:       &InstallState{Release: "v0.2.10", Skills: skills},
			latestErr:   errors.New("no route to host"),
			agent:       agents.NameCursor,
			wantLookups: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := setupTestHome(t)
			t.Setenv("DATABRICKS_SKILLS_REF", tc.pinnedRef)
			withCachedCompat(t, skillsLatestSentinel)
			lookups := stubLatestSkillsReleaseRef(t, "v0.2.47", tc.latestErr)
			if tc.state != nil {
				saveGlobalState(t, tc.state)
			}
			if tc.codexCLIPlugin {
				writeCodexCLIPlugin(t, home)
			}

			installed, available, outdated := OutdatedTooling(t.Context(), tc.agent)
			assert.Equal(t, tc.wantInstalled, installed)
			assert.Equal(t, tc.wantAvailable, available)
			assert.Equal(t, tc.wantInstalled != "", outdated)
			// An installation without a known release costs no lookup.
			assert.Equal(t, tc.wantLookups, *lookups)
		})
	}
}

func TestOutdatedToolingIgnoresALookupThatRanOutOfTime(t *testing.T) {
	setupTestHome(t)
	saveGlobalState(t, &InstallState{Release: "v0.2.10", Skills: map[string]string{"databricks-core": "0.1.0"}})
	// The pinned ref resolves without the network, so only the bound decides.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.47")

	_, _, outdated := OutdatedTooling(t.Context(), agents.NameCursor)
	require.True(t, outdated)

	orig := outdatedLookupTimeout
	t.Cleanup(func() { outdatedLookupTimeout = orig })
	outdatedLookupTimeout = 0
	_, _, outdated = OutdatedTooling(t.Context(), agents.NameCursor)
	assert.False(t, outdated)
}
