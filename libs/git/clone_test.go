package git

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitExpandUrl(t *testing.T) {
	// case: only repository name is specified
	assert.Equal(t, "https://github.com/databricks/abc", expandUrl("abc"))

	// case: both org and repo name are specified
	assert.Equal(t, "https://github.com/abc/def", expandUrl("abc/def"))

	// case: full path of the repo is specified
	assert.Equal(t, "https://github.com/databricks/cli.git", expandUrl("https://github.com/databricks/cli.git"))
	assert.Equal(t, "git@github.com:databricks/cli.git", expandUrl("git@github.com:databricks/cli.git"))

	// case: invalid repository references
	assert.Equal(t, "https://github.com/databricks/", expandUrl(""))
	assert.Equal(t, "https://github.com//", expandUrl("/"))
	assert.Equal(t, "https://github.com//abc/def", expandUrl("/abc/def"))
}

func TestGitCloneArgs(t *testing.T) {
	// case: No branch / tag specified. In this case git clones the default branch
	assert.Equal(t, []string{"clone", "abc", "/def", "--depth=1", "--no-tags"}, cloneOptions{
		Reference:     "",
		RepositoryUrl: "abc",
		TargetPath:    "/def",
	}.args())

	// case: A branch is specified.
	assert.Equal(t, []string{"clone", "abc", "/def", "--depth=1", "--no-tags", "--branch", "my-branch"}, cloneOptions{
		Reference:     "my-branch",
		RepositoryUrl: "abc",
		TargetPath:    "/def",
	}.args())
}

func TestGitCloneWithGitNotFound(t *testing.T) {
	// We set $PATH here so the git CLI cannot be found by the clone function
	t.Setenv("PATH", "")
	tmpDir := t.TempDir()

	err := Clone(context.Background(), "abc", "", tmpDir)
	assert.ErrorIs(t, err, exec.ErrNotFound)
}
