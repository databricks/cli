package installer

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/skills"
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

	skills.NormalizeManifest(m)

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
	skills.NormalizeManifest(stableOnly)
	assert.False(t, manifestHasExperimental(stableOnly))

	withExperimental := &Manifest{
		Skills: map[string]SkillMeta{
			"databricks-apps":    {Version: "0.1.0"},
			"databricks-iceberg": {Version: "0.0.1", RepoDir: experimentalRepoPath},
		},
	}
	skills.NormalizeManifest(withExperimental)
	assert.True(t, manifestHasExperimental(withExperimental))
}

func TestNormalizeManifestOnlyExperimentalSkills(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillMeta{
			"x": {Version: "0.0.1", RepoDir: experimentalRepoPath},
		},
	}

	skills.NormalizeManifest(m)

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
		{name: "at the cap", body: strings.Repeat(" ", skills.MaxFetchBodySize-2) + "{}"},
		{name: "one byte over the cap", body: strings.Repeat(" ", skills.MaxFetchBodySize-1) + "{}", wantErr: true},
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

// TestCheckManifestPaths covers every skill name and file in a fetched
// manifest becoming a local path: an escape fails the manifest, and an entry
// that is not portable makes unportableSkill skip only its skill.
func TestCheckManifestPaths(t *testing.T) {
	tests := []struct {
		name  string
		skill string
		file  string
		// wantErr is the error after the ErrUnsafeManifestPath prefix.
		wantErr string
		// wantSkipped is true when unportableSkill skips the skill.
		wantSkipped bool
	}{
		{name: "regular file", skill: "databricks-core", file: "SKILL.md"},
		{name: "nested file", skill: "databricks-core", file: "references/auth.md"},
		{name: "name with digits", skill: "ai2-functions", file: "SKILL.md"},
		{name: "64-character name", skill: strings.Repeat("a", 64), file: "SKILL.md"},
		{name: "device name prefix", skill: "console", file: "SKILL.md"},
		{name: "device name with a suffix", skill: "nul-tools", file: "SKILL.md"},
		{name: "file with a device name prefix", skill: "databricks-core", file: "references/console.md"},

		{name: "file escaping the skill dir", skill: "databricks-core", file: "../../../.bashrc", wantErr: `skill "databricks-core" lists file "../../../.bashrc"`},
		{name: "absolute file", skill: "databricks-core", file: "/etc/passwd", wantErr: `skill "databricks-core" lists file "/etc/passwd"`},
		{name: "file escaping after a subdirectory", skill: "databricks-core", file: "references/../../x", wantErr: `skill "databricks-core" lists file "references/../../x"`},
		{name: "file escaping with backslashes", skill: "databricks-core", file: `a\..\..\x`, wantErr: `skill "databricks-core" lists file "a\\..\\..\\x"`},
		{name: "file rooted with a backslash", skill: "databricks-core", file: `\x`, wantErr: `skill "databricks-core" lists file "\\x"`},
		{name: "file on a Windows drive", skill: "databricks-core", file: "C:x", wantErr: `skill "databricks-core" lists file "C:x"`},
		{name: "skill name escaping the skills dir", skill: "../.ssh", file: "SKILL.md", wantErr: `skill name "../.ssh"`},
		{name: "nested skill name", skill: "a/b", file: "SKILL.md", wantErr: `skill name "a/b"`},
		{name: "skill name with a backslash", skill: `a\b`, file: "SKILL.md", wantErr: `skill name "a\\b"`},
		{name: "dot skill name", skill: ".", file: "SKILL.md", wantErr: `skill name "."`},
		{name: "empty skill name", skill: "", file: "SKILL.md", wantErr: `skill name ""`},
		{name: "installer state file name", skill: ".state.json", file: "SKILL.md", wantErr: `skill name ".state.json"`},

		{name: "uppercase name", skill: "Databricks-Core", file: "SKILL.md", wantSkipped: true},
		{name: "trailing dot", skill: "databricks-core.", file: "SKILL.md", wantSkipped: true},
		{name: "trailing space", skill: "databricks-core ", file: "SKILL.md", wantSkipped: true},
		{name: "underscore", skill: "databricks_core", file: "SKILL.md", wantSkipped: true},
		{name: "leading hyphen", skill: "-databricks", file: "SKILL.md", wantSkipped: true},
		{name: "double hyphen", skill: "databricks--core", file: "SKILL.md", wantSkipped: true},
		{name: "65-character name", skill: strings.Repeat("a", 65), file: "SKILL.md", wantSkipped: true},
		{name: "Windows device name nul", skill: "nul", file: "SKILL.md", wantSkipped: true},
		{name: "Windows device name com1", skill: "com1", file: "SKILL.md", wantSkipped: true},
		{name: "Windows device name lpt9", skill: "lpt9", file: "SKILL.md", wantSkipped: true},
		{name: "file is the skill dir", skill: "databricks-core", file: ".", wantSkipped: true},
		{name: "file resolving to the skill dir", skill: "databricks-core", file: "references/..", wantSkipped: true},
		{name: "file with a dot element", skill: "databricks-core", file: "./SKILL.md", wantSkipped: true},
		{name: "file with a .. element that stays inside", skill: "databricks-core", file: "references/../SKILL.md", wantSkipped: true},
		{name: "file with a double slash", skill: "databricks-core", file: "references//auth.md", wantSkipped: true},
		{name: "empty file", skill: "databricks-core", file: "", wantSkipped: true},
		{name: "file with a backslash", skill: "databricks-core", file: `references\auth.md`, wantSkipped: true},
		{name: "file with a colon", skill: "databricks-core", file: "notes:v2.md", wantSkipped: true},
		{name: "file named like a device", skill: "databricks-core", file: "references/aux.md", wantSkipped: true},
		{name: "directory named like a device", skill: "databricks-core", file: "CON/SKILL.md", wantSkipped: true},
		{name: "file with a question mark", skill: "databricks-core", file: "references/what?.md", wantSkipped: true},
		{name: "file with a control character", skill: "databricks-core", file: "references/a\x1b.md", wantSkipped: true},
		{name: "file with a superscript device name", skill: "databricks-core", file: "COM\u00b9.md", wantSkipped: true},
		{name: "console device file", skill: "databricks-core", file: "CONIN$", wantSkipped: true},
		{name: "device stem with a trailing space", skill: "databricks-core", file: "aux .md", wantSkipped: true},
		{name: "element with a trailing dot", skill: "databricks-core", file: "references./auth.md", wantSkipped: true},
		{name: "file with a trailing dot", skill: "databricks-core", file: "SKILL.md.", wantSkipped: true},
		{name: "dotfile", skill: "databricks-core", file: "references/.gitkeep"},
		{name: "underscore and hyphen in a file", skill: "databricks-core", file: "scripts/run_job-v2.py"},
		{name: "file with plus signs", skill: "databricks-core", file: "references/c++.md"},
		{name: "file with an at sign", skill: "databricks-core", file: "assets/logo@2x.png"},
		{name: "template folder with braces", skill: "databricks-core", file: "{{.project_name}}/databricks.yml"},
		{name: "file with an inner space", skill: "databricks-core", file: "references/getting started.md"},
		{name: "file with parentheses, comma and tilde", skill: "databricks-core", file: "notes (v2),~draft.md"},
		{name: "non-ASCII file", skill: "databricks-core", file: "references/caf\u00e9.md"},
		{name: "element with a trailing space", skill: "databricks-core", file: "references /auth.md", wantSkipped: true},
		{name: "file with a pipe", skill: "databricks-core", file: "a|b.md", wantSkipped: true},
		{name: "file with a quote", skill: "databricks-core", file: `a"b.md`, wantSkipped: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manifest{Skills: map[string]SkillMeta{tc.skill: {Files: []string{tc.file}}}}
			err := skills.CheckManifestPaths(m)
			if tc.wantErr != "" {
				require.ErrorIs(t, err, ErrUnsafeManifestPath)
				assert.EqualError(t, err, ErrUnsafeManifestPath.Error()+": "+tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantSkipped, unportableSkill(tc.skill, []string{tc.file}) != "")
		})
	}
}

