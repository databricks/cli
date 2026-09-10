package apps

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGitRepositoryFlags wires gitRepositoryFlags (in create mode, so
// --caller-git-credential-id is available) onto a fresh command, sets the given
// flags, runs the PreRunE, and returns the resulting pointer + error.
func runGitRepositoryFlags(t *testing.T, argv []string) (*apps.GitRepository, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var target *apps.GitRepository
	pre := gitRepositoryFlags(cmd, &target, true)
	require.NoError(t, cmd.ParseFlags(argv))
	return target, pre(cmd, nil)
}

func runGitSourceFlags(t *testing.T, argv []string) (*apps.GitSource, error) {
	t.Helper()
	cmd := &cobra.Command{}
	// The generated deploy command registers --source-code-path (the workspace
	// source). Register it here too so the workspace-vs-Git mutual-exclusion
	// guard can be exercised.
	cmd.Flags().String("source-code-path", "", "")
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

	t.Run("auto-deploy and caller credential populate the struct", func(t *testing.T) {
		target, err := runGitRepositoryFlags(t, []string{
			"--git-url", "https://github.com/databricks/git_app_repo.git",
			"--git-provider", "gitHub",
			"--git-auto-deploy",
			"--caller-git-credential-id", "93488329053511",
		})
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.True(t, target.AutoDeploy)
		assert.Equal(t, int64(93488329053511), target.CallerCredentialId)
	})

	t.Run("auto-deploy=false is force-sent", func(t *testing.T) {
		target, err := runGitRepositoryFlags(t, []string{
			"--git-url", "https://github.com/x/y",
			"--git-provider", "gitHub",
			"--git-auto-deploy=false",
		})
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.False(t, target.AutoDeploy)
		// Otherwise omitempty would drop the field and the server could not
		// distinguish "disable auto-deploy" from "unset".
		assert.Contains(t, target.ForceSendFields, "AutoDeploy")
	})

	t.Run("auto-deploy without url and provider errors", func(t *testing.T) {
		_, err := runGitRepositoryFlags(t, []string{"--git-auto-deploy"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "require --git-url and --git-provider")
	})

	t.Run("caller credential without url and provider errors", func(t *testing.T) {
		_, err := runGitRepositoryFlags(t, []string{"--caller-git-credential-id", "42"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "require --git-url and --git-provider")
	})

	t.Run("caller credential is create-only", func(t *testing.T) {
		// The server rejects caller_credential_id on UpdateApp, so update mode
		// (withCallerCredential=false) must not register the flag at all.
		cmd := &cobra.Command{}
		var target *apps.GitRepository
		gitRepositoryFlags(cmd, &target, false)
		assert.Nil(t, cmd.Flags().Lookup("caller-git-credential-id"))
		// The auto-deploy modifier is still available on update.
		assert.NotNil(t, cmd.Flags().Lookup("git-auto-deploy"))
	})

	t.Run("update mode reports only auto-deploy in the requires-repo error", func(t *testing.T) {
		cmd := &cobra.Command{}
		var target *apps.GitRepository
		pre := gitRepositoryFlags(cmd, &target, false)
		require.NoError(t, cmd.ParseFlags([]string{"--git-auto-deploy"}))
		err := pre(cmd, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--git-auto-deploy require")
		assert.NotContains(t, err.Error(), "caller-git-credential-id")
	})
}

// runAutoDeployBranchRule mimics the create command: it registers both the git
// repository and git source flags, parses argv, and runs requireBranchForAutoDeploy.
func runAutoDeployBranchRule(t *testing.T, argv []string) error {
	t.Helper()
	cmd := &cobra.Command{}
	var repo *apps.GitRepository
	var src *apps.GitSource
	gitRepositoryFlags(cmd, &repo, true)
	gitSourceFlags(cmd, &src)
	require.NoError(t, cmd.ParseFlags(argv))
	return requireBranchForAutoDeploy(cmd, nil)
}

func TestRequireBranchForAutoDeploy(t *testing.T) {
	repo := []string{"--git-url", "https://github.com/x/y", "--git-provider", "gitHub"}

	t.Run("auto-deploy on with a branch is allowed", func(t *testing.T) {
		err := runAutoDeployBranchRule(t, append(repo, "--git-auto-deploy", "--git-branch", "main"))
		require.NoError(t, err)
	})

	t.Run("auto-deploy on without a branch errors", func(t *testing.T) {
		err := runAutoDeployBranchRule(t, append(repo, "--git-auto-deploy"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires --git-branch")
	})

	t.Run("auto-deploy on with only a tag errors", func(t *testing.T) {
		err := runAutoDeployBranchRule(t, append(repo, "--git-auto-deploy", "--git-tag", "v1.0.0"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires --git-branch")
	})

	t.Run("auto-deploy off does not require a branch", func(t *testing.T) {
		err := runAutoDeployBranchRule(t, append(repo, "--git-auto-deploy=false"))
		require.NoError(t, err)
	})

	t.Run("auto-deploy unset does not require a branch", func(t *testing.T) {
		err := runAutoDeployBranchRule(t, repo)
		require.NoError(t, err)
	})

	t.Run("no-op when the command has no git-branch flag", func(t *testing.T) {
		// A command that wires only the repository flags has no branch to
		// demand, so the rule must stay out of its way.
		cmd := &cobra.Command{}
		var target *apps.GitRepository
		gitRepositoryFlags(cmd, &target, true)
		require.NoError(t, cmd.ParseFlags([]string{"--git-url", "https://github.com/x/y", "--git-provider", "gitHub", "--git-auto-deploy"}))
		require.NoError(t, requireBranchForAutoDeploy(cmd, nil))
	})
}

// runRepoForGitSourceRule mimics the create command (both flag groups) and runs
// requireRepoForGitSourceOnCreate.
func runRepoForGitSourceRule(t *testing.T, argv []string) error {
	t.Helper()
	cmd := &cobra.Command{}
	var repo *apps.GitRepository
	var src *apps.GitSource
	gitRepositoryFlags(cmd, &repo, true)
	gitSourceFlags(cmd, &src)
	require.NoError(t, cmd.ParseFlags(argv))
	return requireRepoForGitSourceOnCreate(cmd, nil)
}

func TestRequireRepoForGitSourceOnCreate(t *testing.T) {
	repo := []string{"--git-url", "https://github.com/x/y", "--git-provider", "gitHub"}

	t.Run("git source with a repo is allowed", func(t *testing.T) {
		require.NoError(t, runRepoForGitSourceRule(t, append(repo, "--git-branch", "main")))
	})

	t.Run("branch without a repo errors", func(t *testing.T) {
		err := runRepoForGitSourceRule(t, []string{"--git-branch", "main"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "require --git-url and --git-provider on create")
	})

	t.Run("tag without a repo errors", func(t *testing.T) {
		err := runRepoForGitSourceRule(t, []string{"--git-tag", "v1.0.0"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "require --git-url and --git-provider on create")
	})

	t.Run("source-code-path with a ref but no repo errors", func(t *testing.T) {
		err := runRepoForGitSourceRule(t, []string{"--git-commit", "abc123", "--git-source-code-path", "my-app"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "require --git-url and --git-provider on create")
	})

	t.Run("no git source flags is a no-op", func(t *testing.T) {
		require.NoError(t, runRepoForGitSourceRule(t, repo))
	})
}

func TestApplyPreservedAutoDeploy(t *testing.T) {
	t.Run("carries a currently-enabled auto_deploy forward", func(t *testing.T) {
		repo := &apps.GitRepository{Url: "https://github.com/x/y", Provider: "gitHub"}
		applyPreservedAutoDeploy(repo, true)
		assert.True(t, repo.AutoDeploy)
	})

	t.Run("leaves a currently-disabled auto_deploy off", func(t *testing.T) {
		repo := &apps.GitRepository{Url: "https://github.com/x/y", Provider: "gitHub"}
		applyPreservedAutoDeploy(repo, false)
		assert.False(t, repo.AutoDeploy)
	})

	t.Run("nil repo is a no-op", func(t *testing.T) {
		assert.NotPanics(t, func() { applyPreservedAutoDeploy(nil, true) })
	})
}

// newPreserveCmd builds an update-like command with the repo flags and a --json
// flag so the preserve guards can be exercised without a workspace client.
func newPreserveCmd(t *testing.T, req *apps.UpdateAppRequest, argv []string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("json", "", "")
	gitRepositoryFlags(cmd, &req.App.GitRepository, false)
	require.NoError(t, cmd.ParseFlags(argv))
	return cmd
}

func TestPreserveAutoDeployOnUpdate_Guards(t *testing.T) {
	// Each guard returns before the GetApp call, so no client is needed. If a
	// guard failed to short-circuit, WorkspaceClient(ctx) would panic here.
	t.Run("skips when --json is set", func(t *testing.T) {
		req := &apps.UpdateAppRequest{}
		cmd := newPreserveCmd(t, req, []string{"--json", "{}", "--git-url", "https://github.com/x/y", "--git-provider", "gitHub"})
		require.NoError(t, preserveAutoDeployOnUpdate(req)(cmd, []string{"my-app"}))
	})

	t.Run("skips when --git-auto-deploy was passed", func(t *testing.T) {
		req := &apps.UpdateAppRequest{}
		cmd := newPreserveCmd(t, req, []string{"--git-url", "https://github.com/x/y", "--git-provider", "gitHub", "--git-auto-deploy"})
		require.NoError(t, preserveAutoDeployOnUpdate(req)(cmd, []string{"my-app"}))
	})

	t.Run("skips when no repository change is being sent", func(t *testing.T) {
		req := &apps.UpdateAppRequest{}
		cmd := newPreserveCmd(t, req, nil) // GitRepository stays nil
		require.NoError(t, preserveAutoDeployOnUpdate(req)(cmd, []string{"my-app"}))
	})

	t.Run("skips when no app name is available", func(t *testing.T) {
		req := &apps.UpdateAppRequest{}
		cmd := newPreserveCmd(t, req, []string{"--git-url", "https://github.com/x/y", "--git-provider", "gitHub"})
		require.NoError(t, preserveAutoDeployOnUpdate(req)(cmd, nil))
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

	t.Run("workspace source-code-path combined with a git ref errors", func(t *testing.T) {
		_, err := runGitSourceFlags(t, []string{
			"--git-branch", "main",
			"--source-code-path", "/Workspace/Users/me/app",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be combined")
	})

	t.Run("workspace source-code-path alone leaves git source nil", func(t *testing.T) {
		target, err := runGitSourceFlags(t, []string{"--source-code-path", "/Workspace/Users/me/app"})
		require.NoError(t, err)
		assert.Nil(t, target)
	})
}
