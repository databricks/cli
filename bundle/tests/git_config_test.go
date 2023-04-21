package config_tests

import (
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: end to end loading fails here due to a multiple resources defined error
// repro and see what's going wrong

func TestGitConfigLoadForOriginUrlDefined(t *testing.T) {
	b, err := bundle.Load("./git_config/origin_url_defined")
	require.NoError(t, err)
	assert.Equal(t, "my-branch", b.Config.GitConfig.Branch)
	assert.Equal(t, "0e7a60e693b17f119641d6108f8aa3d2bcfb967d", b.Config.GitConfig.Commit)
	assert.Equal(t, "https://www.foo.com/bar", b.Config.GitConfig.RemoteUrl)
}

func TestGitConfigLoadForUserDefinedGitConfigNotOverridenByEnv(t *testing.T) {
	b, err := bundle.Load("./git_config/user_entered_config")
	require.NoError(t, err)
	assert.Equal(t, "user-entered-branch", b.Config.GitConfig.Branch)
	assert.Equal(t, "user-entered-commit", b.Config.GitConfig.Commit)
	assert.Equal(t, "user-entered-url", b.Config.GitConfig.RemoteUrl)
}

func TestGitConfigNoRepo(t *testing.T) {
	b, err := bundle.Load("./git_config/no_repo")
	require.NoError(t, err)
	assert.Equal(t, "", b.Config.GitConfig.Branch)
	assert.Equal(t, "", b.Config.GitConfig.Commit)
	assert.Equal(t, "", b.Config.GitConfig.RemoteUrl)
}

func TestGitConfigEmptyRepo(t *testing.T) {
	b, err := bundle.Load("./git_config/empty_repo")
	require.NoError(t, err)
	assert.Equal(t, "master", b.Config.GitConfig.Branch)
	assert.Equal(t, "", b.Config.GitConfig.Commit)
	assert.Equal(t, "", b.Config.GitConfig.RemoteUrl)
}

func TestGitConfigWhenCheckedintoBranchWithCommits(t *testing.T) {
	b, err := bundle.Load("./git_config/checked_into_branch_with_commits")
	require.NoError(t, err)
	assert.Equal(t, "my-first-branch", b.Config.GitConfig.Branch)
	assert.Equal(t, "2b2deba30cff8a946188d8fdc0e3862c118031f8", b.Config.GitConfig.Commit)
	assert.Equal(t, "", b.Config.GitConfig.RemoteUrl)
}

func TestGitConfigWhenCheckedintoCommit(t *testing.T) {
	b, err := bundle.Load("./git_config/checked_into_commit")
	require.NoError(t, err)
	assert.Equal(t, "", b.Config.GitConfig.Branch)
	assert.Equal(t, "85ba9c1bb2c5bd1334db0efee908f1507a400372", b.Config.GitConfig.Commit)
	assert.Equal(t, "", b.Config.GitConfig.RemoteUrl)
}
