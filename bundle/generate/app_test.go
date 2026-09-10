package generate

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertAppToValueWorkspaceSource(t *testing.T) {
	app := &apps.App{Name: "my-app", Description: "desc"}

	v, err := ConvertAppToValue(app, "../src/app")
	require.NoError(t, err)

	scp, err := dyn.Get(v, "source_code_path")
	require.NoError(t, err)
	assert.Equal(t, "../src/app", scp.MustString())

	_, err = dyn.Get(v, "git_repository")
	assert.Error(t, err)
	_, err = dyn.Get(v, "git_source")
	assert.Error(t, err)
}

func TestConvertAppToValueGitBacked(t *testing.T) {
	app := &apps.App{
		Name:        "my-app",
		Description: "desc",
		GitRepository: &apps.GitRepository{
			Url:        "https://github.com/my-org/my-repo",
			Provider:   "gitHub",
			AutoDeploy: true,
		},
		GitSource: &apps.GitSource{
			Branch:         "main",
			SourceCodePath: "apps/my-app",
			// System-populated; must not be written back.
			ResolvedCommit: "abc123",
		},
	}

	// The workspace source path is passed but must be ignored for a git-backed app.
	v, err := ConvertAppToValue(app, "../src/app")
	require.NoError(t, err)

	_, err = dyn.Get(v, "source_code_path")
	assert.Error(t, err, "git-backed app must not emit a workspace source_code_path")

	url, err := dyn.Get(v, "git_repository.url")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/my-org/my-repo", url.MustString())

	provider, err := dyn.Get(v, "git_repository.provider")
	require.NoError(t, err)
	assert.Equal(t, "gitHub", provider.MustString())

	autoDeploy, err := dyn.Get(v, "git_repository.auto_deploy")
	require.NoError(t, err)
	assert.True(t, autoDeploy.MustBool())

	branch, err := dyn.Get(v, "git_source.branch")
	require.NoError(t, err)
	assert.Equal(t, "main", branch.MustString())

	gscp, err := dyn.Get(v, "git_source.source_code_path")
	require.NoError(t, err)
	assert.Equal(t, "apps/my-app", gscp.MustString())

	_, err = dyn.Get(v, "git_source.resolved_commit")
	assert.Error(t, err, "resolved_commit is output-only and must be omitted")
}

func TestConvertAppToValueGitBackedDefaultSourceFallback(t *testing.T) {
	app := &apps.App{
		Name:          "my-app",
		GitRepository: &apps.GitRepository{Url: "https://github.com/my-org/my-repo", Provider: "gitHub"},
		// git_source unset; fall back to the deployed reference in default_git_source.
		DefaultGitSource: &apps.GitSource{Tag: "v1.0.0"},
	}

	v, err := ConvertAppToValue(app, "../src/app")
	require.NoError(t, err)

	tag, err := dyn.Get(v, "git_source.tag")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", tag.MustString())

	// auto_deploy is omitted when unset.
	_, err = dyn.Get(v, "git_repository.auto_deploy")
	assert.Error(t, err)
}
