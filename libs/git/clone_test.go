package git

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitCloneArgs(t *testing.T) {
	// case: No branch / tag specified. In this case git clones the default branch
	assert.Equal(t, []string{"clone", "abc", "/def", "--no-tags", "--depth=1"}, cloneOptions{
		Reference:     "",
		RepositoryUrl: "abc",
		TargetPath:    "/def",
		Shallow:       true,
	}.args())

	// case: A branch is specified.
	assert.Equal(t, []string{"clone", "abc", "/def", "--no-tags", "--branch", "my-branch", "--depth=1"}, cloneOptions{
		Reference:     "my-branch",
		RepositoryUrl: "abc",
		TargetPath:    "/def",
		Shallow:       true,
	}.args())

	// case: deep cloning
	assert.Equal(t, []string{"clone", "abc", "/def", "--no-tags"}, cloneOptions{
		Reference:     "",
		RepositoryUrl: "abc",
		TargetPath:    "/def",
		Shallow:       false,
	}.args())
}

func TestGitCloneWithGitNotFound(t *testing.T) {
	// We set $PATH here so the git CLI cannot be found by the clone function
	t.Setenv("PATH", "")
	tmpDir := t.TempDir()

	err := Clone(context.Background(), "abc", "", tmpDir)
	assert.ErrorIs(t, err, exec.ErrNotFound)
}
