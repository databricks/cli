package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWorktree(t *testing.T) string {
	var err error

	tmpDir := t.TempDir()

	// Checkout path
	err = os.MkdirAll(filepath.Join(tmpDir, "my_worktree"), os.ModePerm)
	require.NoError(t, err)

	// Main $GIT_COMMON_DIR
	err = os.MkdirAll(filepath.Join(tmpDir, ".git"), os.ModePerm)
	require.NoError(t, err)

	// Worktree $GIT_DIR
	err = os.MkdirAll(filepath.Join(tmpDir, ".git/worktrees/my_worktree"), os.ModePerm)
	require.NoError(t, err)

	return tmpDir
}

func writeGitDir(t *testing.T, dir, content string) {
	err := os.WriteFile(filepath.Join(dir, "my_worktree/.git"), []byte(content), os.ModePerm)
	require.NoError(t, err)
}

func writeGitCommonDir(t *testing.T, dir, content string) {
	err := os.WriteFile(filepath.Join(dir, ".git/worktrees/my_worktree/commondir"), []byte(content), os.ModePerm)
	require.NoError(t, err)
}

func verifyCorrectDirs(t *testing.T, dir string) {
	gitDir, gitCommonDir, err := resolveGitDirs(vfs.MustNew(filepath.Join(dir, "my_worktree")))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".git/worktrees/my_worktree"), gitDir.Native())
	assert.Equal(t, filepath.Join(dir, ".git"), gitCommonDir.Native())
}

func TestWorktreeResolveGitDir(t *testing.T) {
	dir := setupWorktree(t)
	writeGitCommonDir(t, dir, "../..")

	t.Run("relative", func(t *testing.T) {
		writeGitDir(t, dir, "gitdir: "+"../.git/worktrees/my_worktree")
		verifyCorrectDirs(t, dir)
	})

	t.Run("absolute", func(t *testing.T) {
		writeGitDir(t, dir, "gitdir: "+filepath.Join(dir, ".git/worktrees/my_worktree"))
		verifyCorrectDirs(t, dir)
	})

	t.Run("additional spaces", func(t *testing.T) {
		writeGitDir(t, dir, fmt.Sprintf("gitdir:    %s     \n\n\n", "../.git/worktrees/my_worktree"))
		verifyCorrectDirs(t, dir)
	})

	t.Run("empty", func(t *testing.T) {
		writeGitDir(t, dir, "")

		_, _, err := resolveGitDirs(vfs.MustNew(filepath.Join(dir, "my_worktree")))
		assert.ErrorContains(t, err, ` to contain a line with "gitdir: [...]"`)
	})
}

func TestWorktreeResolveCommonDir(t *testing.T) {
	dir := setupWorktree(t)
	writeGitDir(t, dir, "gitdir: "+"../.git/worktrees/my_worktree")

	t.Run("relative", func(t *testing.T) {
		writeGitCommonDir(t, dir, "../..")
		verifyCorrectDirs(t, dir)
	})

	t.Run("absolute", func(t *testing.T) {
		writeGitCommonDir(t, dir, filepath.Join(dir, ".git"))
		verifyCorrectDirs(t, dir)
	})

	t.Run("additional spaces", func(t *testing.T) {
		writeGitCommonDir(t, dir, "    ../..    \n\n\n")
		verifyCorrectDirs(t, dir)
	})

	t.Run("empty", func(t *testing.T) {
		writeGitCommonDir(t, dir, "")

		_, _, err := resolveGitDirs(vfs.MustNew(filepath.Join(dir, "my_worktree")))
		assert.ErrorContains(t, err, `expected "commondir" file in worktree git folder at `)
	})

	t.Run("missing", func(t *testing.T) {
		_, _, err := resolveGitDirs(vfs.MustNew(filepath.Join(dir, "my_worktree")))
		assert.ErrorContains(t, err, `expected "commondir" file in worktree git folder at `)
	})
}
