package installer

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