func TestUnportableSkillPathClashes(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{name: "files differing only in case", files: []string{"SKILL.md", "skill.md"}, want: `paths "SKILL.md" and "skill.md" differ only in case`},
		{name: "folders differing only in case", files: []string{"refs/a.md", "REFS/b.md"}, want: `paths "refs" and "REFS" differ only in case`},
		{name: "file and folder differing only in case", files: []string{"Refs", "refs/a.md"}, want: `paths "Refs" and "refs" differ only in case`},
		{name: "path used as a file and a folder", files: []string{"scripts", "scripts/run.py"}, want: `path "scripts" is both a file and a folder`},
		{name: "folder then file of the same path", files: []string{"scripts/run.py", "scripts"}, want: `path "scripts" is both a file and a folder`},
		{name: "same file listed twice", files: []string{"SKILL.md", "SKILL.md"}},
		{name: "same name in different folders", files: []string{"SKILL.md", "references/SKILL.md"}},
		{name: "files sharing a folder", files: []string{"refs/a.md", "refs/b.md"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, unportableSkill("databricks-core", tc.files))
		})
	}
}

func TestCheckManifestPathsReportsTheFirstSkillInOrder(t *testing.T) {
	m := &Manifest{Skills: map[string]SkillMeta{
		"b-skill": {Files: []string{"../b"}},
		"a-skill": {Files: []string{"../a"}},
		"../c":    {Files: []string{"SKILL.md"}},
	}}
	for range 10 {
		assert.EqualError(t, skills.CheckManifestPaths(m), ErrUnsafeManifestPath.Error()+`: skill name "../c"`)
	}
}

// TestFetchSkillsManifestChecksEverySource checks the manifest at the point
// every command fetches it through, whatever the source.
func TestFetchSkillsManifestChecksEverySource(t *testing.T) {
	src := &mockManifestSource{manifest: &Manifest{Skills: map[string]SkillMeta{
		"databricks-core": {Files: []string{"../../.bashrc"}},
	}}}
	_, _, err := FetchSkillsManifestWithFallback(t.Context(), src, "v1", false)
	require.ErrorIs(t, err, ErrUnsafeManifestPath)
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
	require.ErrorIs(t, err, ErrUnsafeManifestPath)

	after, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestInstallSkipsUnportableSkill(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	m := testManifest()
	m.Skills["Databricks_Core"] = SkillMeta{Version: "0.1.0", Files: []string{"SKILL.md"}}
	agent := testAgent(tmp)

	require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: m}, []*agents.Agent{agent}, InstallOptions{}))
	skillsDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	assert.DirExists(t, filepath.Join(skillsDir, "databricks-sql"))
	assert.NoDirExists(t, filepath.Join(skillsDir, "Databricks_Core"))

	err := InstallSkillsForAgents(ctx, &mockManifestSource{manifest: m}, []*agents.Agent{agent}, InstallOptions{SpecificSkills: []string{"Databricks_Core"}})
	skillErr, ok := errors.AsType[*SkillError](err)
	require.True(t, ok, "got %v", err)
	assert.Equal(t, ReasonUnportableSkill, skillErr.Reason)
}
