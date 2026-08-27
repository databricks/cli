package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderForHost(t *testing.T) {
	cases := map[string]string{
		"github.com":             "gitHub",
		"GitHub.com":             "gitHub",
		"gitlab.com":             "gitLab",
		"bitbucket.org":          "bitbucketCloud",
		"dev.azure.com":          "azureDevOpsServices",
		"myorg.visualstudio.com": "azureDevOpsServices",
		"git.internal.corp":      "",
		"github.example.com":     "",
		"":                       "",
	}
	for host, want := range cases {
		assert.Equal(t, want, providerForHost(host), "host=%q", host)
	}
}

func TestNormalizeGitOrigin(t *testing.T) {
	cases := []struct {
		name         string
		raw          string
		wantURL      string
		wantProvider string
	}{
		{"https", "https://github.com/my-org/my-repo", "https://github.com/my-org/my-repo", "gitHub"},
		{"https dot git", "https://github.com/my-org/my-repo.git", "https://github.com/my-org/my-repo", "gitHub"},
		{"https trailing slash", "https://github.com/my-org/my-repo/", "https://github.com/my-org/my-repo", "gitHub"},
		{"scp ssh", "git@github.com:my-org/my-repo.git", "https://github.com/my-org/my-repo", "gitHub"},
		{"ssh scheme", "ssh://git@github.com/my-org/my-repo.git", "https://github.com/my-org/my-repo", "gitHub"},
		{"gitlab", "https://gitlab.com/my-org/my-repo.git", "https://gitlab.com/my-org/my-repo", "gitLab"},
		{"azure", "git@ssh.dev.azure.com:v3/org/proj/repo", "", ""}, // ssh.dev.azure.com is not dev.azure.com
		{"unknown host", "https://git.internal.corp/my-org/my-repo.git", "", ""},
		{"empty path", "https://github.com/", "", ""},
		{"empty", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, gotProvider := normalizeGitOrigin(tc.raw)
			assert.Equal(t, tc.wantURL, gotURL)
			assert.Equal(t, tc.wantProvider, gotProvider)
		})
	}
}

// writeGitRepo lays down a minimal .git directory (config + HEAD) that
// git.FetchRepositoryInfo reads for local paths. headRef is the raw HEAD
// contents, e.g. "ref: refs/heads/main" for a branch or a bare SHA for a
// detached checkout.
func writeGitRepo(t *testing.T, dir, originURL, headRef string) {
	t.Helper()
	gitDir := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))

	config := "[core]\n\trepositoryformatversion = 0\n"
	if originURL != "" {
		config += "[remote \"origin\"]\n\turl = " + originURL + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(headRef), 0o644))
}

func TestDetectGitScaffoldSource(t *testing.T) {
	t.Run("repo root in-place", func(t *testing.T) {
		dir := t.TempDir()
		writeGitRepo(t, dir, "https://github.com/my-org/my-repo.git", "ref: refs/heads/main")

		got := detectGitScaffoldSource(t.Context(), dir, nil)
		assert.Equal(t, gitScaffoldSource{
			URL:            "https://github.com/my-org/my-repo",
			Provider:       "gitHub",
			Branch:         "main",
			SourceCodePath: "./",
		}, got)
		assert.True(t, got.active())
	})

	t.Run("app in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		writeGitRepo(t, dir, "git@github.com:my-org/my-repo.git", "ref: refs/heads/feature")
		sub := filepath.Join(dir, "apps", "my-app")
		require.NoError(t, os.MkdirAll(sub, 0o755))

		got := detectGitScaffoldSource(t.Context(), sub, nil)
		assert.Equal(t, "apps/my-app", got.SourceCodePath)
		assert.Equal(t, "https://github.com/my-org/my-repo", got.URL)
		assert.Equal(t, "feature", got.Branch)
	})

	t.Run("not-yet-created subdir probes parent", func(t *testing.T) {
		dir := t.TempDir()
		writeGitRepo(t, dir, "https://github.com/my-org/my-repo.git", "ref: refs/heads/main")
		// The subdir does not exist yet, mirroring `apps init --name my-app`.
		got := detectGitScaffoldSource(t.Context(), filepath.Join(dir, "my-app"), nil)
		assert.Equal(t, "my-app", got.SourceCodePath)
		assert.True(t, got.active())
	})

	t.Run("no repository", func(t *testing.T) {
		got := detectGitScaffoldSource(t.Context(), t.TempDir(), nil)
		assert.Equal(t, gitScaffoldSource{}, got)
		assert.False(t, got.active())
	})

	t.Run("no origin remote", func(t *testing.T) {
		dir := t.TempDir()
		writeGitRepo(t, dir, "", "ref: refs/heads/main")
		assert.Equal(t, gitScaffoldSource{}, detectGitScaffoldSource(t.Context(), dir, nil))
	})

	t.Run("unknown provider host", func(t *testing.T) {
		dir := t.TempDir()
		writeGitRepo(t, dir, "https://git.internal.corp/my-org/my-repo.git", "ref: refs/heads/main")
		assert.Equal(t, gitScaffoldSource{}, detectGitScaffoldSource(t.Context(), dir, nil))
	})

	t.Run("detached HEAD has no branch", func(t *testing.T) {
		dir := t.TempDir()
		writeGitRepo(t, dir, "https://github.com/my-org/my-repo.git", "0000000000000000000000000000000000000000")
		assert.Equal(t, gitScaffoldSource{}, detectGitScaffoldSource(t.Context(), dir, nil))
	})
}
