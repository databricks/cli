package installer

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeManifestStampsSourceNameAndRepoDir(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillMeta{
			"databricks-apps":    {Version: "0.1.0", Files: []string{"SKILL.md"}},
			"databricks-iceberg": {Version: "0.0.1", Files: []string{"SKILL.md"}, RepoDir: experimentalRepoPath},
		},
	}

	normalizeManifest(m)

	stable := m.Skills["databricks-apps"]
	assert.False(t, stable.IsExperimental())
	assert.Equal(t, stableSkillsRepoPath, stable.RepoDir)
	assert.Equal(t, "databricks-apps", stable.SourceName)

	exp, ok := m.Skills["databricks-iceberg"]
	assert.True(t, ok)
	assert.True(t, exp.IsExperimental())
	assert.Equal(t, experimentalRepoPath, exp.RepoDir)
	assert.Equal(t, "databricks-iceberg", exp.SourceName)
}

func TestManifestHasExperimental(t *testing.T) {
	stableOnly := &Manifest{Skills: map[string]SkillMeta{
		"databricks-apps": {Version: "0.1.0"},
	}}
	normalizeManifest(stableOnly)
	assert.False(t, manifestHasExperimental(stableOnly))

	withExperimental := &Manifest{
		Skills: map[string]SkillMeta{
			"databricks-apps":    {Version: "0.1.0"},
			"databricks-iceberg": {Version: "0.0.1", RepoDir: experimentalRepoPath},
		},
	}
	normalizeManifest(withExperimental)
	assert.True(t, manifestHasExperimental(withExperimental))
}

func TestNormalizeManifestOnlyExperimentalSkills(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillMeta{
			"x": {Version: "0.0.1", RepoDir: experimentalRepoPath},
		},
	}

	normalizeManifest(m)

	got, ok := m.Skills["x"]
	assert.True(t, ok)
	assert.True(t, got.IsExperimental())
	assert.Equal(t, experimentalRepoPath, got.RepoDir)
	assert.Equal(t, "x", got.SourceName)
}

func TestFetchManifestSizeCap(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		// Whitespace pads a valid manifest to exactly the cap: it still parses.
		{name: "at the cap", body: strings.Repeat(" ", maxFetchBodySize-2) + "{}"},
		{name: "one byte over the cap", body: strings.Repeat(" ", maxFetchBodySize-1) + "{}", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", server.URL)

			_, err := (&GitHubManifestSource{}).FetchManifest(t.Context(), "v1")
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrManifestTooLarge)
			_, ok := errors.AsType[*FetchError](err)
			assert.True(t, ok)
			assert.Equal(t, "skills manifest is larger than 1 MiB", err.Error())
		})
	}
}

// TestFetchManifestRejectsPathsOutsideSkillDir covers manifest entries that
// would make an install write outside the skill's own directory: every skill
// name and file in the fetched manifest becomes a local path.
func TestFetchManifestRejectsPathsOutsideSkillDir(t *testing.T) {
	tests := []struct {
		name    string
		skill   string
		file    string
		wantErr string
	}{
		{name: "regular file", skill: "databricks-core", file: "SKILL.md"},
		{name: "nested file", skill: "databricks-core", file: "references/auth.md"},
		{name: "file escaping the skill dir", skill: "databricks-core", file: "../../../.bashrc", wantErr: `skill "databricks-core" lists file "../../../.bashrc" outside its directory`},
		{name: "absolute file", skill: "databricks-core", file: "/etc/passwd", wantErr: `skill "databricks-core" lists file "/etc/passwd" outside its directory`},
		{name: "file escaping after a subdirectory", skill: "databricks-core", file: "references/../../x", wantErr: `skill "databricks-core" lists file "references/../../x" outside its directory`},
		{name: "name with digits", skill: "ai2-functions", file: "SKILL.md"},
		{name: "64-character name", skill: strings.Repeat("a", 64), file: "SKILL.md"},
		{name: "skill name escaping the skills dir", skill: "../.ssh", file: "SKILL.md", wantErr: `skill name "../.ssh" must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "nested skill name", skill: "a/b", file: "SKILL.md", wantErr: `skill name "a/b" must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "dot skill name", skill: ".", file: "SKILL.md", wantErr: `skill name "." must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "installer state file name", skill: ".state.json", file: "SKILL.md", wantErr: `skill name ".state.json" must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "uppercase name", skill: "Databricks-Core", file: "SKILL.md", wantErr: `skill name "Databricks-Core" must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "trailing dot", skill: "databricks-core.", file: "SKILL.md", wantErr: `skill name "databricks-core." must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "trailing space", skill: "databricks-core ", file: "SKILL.md", wantErr: `skill name "databricks-core " must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "underscore", skill: "databricks_core", file: "SKILL.md", wantErr: `skill name "databricks_core" must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "leading hyphen", skill: "-databricks", file: "SKILL.md", wantErr: `skill name "-databricks" must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "double hyphen", skill: "databricks--core", file: "SKILL.md", wantErr: `skill name "databricks--core" must be at most 64 lowercase letters, digits and single hyphens`},
		{name: "Windows device name nul", skill: "nul", file: "SKILL.md", wantErr: `skill name "nul" is a reserved Windows device name`},
		{name: "Windows device name con", skill: "con", file: "SKILL.md", wantErr: `skill name "con" is a reserved Windows device name`},
		{name: "Windows device name com1", skill: "com1", file: "SKILL.md", wantErr: `skill name "com1" is a reserved Windows device name`},
		{name: "Windows device name lpt9", skill: "lpt9", file: "SKILL.md", wantErr: `skill name "lpt9" is a reserved Windows device name`},
		{name: "device name prefix", skill: "console", file: "SKILL.md"},
		{name: "device name with a suffix", skill: "nul-tools", file: "SKILL.md"},
		{name: "65-character name", skill: strings.Repeat("a", 65), file: "SKILL.md", wantErr: fmt.Sprintf(`skill name %q must be at most 64 lowercase letters, digits and single hyphens`, strings.Repeat("a", 65))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"version":"1","skills":{%q:{"version":"0.1.0","files":[%q]}}}`, tc.skill, tc.file)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			t.Cleanup(server.Close)
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", server.URL)

			manifest, err := (&GitHubManifestSource{}).FetchManifest(t.Context(), "v1")
			if tc.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, []string{tc.file}, manifest.Skills[tc.skill].Files)
				return
			}
			assert.EqualError(t, err, "invalid skills manifest: "+tc.wantErr)
			_, ok := errors.AsType[*FetchError](err)
			assert.True(t, ok)
		})
	}
}

func TestInstallRejectsStateFileSkillNameWithoutTouchingState(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: testManifest()}, []*agents.Agent{agent}, InstallOptions{}))

	statePath := filepath.Join(tmp, ".databricks", "aitools", "skills", stateFileName)
	before, err := os.ReadFile(statePath)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"1","skills":{".state.json":{"version":"0.1.0","files":["SKILL.md"]}}}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", server.URL)

	err = InstallSkillsForAgents(ctx, &GitHubManifestSource{}, []*agents.Agent{agent}, InstallOptions{})
	require.ErrorContains(t, err, `skill name ".state.json"`)

	after, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}
