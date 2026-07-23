package apps

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGitRepositoryFlags wires gitRepositoryFlags onto a fresh command, sets the
// given flags, runs the PreRunE, and returns the resulting pointer + error.
func runGitRepositoryFlags(t *testing.T, argv []string) (*apps.GitRepository, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var target *apps.GitRepository
	pre := gitRepositoryFlags(cmd, &target)
	require.NoError(t, cmd.ParseFlags(argv))
	return target, pre(cmd, nil)
}

func runGitSourceFlags(t *testing.T, argv []string) (*apps.GitSource, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var target *apps.GitSource
	pre := gitSourceFlags(cmd, &target)
	require.NoError(t, cmd.ParseFlags(argv))
	return target, pre(cmd, nil)
}

func TestGitRepositoryFlags(t *testing.T) {
	t.Run("no flags leaves target nil", func(t *testing.T) {
		target, err := runGitRepositoryFlags(t, nil)
		require.NoError(t, err)
		assert.Nil(t, target)
	})

	t.Run("url and provider populate the struct", func(t *testing.T) {
		target, err := runGitRepositoryFlags(t, []string{
			"--git-url", "https://github.com/databricks/git_app_repo.git",
			"--git-provider", "gitHub",
		})
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, "https://github.com/databricks/git_app_repo.git", target.Url)
		assert.Equal(t, "gitHub", target.Provider)
	})

	t.Run("url without provider errors", func(t *testing.T) {
		_, err := runGitRepositoryFlags(t, []string{"--git-url", "https://github.com/x/y"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be set together")
	})

	t.Run("provider without url errors", func(t *testing.T) {
		_, err := runGitRepositoryFlags(t, []string{"--git-provider", "gitHub"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be set together")
	})
}

func TestGitSourceFlags(t *testing.T) {
	t.Run("no flags leaves target nil", func(t *testing.T) {
		target, err := runGitSourceFlags(t, nil)
		require.NoError(t, err)
		assert.Nil(t, target)
	})

	t.Run("branch populates the struct", func(t *testing.T) {
		target, err := runGitSourceFlags(t, []string{"--git-branch", "main"})
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, "main", target.Branch)
		assert.Empty(t, target.Tag)
		assert.Empty(t, target.Commit)
	})

	t.Run("commit with source-code-path populates both", func(t *testing.T) {
		target, err := runGitSourceFlags(t, []string{
			"--git-commit", "abc123",
			"--git-source-code-path", "my-app",
		})
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, "abc123", target.Commit)
		assert.Equal(t, "my-app", target.SourceCodePath)
	})

	t.Run("source-code-path without a ref errors", func(t *testing.T) {
		_, err := runGitSourceFlags(t, []string{"--git-source-code-path", "my-app"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires one of")
	})
}
