package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitConfigEmptyRepo(t *testing.T) {
	l := &configLoader{
		gitDirPath: "./testdata_git/empty_repo",
	}

	branch, err := l.Branch()
	require.NoError(t, err)
	assert.Equal(t, "master", branch)

	commit, err := l.Commit()
	require.NoError(t, err)
	assert.Equal(t, "", commit)

	remoteUrl, err := l.HttpsOrigin()
	require.NoError(t, err)
	assert.Equal(t, "", remoteUrl)
}

func TestGitConfigOriginUrl(t *testing.T) {
	l := &configLoader{
		gitDirPath: "./testdata_git/origin_url",
	}

	branch, err := l.Branch()
	require.NoError(t, err)
	assert.Equal(t, "my-branch", branch)

	commit, err := l.Commit()
	require.NoError(t, err)
	assert.Equal(t, "0e7a60e693b17f119641d6108f8aa3d2bcfb967d", commit)

	remoteUrl, err := l.HttpsOrigin()
	require.NoError(t, err)
	assert.Equal(t, "https://www.foo.com/bar", remoteUrl)

	name, err := l.RepositoryName()
	require.NoError(t, err)
	assert.Equal(t, "bar", name)
}

func TestGitConfigBranchCheckout(t *testing.T) {
	l := &configLoader{
		gitDirPath: "./testdata_git/branch_checkout",
	}

	branch, err := l.Branch()
	require.NoError(t, err)
	assert.Equal(t, "my-branch", branch)

	commit, err := l.Commit()
	require.NoError(t, err)
	assert.Equal(t, "2b2deba30cff8a946188d8fdc0e3862c118031f8", commit)

	remoteUrl, err := l.HttpsOrigin()
	require.NoError(t, err)
	assert.Equal(t, "", remoteUrl)
}

func TestGitConfigCommitCheckout(t *testing.T) {
	l := &configLoader{
		gitDirPath: "./testdata_git/commit_checkout",
	}

	branch, err := l.Branch()
	require.NoError(t, err)
	assert.Equal(t, "", branch)

	commit, err := l.Commit()
	require.NoError(t, err)
	assert.Equal(t, "85ba9c1bb2c5bd1334db0efee908f1507a400372", commit)

	remoteUrl, err := l.HttpsOrigin()
	require.NoError(t, err)
	assert.Equal(t, "", remoteUrl)
}
