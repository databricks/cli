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
	assert.Equal(t, "https://github.com/abc/def@main", expandUrl("abc/def@main"))
	assert.Equal(t, "https://github.com/abc/def", expandUrl("abc/def"))

	// case: full path of the repo is specified
	assert.Equal(t, "https://github.com/databricks/cli.git@snapshot", expandUrl("https://github.com/databricks/cli.git@snapshot"))
	assert.Equal(t, "https://github.com/databricks/cli.git", expandUrl("https://github.com/databricks/cli.git"))
	assert.Equal(t, "git@github.com:databricks/cli.git@v1.0.0", expandUrl("git@github.com:databricks/cli.git@v1.0.0"))
	assert.Equal(t, "git@github.com:databricks/cli.git", expandUrl("git@github.com:databricks/cli.git"))

	// case: invalid repository references
	assert.Equal(t, "https://github.com/databricks/", expandUrl(""))
	assert.Equal(t, "https://github.com//", expandUrl("/"))
	assert.Equal(t, "https://github.com//abc/def", expandUrl("/abc/def"))
}

func TestGitParseCloneOptions(t *testing.T) {
	// cases: only repository name is specified
	assert.Equal(t, cloneOptions{
		Reference:     "main",
		RepositoryUrl: "https://github.com/databricks/abc",
		TargetPath:    "foo",
	}, parseCloneOptions("abc@main", "foo"))

	assert.Equal(t, cloneOptions{
		Reference:     "",
		RepositoryUrl: "https://github.com/databricks/abc",
		TargetPath:    "bar",
	}, parseCloneOptions("abc", "bar"))

	assert.Equal(t, cloneOptions{
		Reference:     "v0.0.1",
		RepositoryUrl: "https://github.com/databricks/abc",
		TargetPath:    "bar",
	}, parseCloneOptions("abc@v0.0.1", "bar"))

	// cases: both repository name and organization is specified
	assert.Equal(t, cloneOptions{
		Reference:     "v0.0.1",
		RepositoryUrl: "https://github.com/abc/def",
		TargetPath:    "bar",
	}, parseCloneOptions("abc/def@v0.0.1", "bar"))

	assert.Equal(t, cloneOptions{
		Reference:     "",
		RepositoryUrl: "https://github.com/abc/def",
		TargetPath:    "bar",
	}, parseCloneOptions("abc/def", "bar"))

	// cases: full path of the repository is specified
	assert.Equal(t, cloneOptions{
		Reference:     "",
		RepositoryUrl: "https://github.com/databricks/cli.git",
		TargetPath:    "/bar",
	}, parseCloneOptions("https://github.com/databricks/cli.git@snapshot", "/bar"))

	assert.Equal(t, cloneOptions{
		Reference:     "",
		RepositoryUrl: "https://github.com/databricks/cli.git",
		TargetPath:    "/bar",
	}, parseCloneOptions("https://github.com/databricks/cli.git", "/bar"))

	assert.Equal(t, cloneOptions{
		Reference:     "v1.0.0",
		RepositoryUrl: "git@github.com:databricks/cli.git",
		TargetPath:    "/bar/foo",
	}, parseCloneOptions("git@github.com:databricks/cli.git@v1.0.0", "/bar/foo"))

	assert.Equal(t, cloneOptions{
		Reference:     "",
		RepositoryUrl: "git@github.com:databricks/cli.git",
		TargetPath:    "/bar/foo",
	}, parseCloneOptions("git@github.com:databricks/cli.git", "/bar/foo"))

	assert.Equal(t, cloneOptions{
		Reference:     "",
		RepositoryUrl: "https://github.com/databricks/cli.git",
		TargetPath:    "/bar",
	}, parseCloneOptions("https://github.com/databricks/cli.git@snapshot", "/bar"))
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

	err := Clone(context.Background(), "abc", tmpDir)
	assert.ErrorIs(t, err, exec.ErrNotFound)
}
